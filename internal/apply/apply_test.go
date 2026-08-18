package apply

import (
	"context"
	"strings"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/spec"
)

type fake struct {
	created   mowl.Program
	segments  int
	intervals int
	attached  []int
	deleted   []int
	existing  []mowl.Program
}

func (f *fake) CreateCategory(_ context.Context, _ string) (int, error)     { return 8000, nil }
func (f *fake) MyPrograms(_ context.Context, _ int) ([]mowl.Program, error) { return f.existing, nil }
func (f *fake) CreateProgram(_ context.Context, p mowl.Program) (mowl.Program, error) {
	p.ProgramID = 700
	f.created = p
	return p, nil
}
func (f *fake) CreateSegment(_ context.Context, _ string, _, _ int, _ mowl.SegmentFlags) (int, error) {
	f.segments++
	return 300 + f.segments, nil
}
func (f *fake) SetIntervals(_ context.Context, _ int, ivs []mowl.Interval) error {
	f.intervals += len(ivs)
	return nil
}
func (f *fake) AttachSegments(_ context.Context, _ int, ids []int) error {
	f.attached = ids
	return nil
}
func (f *fake) ProgramTSS(_ context.Context, _ int) (float64, error) { return 74.2, nil }
func (f *fake) DeleteProgram(_ context.Context, id int) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func demoCourse() spec.Course {
	return spec.Course{
		Name: "Ride", Category: "My Rides", Playlist: spec.Playlist{SpotifyID: "sp"},
		Segments: []spec.Segment{
			{Name: "W", Type: "warmup", Tracks: []int{1},
				Intervals: []spec.Interval{{Duration: 200, Cadence: [2]int{80, 85}, Intensity: spec.IntensityValue{From: 45, To: 55}, Position: "seated"}}},
			{Name: "M", Type: "intervals", Tracks: []int{2},
				Intervals: []spec.Interval{{Duration: 180, Cadence: [2]int{90, 95}, Intensity: spec.IntensityValue{From: 70, To: 80}, Position: "standing"}}},
		},
	}
}

func demoPlaylist() mowl.Playlist {
	return mowl.Playlist{PlaylistID: 900}
}

func TestApplyHappyPath(t *testing.T) {
	f := &fake{}
	res, err := Apply(context.Background(), f, demoCourse(), demoPlaylist(), spec.Styles{}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if res.ProgramID != 700 || res.PlaylistID != 900 || res.ServerTSS != 74.2 {
		t.Fatalf("result = %+v", res)
	}
	if f.segments != 2 || f.intervals != 2 || len(f.attached) != 2 || f.created.PlaylistID != 900 {
		t.Fatalf("wiring off: %+v", f)
	}
	if f.attached[0] != 301 || f.attached[1] != 302 {
		t.Fatalf("attach order wrong: %v", f.attached)
	}
	if f.created.Description != "" {
		t.Fatalf("expected empty description for style-less course, got %q", f.created.Description)
	}
}

func TestApplyIdempotentReplace(t *testing.T) {
	f := &fake{existing: []mowl.Program{{ProgramID: 55, Name: "Ride"}}}
	if _, err := Apply(context.Background(), f, demoCourse(), demoPlaylist(), spec.Styles{}, 42); err != nil {
		t.Fatal(err)
	}
	if len(f.deleted) != 1 || f.deleted[0] != 55 {
		t.Fatalf("expected replace-delete of 55, got %v", f.deleted)
	}
}

func TestApplyWritesStyleDescription(t *testing.T) {
	f := &fake{}
	c := demoCourse()
	c.Style = []string{"road_cycling"}
	styles := spec.Styles{"road_cycling": spec.Style{Position: "seated"}}

	if _, err := Apply(context.Background(), f, c, demoPlaylist(), styles, 42); err != nil {
		t.Fatal(err)
	}
	if f.created.Description == "" {
		t.Fatalf("expected non-empty description for course with style tags")
	}
	if !strings.Contains(f.created.Description, "road_cycling") {
		t.Fatalf("expected description to mention style tag, got %q", f.created.Description)
	}
}
