package mowl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/config"
)

func TestAuthenticateTwoStep(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/Authentication/Authenticate/"):
			w.Write([]byte(`{"Data":"ticket-1","Error":null}`))
		case strings.HasPrefix(r.URL.Path, "/v1/Authentication/Ticket/"):
			if !strings.Contains(r.URL.Path, "ticket-1") {
				t.Errorf("ticket not in path: %s", r.URL.Path)
			}
			w.Write([]byte(`{"Data":"session-tok","Error":null}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	cfg := config.Config{APIBase: srv.URL, ClientVersion: "8.8.2",
		AppPublicKey: "PUB", AppPrivateKey: "PRIV"}
	tok, err := Authenticate(context.Background(), cfg, "a@b.c", "pw", srv.Client())
	if err != nil || tok != "session-tok" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
}
