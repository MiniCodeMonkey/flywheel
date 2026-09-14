package spec

import "gopkg.in/yaml.v3"

type Course struct {
	Name     string   `yaml:"name"`
	Category string   `yaml:"category"`
	Activity string   `yaml:"activity"`
	Targets  Targets  `yaml:"targets"`
	Style    []string `yaml:"style"`
	Playlist Playlist `yaml:"playlist"`
	// Scaffold records the options that produced this course, so the ride can
	// be regenerated exactly rather than reconstructed by memory. Absent on a
	// hand-written course.
	Scaffold *Scaffold `yaml:"scaffold,omitempty"`
	Segments []Segment `yaml:"segments"`
}

// Scaffold is the provenance of a generated course: everything `flywheel
// scaffold` was told, so the same file can be rebuilt from it.
type Scaffold struct {
	TSS                int      `yaml:"tss"`
	MinSection         int      `yaml:"min_section"`
	MaxSection         int      `yaml:"max_section"`
	EndHot             bool     `yaml:"end_hot"`
	ACC                float64  `yaml:"acc"`
	MaxStanding        int      `yaml:"max_standing"`
	StandingCadenceMin int      `yaml:"standing_cadence_min"`
	StandingCadenceMax int      `yaml:"standing_cadence_max"`
	Segments           []string `yaml:"segments"`
}

type Targets struct {
	DurationMin int `yaml:"duration_min"`
	TSS         int `yaml:"tss"`
}

type Playlist struct {
	SpotifyID string `yaml:"spotify_id"`
	// CrossfadeSec is the Spotify crossfade in seconds; nil means the
	// MOWL-required default. See Crossfade.
	CrossfadeSec *int `yaml:"crossfade_sec,omitempty"`
}

type Segment struct {
	Name      string     `yaml:"name"`
	Type      string     `yaml:"type"`
	Tracks    []int      `yaml:"tracks"`
	Style     []string   `yaml:"style"`
	Intervals []Interval `yaml:"intervals"`
}

type Interval struct {
	Duration  int            `yaml:"duration"`
	Cadence   [2]int         `yaml:"cadence"`
	Intensity IntensityValue `yaml:"intensity"`
	Position  string         `yaml:"position"`
	// Cycle is the interval kind: empty or "sb" for a normal block, "acc" for
	// an acceleration burst, "free" for free tempo.
	Cycle string `yaml:"cycle,omitempty"`
}

// IntensityValue accepts a scalar (steady) or a [from,to] pair (ramp).
type IntensityValue struct{ From, To int }

func (v *IntensityValue) UnmarshalYAML(n *yaml.Node) error {
	var scalar int
	if err := n.Decode(&scalar); err == nil {
		v.From, v.To = scalar, scalar
		return nil
	}
	var pair []int
	if err := n.Decode(&pair); err != nil {
		return err
	}
	if len(pair) != 2 {
		return errShape
	}
	v.From, v.To = pair[0], pair[1]
	return nil
}

var errShape = &yamlErr{"intensity must be a number or [from,to]"}

type yamlErr struct{ msg string }

func (e *yamlErr) Error() string { return e.msg }

func ParseCourse(b []byte) (Course, error) {
	var c Course
	err := yaml.Unmarshal(b, &c)
	return c, err
}
