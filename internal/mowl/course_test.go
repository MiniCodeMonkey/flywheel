// internal/mowl/course_test.go
package mowl

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/config"
)

func TestSetIntervalsSendsReplaceExisting(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.Write([]byte(`{"Data":[{"IntervalID":1}],"Error":null}`))
	}))
	defer srv.Close()
	c := New(config.Config{APIBase: srv.URL, ClientVersion: "8.8.2"}, "t", srv.Client())
	err := c.SetIntervals(context.Background(), 5, []Interval{
		{Duration: 120, RPMFrom: 80, RPMTo: 90, Intensity: 55, FTPFrom: 50, FTPTo: 60, PositionTypeID: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["ReplaceExisting"] != true {
		t.Fatalf("ReplaceExisting not set: %v", gotBody)
	}
	if _, ok := gotBody["Intervals"].([]any); !ok {
		t.Fatalf("Intervals missing: %v", gotBody)
	}
}

func TestCreateCategoryReturnsID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Data":{"ProgramCategoryID":8654,"Name":"X"},"Error":null}`))
	}))
	defer srv.Close()
	c := New(config.Config{APIBase: srv.URL, ClientVersion: "8.8.2"}, "t", srv.Client())
	id, err := c.CreateCategory(context.Background(), "X")
	if err != nil || id != 8654 {
		t.Fatalf("id=%d err=%v", id, err)
	}
}
