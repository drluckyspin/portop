package cli

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/padovanl/portop/internal/app"
	"github.com/padovanl/portop/internal/procinfo"
	"github.com/padovanl/portop/internal/scanner"
)

//go:embed web/index.html
var webPage string

//go:embed web/app.css
var webCSS string

//go:embed web/app.js
var webJS string

func webHandler(collect func(context.Context, app.Options) ([]app.Row, error), filter string, listenOnly bool, opts app.Options, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, webPage)
	})
	mux.HandleFunc("GET /app.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = io.WriteString(w, webCSS)
	})
	mux.HandleFunc("GET /app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = io.WriteString(w, webJS)
	})
	mux.HandleFunc("GET /api/ports", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		rows, err := collect(ctx, opts)
		if err != nil {
			http.Error(w, "port scan failed", http.StatusInternalServerError)
			return
		}
		out := make([]jsonRow, 0, len(rows))
		for _, row := range rows {
			if listenOnly && row.State != scanner.StateListen {
				continue
			}
			if !row.Matches(filter) {
				continue
			}
			out = append(out, toJSONRow(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	})
	currentProcess := func(r *http.Request, pid int) (bool, error) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		rows, err := collect(ctx, opts)
		if err != nil {
			return false, err
		}
		for _, row := range rows {
			if row.PID == pid {
				return true, nil
			}
		}
		return false, nil
	}
	processRequest := func(w http.ResponseWriter, r *http.Request) (int, bool) {
		if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Portop-Token")), []byte(token)) != 1 {
			http.Error(w, "action token required; open the URL printed by portop", http.StatusForbidden)
			return 0, false
		}
		pid, err := strconv.Atoi(r.PathValue("pid"))
		if err != nil || pid <= 0 {
			http.Error(w, "invalid PID", http.StatusBadRequest)
			return 0, false
		}
		visible, err := currentProcess(r, pid)
		if err != nil {
			http.Error(w, "port scan failed", http.StatusInternalServerError)
			return 0, false
		}
		if !visible {
			http.Error(w, "process no longer owns a visible socket", http.StatusNotFound)
			return 0, false
		}
		return pid, true
	}
	mux.HandleFunc("GET /api/process/{pid}", func(w http.ResponseWriter, r *http.Request) {
		pid, ok := processRequest(w, r)
		if !ok {
			return
		}
		info, err := procinfo.Load(pid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(info)
	})
	mux.HandleFunc("POST /api/process/{pid}/signal", func(w http.ResponseWriter, r *http.Request) {
		pid, ok := processRequest(w, r)
		if !ok {
			return
		}
		var action struct {
			StartTime string `json:"start_time"`
			Force     bool   `json:"force"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&action); err != nil {
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
		}
		if err := signalProcess(pid, action.StartTime, action.Force); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if parsed, _, err := net.SplitHostPort(host); err == nil {
			host = parsed
		}
		if host != "localhost" && !net.ParseIP(host).IsLoopback() {
			http.Error(w, "invalid host", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; script-src 'self'; connect-src 'self'")
		mux.ServeHTTP(w, r)
	})
}

func webAddress(address string, port int, addrSet, portSet bool) (string, error) {
	if addrSet && portSet {
		return "", fmt.Errorf("use --web-port or --web-addr, not both")
	}
	if portSet {
		if port < 1 || port > 65535 {
			return "", fmt.Errorf("--web-port must be between 1 and 65535")
		}
		address = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	host, rawPort, err := net.SplitHostPort(address)
	if err != nil || (host != "localhost" && !net.ParseIP(host).IsLoopback()) {
		return "", fmt.Errorf("--web-addr must be a loopback host:port")
	}
	n, err := strconv.Atoi(rawPort)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("--web-addr port must be between 1 and 65535")
	}
	return address, nil
}

func runWeb(stdout, stderr io.Writer, address, filter string, listenOnly bool, opts app.Options) int {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintln(stderr, "portop: web: "+err.Error())
		return 1
	}
	defer listener.Close()
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		fmt.Fprintln(stderr, "portop: web: could not create action token")
		return 1
	}
	token := hex.EncodeToString(secret)
	fmt.Fprintf(stdout, "portop web: http://%s/#token=%s\n", listener.Addr(), token)
	collector := app.NewCollector()
	var mu sync.Mutex
	server := &http.Server{
		Handler: webHandler(func(ctx context.Context, opts app.Options) ([]app.Row, error) {
			mu.Lock()
			defer mu.Unlock()
			return collector.Collect(ctx, opts)
		}, filter, listenOnly, opts, token),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(stderr, "portop: web: "+err.Error())
		return 1
	}
	return 0
}
