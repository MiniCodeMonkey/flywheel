package spec

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Style struct {
	Position       string `yaml:"position"`
	Cadence        [2]int `yaml:"cadence"`
	MinIntervalSec int    `yaml:"min_interval_sec"`
	IntensitySwing string `yaml:"intensity_swing"`
}

type Styles map[string]Style

func ParseStyles(b []byte) (Styles, error) {
	var s Styles
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func (s Styles) Resolve(tags []string) Style {
	var out Style
	for _, tag := range tags {
		st, ok := s[tag]
		if !ok {
			continue
		}
		if st.Position != "" {
			out.Position = st.Position
		}
		if st.Cadence != [2]int{} {
			out.Cadence = st.Cadence
		}
		if st.MinIntervalSec != 0 {
			out.MinIntervalSec = st.MinIntervalSec
		}
		if st.IntensitySwing != "" {
			out.IntensitySwing = st.IntensitySwing
		}
	}
	return out
}

func (s Styles) Describe(tags []string) string {
	var known []string
	for _, t := range tags {
		if _, ok := s[t]; ok {
			known = append(known, t)
		} else {
			known = append(known, t+" (advisory)")
		}
	}
	if len(known) == 0 {
		return ""
	}
	return fmt.Sprintf("Style: %s", strings.Join(known, ", "))
}
