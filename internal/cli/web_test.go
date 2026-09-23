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

func TestWebAddressMustBeLoopback(t *testing.T) {
	for _, address := range []string{"0.0.0.0:8088", "example.com:8088", ":8088"} {
		var out, errOut bytes.Buffer
		if code := Run([]string{"--web", "--web-addr", address}, &out, &errOut); code != 2 {
			t.Errorf("%s: exit %d", address, code)
		}
	}
}
