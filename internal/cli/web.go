package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/padovanl/portop/internal/app"
	"github.com/padovanl/portop/internal/scanner"
)

// webHandler is also used by tests with a supplied collector.
func webHandler(collect func(context.Context, app.Options) ([]app.Row, error), filter string, listenOnly bool, opts app.Options) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, webPage)
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
			if filter != "" && !matchesFilter(row, filter) {
				continue
			}
			out = append(out, toJSONRow(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
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
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'")
		mux.ServeHTTP(w, r)
	})
}

func runWeb(stdout, stderr io.Writer, address, filter string, listenOnly bool, opts app.Options) int {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintln(stderr, "portop: web: "+err.Error())
		return 1
	}
	defer listener.Close()
	fmt.Fprintf(stdout, "portop web: http://%s\n", listener.Addr())
	collector := app.NewCollector
	server := &http.Server{
		Handler: webHandler(func(ctx context.Context, opts app.Options) ([]app.Row, error) {
			return collector().Collect(ctx, opts)
		}, filter, listenOnly, opts),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(stderr, "portop: web: "+err.Error())
		return 1
	}
	return 0
}

const webPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>portop · ports</title><style>
:root{color-scheme:dark;font-family:system-ui,sans-serif;background:#111827;color:#e5e7eb}
body{max-width:1150px;margin:0 auto;padding:2rem}header{display:flex;align-items:center;justify-content:space-between;gap:1rem;flex-wrap:wrap}
h1{margin:0;color:#6ee7b7}p{color:#9ca3af}input{background:#1f2937;color:inherit;border:1px solid #4b5563;border-radius:6px;padding:.7rem;font:inherit;min-width:16rem}
.table{overflow-x:auto}table{width:100%;border-collapse:collapse;margin-top:1.5rem}th{text-align:left;color:#9ca3af;font-size:.8rem;text-transform:uppercase}td,th{padding:.8rem;border-bottom:1px solid #374151}tbody tr:hover{background:#1f2937}code{color:#a7f3d0}#status{min-height:1.5em}
</style></head><body><header><div><h1>portop</h1><p>Live local ports · refreshes every 2 seconds</p></div><input id="filter" type="search" placeholder="Filter port, process, PID" aria-label="Filter ports"></header><p id="status" role="status"></p><div class="table"><table><thead><tr><th>Protocol</th><th>Local</th><th>State</th><th>PID</th><th>Process</th><th>CPU</th><th>Service / container</th></tr></thead><tbody id="rows"></tbody></table></div><script>
const body=document.getElementById('rows'),status=document.getElementById('status'),filter=document.getElementById('filter');let rows=[];
function render(){const q=filter.value.toLowerCase();body.replaceChildren();let shown=0;for(const row of rows){if(q&&!String(row.local_port).includes(q)&&!String(row.pid||'').includes(q)&&!(row.process||'').toLowerCase().includes(q))continue;shown++;const tr=document.createElement('tr');for(const value of [row.protocol,row.local_address+':'+row.local_port,row.state,row.pid||'—',row.process||'—',row.cpu_percent.toFixed(1)+'%',row.systemd_unit||row.container||'—']){const td=document.createElement('td');td.textContent=value;tr.append(td)}body.append(tr)}status.textContent=shown+' sockets visible';}
async function refresh(){try{const res=await fetch('/api/ports',{cache:'no-store'});if(!res.ok)throw Error('scan failed');rows=await res.json();render()}catch(e){status.textContent='Unable to refresh ports: '+e.message}}
filter.addEventListener('input',render);refresh();setInterval(refresh,2000);
</script></body></html>`
