package spec

import (
	"os"
	"testing"
)

func TestResolveMergesTags(t *testing.T) {
	b, _ := os.ReadFile("testdata/styles.yaml")
	s, err := ParseStyles(b)
	if err != nil {
		t.Fatal(err)
	}
	got := s.Resolve([]string{"road_cycling", "punchy"})
	if got.Position != "seated" { // from road_cycling, punchy doesn't set it
		t.Fatalf("position = %q", got.Position)
	}
	if got.MinIntervalSec != 30 { // punchy overrides
		t.Fatalf("min = %d", got.MinIntervalSec)
	}
	if got.IntensitySwing != "large" {
		t.Fatalf("swing = %q", got.IntensitySwing)
	}
}

func TestResolveUnknownTagNoError(t *testing.T) {
	s := Styles{}
	got := s.Resolve([]string{"nonexistent"})
	if got.Position != "" {
		t.Fatalf("expected zero style, got %+v", got)
	}
}
