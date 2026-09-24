package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/padovanl/portop/internal/app"
	"github.com/padovanl/portop/internal/scanner"
)

func TestWebHandler(t *testing.T) {
	rows := []app.Row{
		{Protocol: scanner.TCP, LocalAddr: net.ParseIP("127.0.0.1"), LocalPort: 8080, State: scanner.StateListen, PID: 42, ProcessName: "<script>"},
		{Protocol: scanner.TCP, LocalAddr: net.ParseIP("127.0.0.1"), LocalPort: 8081, State: scanner.StateEstablished},
	}
	h := webHandler(func(context.Context, app.Options) ([]app.Row, error) { return rows, nil }, "8080", true, app.Options{})
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/ports", nil)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("response: %d, headers: %v", response.Code, response.Header())
	}
	var got []jsonRow
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || len(got) != 1 || got[0].LocalPort != 8080 {
		t.Fatalf("rows: %+v, error: %v", got, err)
	}
	for _, path := range []string{"/", "/app.css", "/app.js"} {
		asset := httptest.NewRecorder()
		h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "http://127.0.0.1"+path, nil))
		if asset.Code != http.StatusOK || asset.Body.Len() == 0 || strings.Contains(asset.Header().Get("Content-Security-Policy"), "unsafe-inline") {
			t.Errorf("asset %s: status %d, headers %v", path, asset.Code, asset.Header())
		}
	}
	request.Host = "attacker.example"
	response = httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unexpected host status: %d", response.Code)
	}
}

func TestWebAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		port    int
		addrSet bool
		portSet bool
		want    string
		wantErr bool
	}{
		{name: "default", address: "127.0.0.1:8088", port: 8088, want: "127.0.0.1:8088"},
		{name: "custom port", address: "127.0.0.1:8088", port: 9090, portSet: true, want: "127.0.0.1:9090"},
		{name: "IPv6 loopback", address: "[::1]:9090", addrSet: true, want: "[::1]:9090"},
		{name: "remote address", address: "0.0.0.0:8088", addrSet: true, wantErr: true},
		{name: "zero port", address: "127.0.0.1:8088", portSet: true, wantErr: true},
		{name: "large port", address: "127.0.0.1:8088", port: 65536, portSet: true, wantErr: true},
		{name: "conflicting flags", address: "127.0.0.1:9090", port: 9000, addrSet: true, portSet: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := webAddress(tt.address, tt.port, tt.addrSet, tt.portSet)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("webAddress() = %q, %v; want %q, error=%v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestWebFlagsRejectInvalidAddress(t *testing.T) {
	for _, args := range [][]string{
		{"--web", "--web-addr", "0.0.0.0:8088"},
		{"--web", "--web-port", "0"},
		{"--web", "--web-port", "9090", "--web-addr", "127.0.0.1:9090"},
	} {
		var out, errOut bytes.Buffer
		if code := Run(args, &out, &errOut); code != 2 {
			t.Errorf("%v: exit %d, stderr %q", args, code, errOut.String())
		}
	}
}
