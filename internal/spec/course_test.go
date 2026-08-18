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

func TestParseCourseRejectsBadIntensity(t *testing.T) {
	tests := []struct {
		name    string
		yamlStr string
	}{
		{
			name: "3-element intensity array",
			yamlStr: `
name: "Test"
category: "Test"
activity: cycling
targets: { duration_min: 55, tss: 75 }
segments:
  - name: Test
    type: test
    intervals:
      - { duration: 60, cadence: [80,85], intensity: [1,2,3], position: seated }
`,
		},
		{
			name: "1-element intensity array",
			yamlStr: `
name: "Test"
category: "Test"
activity: cycling
targets: { duration_min: 55, tss: 75 }
segments:
  - name: Test
    type: test
    intervals:
      - { duration: 60, cadence: [80,85], intensity: [1], position: seated }
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCourse([]byte(tt.yamlStr))
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}
