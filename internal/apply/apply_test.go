package apply

import (
	"context"
	"strings"
	"testing"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/spec"
)

type fake struct {
	created      mowl.Program
	segments     int
	intervals    int
	attached     []int
	deleted      []int
	existing     []mowl.Program
	existingCats []mowl.Category
	createdCat   bool
	setIntervals [][]mowl.Interval
}

func (f *fake) CreateCategory(_ context.Context, _ string) (int, error) {
	f.createdCat = true
	return 8000, nil
}
func (f *fake) MyCategories(_ context.Context, _ int) ([]mowl.Category, error) {
	return f.existingCats, nil
}
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
	f.setIntervals = append(f.setIntervals, ivs)
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

func TestApplyReusesExistingCategory(t *testing.T) {
	f := &fake{existingCats: []mowl.Category{{ProgramCategoryID: 8000, Name: "My Rides"}}}
	// demoCourse uses category "My Rides"; a matching category exists, so
	// CreateCategory must NOT be called (createCat stays false).
	res, err := Apply(context.Background(), f, demoCourse(), mowl.Playlist{PlaylistID: 900}, spec.Styles{}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if f.createdCat {
		t.Fatal("CreateCategory was called despite an existing same-named category")
	}
	if res.ProgramID == 0 {
		t.Fatalf("bad result: %+v", res)
	}
}

func TestApplySetsScaleCoggan(t *testing.T) {
	f := &fake{}
	c := demoCourse() // Main has an interval at 70-80% FTP (avg 75 -> zone 2) and 88 -> zone 3? check demо
	if _, err := Apply(context.Background(), f, c, mowl.Playlist{PlaylistID: 900}, spec.Styles{}, 42); err != nil {
		t.Fatal(err)
	}
	// Warmup interval is 45-55% FTP (avg 50 -> zone 1); Main 70-80 (avg75 -> zone 2).
	if len(f.setIntervals) == 0 {
		t.Fatal("no intervals captured")
	}
	// every interval must carry a non-zero ScaleCoggan
	for _, ivs := range f.setIntervals {
		for _, iv := range ivs {
			if iv.ScaleCoggan < 1 || iv.ScaleCoggan > 7 {
				t.Fatalf("interval ScaleCoggan out of range: %+v", iv)
			}
		}
	}
}
