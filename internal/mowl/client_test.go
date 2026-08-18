package mowl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/config"
)

func TestDoUnwrapsEnvelopeAndSendsHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "tok" ||
			r.Header.Get("itc-client-os-family") != "MacOS" ||
			r.Header.Get("itc-client-version") != "8.8.2" {
			t.Errorf("missing headers: %v", r.Header)
		}
		w.Write([]byte(`{"Data":{"UserID":42},"Error":null,"Stack":null}`))
	}))
	defer srv.Close()
	cfg := config.Config{APIBase: srv.URL, ClientVersion: "8.8.2"}
	c := New(cfg, "tok", srv.Client())
	var out struct{ UserID int }
	if err := c.do(context.Background(), "GET", "/v1/Users/Me", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.UserID != 42 {
		t.Fatalf("UserID = %d", out.UserID)
	}
}

func TestDoRefreshesOn401(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") == "stale" {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`{"Data":true,"Error":null}`))
	}))
	defer srv.Close()
	c := New(config.Config{APIBase: srv.URL, ClientVersion: "8.8.2"}, "stale", srv.Client())
	c.SetReauth(func() (string, error) { return "fresh", nil })
	var out bool
	if err := c.do(context.Background(), "GET", "/v1/x", nil, &out); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || !out {
		t.Fatalf("calls=%d out=%v (want retry once)", calls, out)
	}
}

func TestDoReturnsEnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Data":null,"Error":{"Message":"nope"}}`))
	}))
	defer srv.Close()
	c := New(config.Config{APIBase: srv.URL, ClientVersion: "8.8.2"}, "t", srv.Client())
	err := c.do(context.Background(), "GET", "/v1/x", nil, nil)
	if err == nil || err.Error() != "mowl: nope" {
		t.Fatalf("err = %v", err)
	}
}
