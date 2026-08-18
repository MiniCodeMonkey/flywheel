// internal/cli/client_test.go
package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/config"
)

func TestNewClientLoadsConfigAndSetsReauth(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	c, cfg, err := newClient(context.Background())
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if cfg.APIBase == "" {
		t.Fatal("expected default APIBase to be set")
	}
}

func TestNewClientReauthFailsWithoutCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("MOWL_PASSWORD", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := config.Config{APIBase: srv.URL, ClientVersion: "8.8.2"}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	c, _, err := newClient(context.Background())
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	if _, err := c.Me(context.Background()); err == nil {
		t.Fatal("expected error on 401 with no reauth credentials")
	} else if !strings.Contains(err.Error(), "token expired") {
		t.Fatalf("unexpected error: %v", err)
	}
}
