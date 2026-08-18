package spec

import (
	"os"
	"testing"
)

func TestParseCourse(t *testing.T) {
	b, _ := os.ReadFile("testdata/course.yaml")
	c, err := ParseCourse(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "Heavy Rock 55" || c.Targets.TSS != 75 || len(c.Segments) != 2 {
		t.Fatalf("bad parse: %+v", c)
	}
	// scalar intensity
	if got := c.Segments[0].Intervals[0].Intensity; got.From != 45 || got.To != 45 {
		t.Fatalf("scalar intensity = %+v", got)
	}
	// pair intensity
	if got := c.Segments[1].Intervals[0].Intensity; got.From != 70 || got.To != 80 {
		t.Fatalf("pair intensity = %+v", got)
	}
	if c.Segments[0].Intervals[0].Cadence != [2]int{80, 85} {
		t.Fatalf("cadence = %v", c.Segments[0].Intervals[0].Cadence)
	}
}
