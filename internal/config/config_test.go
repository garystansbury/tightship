package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestExampleFileIsValid(t *testing.T) {
	raw, err := os.ReadFile("../../deploy/config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(raw)
	if err != nil {
		t.Fatalf("the shipped example must validate: %v", err)
	}
	if !c.ModuleEnabled("helpdesk") {
		t.Error("the example should enable helpdesk")
	}
	if c.Dev.AllowDebugIdentity {
		t.Error("the example must not ship with the debug identity header allowed")
	}
}

func TestValidateNamesEveryProblemAtOnce(t *testing.T) {
	_, err := Parse([]byte("org: {}\n"))
	if err == nil {
		t.Fatal("an empty file must not validate")
	}
	for _, want := range []string{"org.name", "server.public_url", "database.host", "domains.staff", "secrets.master_key", "modules"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %s; got: %v", want, err)
		}
	}
}

func TestUnknownKeyIsAnError(t *testing.T) {
	_, err := Parse([]byte("org:\n  name: X\n  colour: blue\n"))
	if err == nil || !strings.Contains(err.Error(), "colour") {
		t.Fatalf("a misspelt key must fail loudly, got: %v", err)
	}
}

func TestPublicURLMustBeHTTPS(t *testing.T) {
	_, err := Parse([]byte("server:\n  public_url: http://x\n"))
	if err == nil || !strings.Contains(err.Error(), "https://") {
		t.Fatalf("want an https complaint, got: %v", err)
	}
}

// Go's ParseDuration stops at hours, so a fortnight would have to be written "336h". This is a
// file an operator edits, and a setting nobody can read at a glance is one that gets copied wrong.
func TestParseDurationUnderstandsDaysAndWeeks(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want time.Duration
	}{
		{"30m", 30 * time.Minute},
		{"8h", 8 * time.Hour},
		{"1d", 24 * time.Hour},
		{"14d", 14 * 24 * time.Hour},
		{"2w", 14 * 24 * time.Hour},
		{"1d12h", 36 * time.Hour},
		{"1w2d", 9 * 24 * time.Hour},
		{"500ms", 500 * time.Millisecond},
	} {
		got, err := ParseDuration(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// Strictness applies to durations too: a value that is not one must be an error rather than a
// silent zero, which would read as "no timeout".
func TestParseDurationRejectsNonsense(t *testing.T) {
	for _, bad := range []string{"", "soon", "14", "d", "14y", "-", "8 hours"} {
		if got, err := ParseDuration(bad); err == nil {
			t.Errorf("accepted %q as %v", bad, got)
		}
	}
}

// The whole point is that the deployment file can say 14d.
func TestSessionDurationsLoadFromYAML(t *testing.T) {
	c, err := Parse([]byte(`
org: {name: Example}
server: {public_url: "https://app.example.org"}
database: {host: h, name: n, user: u}
domains: {staff: example.org}
secrets: {master_key_env: K}
modules: [core]
sessions:
  idle_timeout: 90m
  absolute_lifetime: 2w
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Sessions.IdleTimeout.Std(); got != 90*time.Minute {
		t.Errorf("idle_timeout = %v, want 90m", got)
	}
	if got := c.Sessions.AbsoluteLifetime.Std(); got != 14*24*time.Hour {
		t.Errorf("absolute_lifetime = %v, want 336h", got)
	}
}

// An idle window longer than the absolute lifetime is a contradiction, not a policy: the absolute
// limit always fires first and the idle setting silently never applies.
func TestIdleTimeoutMayNotExceedAbsoluteLifetime(t *testing.T) {
	_, err := Parse([]byte(`
org: {name: Example}
server: {public_url: "https://app.example.org"}
database: {host: h, name: n, user: u}
domains: {staff: example.org}
secrets: {master_key_env: K}
modules: [core]
sessions: {idle_timeout: 30d, absolute_lifetime: 1d}
`))
	if err == nil || !strings.Contains(err.Error(), "can never take effect") {
		t.Errorf("contradictory session limits accepted, err = %v", err)
	}
}

// The __Host- prefix is browser-enforced and is what stops a sibling subdomain setting a session
// cookie this application would trust. Renaming away from it silently loses that.
func TestCookieNameMustKeepTheHostPrefix(t *testing.T) {
	_, err := Parse([]byte(`
org: {name: Example}
server: {public_url: "https://app.example.org"}
database: {host: h, name: n, user: u}
domains: {staff: example.org}
secrets: {master_key_env: K}
modules: [core]
sessions: {cookie_name: tightship_session}
`))
	if err == nil || !strings.Contains(err.Error(), "__Host-") {
		t.Errorf("a cookie name without the prefix was accepted, err = %v", err)
	}
}
