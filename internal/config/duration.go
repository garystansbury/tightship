package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that also understands days and weeks.
//
// Go's ParseDuration stops at hours, so a fortnight has to be written "336h". This is a file an
// operator edits to answer a question like "how long may a session live", and the honest answer
// is in days. A setting nobody can read at a glance is a setting that gets copied wrong.
type Duration time.Duration

// Std returns the standard library value.
func (d Duration) Std() time.Duration { return time.Duration(d) }

func (d Duration) String() string { return time.Duration(d).String() }

// UnmarshalYAML accepts everything time.ParseDuration does, plus a `d` (day) or `w` (week)
// suffix on the leading component: "14d", "2w", "36h", "90m", and "1d12h".
func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	var raw string
	if err := n.Decode(&raw); err != nil {
		return fmt.Errorf("must be a duration such as 30m, 8h or 14d")
	}
	parsed, err := ParseDuration(raw)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// ParseDuration is time.ParseDuration extended with days and weeks. Exposed so the installer and
// tests parse durations exactly as the deployment file does.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var total time.Duration
	rest := s
	// Consume leading <number><d|w> components, then hand whatever is left to the standard
	// parser. Splitting this way means "1d12h" works and the standard units keep their exact
	// existing meaning rather than being reimplemented here.
	for {
		i := 0
		for i < len(rest) && (rest[i] >= '0' && rest[i] <= '9') {
			i++
		}
		if i == 0 || i >= len(rest) {
			break
		}
		unit := rest[i]
		if unit != 'd' && unit != 'w' {
			break
		}
		n, err := strconv.Atoi(rest[:i])
		if err != nil {
			return 0, fmt.Errorf("%q is not a duration", s)
		}
		per := 24 * time.Hour
		if unit == 'w' {
			per = 7 * 24 * time.Hour
		}
		total += time.Duration(n) * per
		rest = rest[i+1:]
	}
	if rest != "" {
		tail, err := time.ParseDuration(rest)
		if err != nil {
			return 0, fmt.Errorf("%q is not a duration (use units such as 30m, 8h, 14d, 2w)", s)
		}
		total += tail
	}
	return total, nil
}
