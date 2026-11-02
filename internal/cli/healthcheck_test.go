package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthURL(t *testing.T) {
	for _, tc := range []struct {
		addr, want string
		wantErr    bool
	}{
		{addr: ":8080", want: "http://127.0.0.1:8080/api/health"},
		{addr: "0.0.0.0:9000", want: "http://127.0.0.1:9000/api/health"},
		{addr: "[::]:8080", want: "http://127.0.0.1:8080/api/health"},
		{addr: "127.0.0.1:8080", want: "http://127.0.0.1:8080/api/health"},
		{addr: "[::1]:8080", want: "http://[::1]:8080/api/health"},
		{addr: "sans-port", wantErr: true},
	} {
		got, err := healthURL(tc.addr)
		if tc.wantErr {
			if err == nil {
				t.Errorf("healthURL(%q) : erreur attendue, obtenu %q", tc.addr, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("healthURL(%q) = %q, %v ; attendu %q", tc.addr, got, err, tc.want)
		}
	}
}

func TestHealthcheck(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
	}))
	url := srv.URL + "/api/health"

	var stderr bytes.Buffer
	if err := Healthcheck(context.Background(), url, &stderr); err != nil {
		t.Errorf("instance saine : %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("sortie inattendue en cas de succès : %q", stderr.String())
	}

	status = http.StatusServiceUnavailable
	err := Healthcheck(context.Background(), url, &stderr)
	if err == nil || err.Error() != "instance en mauvaise santé : statut HTTP 503" {
		t.Errorf("instance en échec : %v", err)
	}

	srv.Close()
	err = Healthcheck(context.Background(), url, &stderr)
	if err == nil || !strings.HasPrefix(err.Error(), "instance injoignable : ") {
		t.Errorf("instance arrêtée : %v", err)
	}
}
