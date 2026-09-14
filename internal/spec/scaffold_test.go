package spec

import "testing"

func TestCourseCarriesItsScaffoldOptions(t *testing.T) {
	c, err := ParseCourse([]byte(`
name: "X"
category: "My Rides"
activity: cycling
targets: { duration_min: 50, tss: 70 }
playlist:
  spotify_id: "abc"
  crossfade_sec: 9
scaffold:
  tss: 71
  min_section: 15
  max_section: 120
  acc: 0.3
  segments: ["Work:intervals:1-4", ":recovery:5"]
segments:
  - name: "Work"
    type: intervals
    tracks: [1]
    intervals:
      - { duration: 60, cadence: [80, 81], intensity: [76, 90], position: seated }
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Scaffold == nil {
		t.Fatal("scaffold block was dropped")
	}
	if c.Scaffold.TSS != 71 || c.Scaffold.MaxSection != 120 || c.Scaffold.ACC != 0.3 {
		t.Errorf("scaffold options not parsed: %+v", c.Scaffold)
	}
	if len(c.Scaffold.Segments) != 2 || c.Scaffold.Segments[0] != "Work:intervals:1-4" {
		t.Errorf("segment specs not parsed: %v", c.Scaffold.Segments)
	}
}

func TestCourseWithoutScaffoldBlockStillParses(t *testing.T) {
	c, err := ParseCourse([]byte(`
name: "X"
activity: cycling
playlist: { spotify_id: "abc" }
segments:
  - name: "W"
    type: intervals
    tracks: [1]
    intervals:
      - { duration: 60, cadence: [80, 81], intensity: [76, 90], position: seated }
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Scaffold != nil {
		t.Errorf("expected no scaffold block, got %+v", c.Scaffold)
	}
}
