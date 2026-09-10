package simulate

import "testing"

func TestReportIsDeterministic(t *testing.T) {
	tracks := []Track{
		trk(1, 120, Section{Duration: 40, Loudness: -12}, Section{Duration: 40, Loudness: -3}, Section{Duration: 40, Loudness: -9}),
		trk(2, 120, Section{Duration: 60, Loudness: -10}, Section{Duration: 60, Loudness: -4}),
	}
	first := Run(Build(course(40, 40, 40, 55, 56), tracks, 9), 3).Text(3)
	for i := 0; i < 20; i++ {
		if got := Run(Build(course(40, 40, 40, 55, 56), tracks, 9), 3).Text(3); got != first {
			t.Fatalf("run %d differs from the first:\n%s\nvs\n%s", i, got, first)
		}
	}
}
