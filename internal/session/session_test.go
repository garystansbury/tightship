package session

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A token has to be unguessable, so the only property worth asserting is that it comes out of the
// CSPRNG at full width and never repeats. 32 bytes base64url is 43 characters.
func TestTokensAreLongAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		tok, err := newToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(tok) != 43 {
			t.Fatalf("token is %d chars, want 43 (256 bits base64url)", len(tok))
		}
		if seen[tok] {
			t.Fatal("newToken repeated a token")
		}
		seen[tok] = true
	}
}

func TestHashTokenIsStableAndDistinct(t *testing.T) {
	if hashToken("a") != hashToken("a") {
		t.Error("hashing is not stable")
	}
	if hashToken("a") == hashToken("b") {
		t.Error("different tokens hashed the same")
	}
	if len(hashToken("a")) != 64 {
		t.Errorf("hash is %d chars, want 64 hex", len(hashToken("a")))
	}
}

// The sliding window is written back at a granularity, not on every request, or every page load
// becomes a write. It still has to be far finer than the idle window, or the window stops being
// meaningfully sliding.
func TestTouchGranularityIsFarBelowTheIdleWindow(t *testing.T) {
	for _, idle := range []time.Duration{time.Minute, time.Hour, 8 * time.Hour, 72 * time.Hour} {
		g := granularity(idle)
		if g < time.Minute {
			t.Errorf("idle %s: granularity %s is under a minute; that is a write per request", idle, g)
		}
		if g > 15*time.Minute {
			t.Errorf("idle %s: granularity %s is too coarse", idle, g)
		}
		if idle > 10*time.Minute && g > idle/10 {
			t.Errorf("idle %s: granularity %s is too close to the window to slide honestly", idle, g)
		}
	}
}

// Every cookie attribute here is load-bearing; this asserts each one rather than the string, so a
// failure names which guarantee was dropped.
func TestSessionCookieCarriesItsSecurityAttributes(t *testing.T) {
	w := httptest.NewRecorder()
	Cookies{Name: "__Host-tightship_session"}.Set(w, "tok", time.Now().Add(time.Hour))

	res := w.Result()
	cks := res.Cookies()
	if len(cks) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cks))
	}
	c := cks[0]
	if !c.HttpOnly {
		t.Error("not HttpOnly: script could read the session token")
	}
	if !c.Secure {
		t.Error("not Secure: the token would travel over plain HTTP")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax (Strict drops the cookie on the OIDC return leg)", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want / (required by the __Host- prefix)", c.Path)
	}
	if c.Domain != "" {
		t.Errorf("Domain = %q, want empty: a Domain attribute shares the cookie with subdomains", c.Domain)
	}
	if c.Value != "tok" {
		t.Errorf("Value = %q, want the token", c.Value)
	}
}

// Clearing has to repeat the attributes, or the browser treats it as a different cookie and
// leaves the original in place — a sign-out that does not sign out.
func TestClearMatchesTheCookieItRemoves(t *testing.T) {
	w := httptest.NewRecorder()
	Cookies{Name: "__Host-tightship_session"}.Clear(w)
	c := w.Result().Cookies()[0]
	if c.Path != "/" || !c.Secure || !c.HttpOnly {
		t.Errorf("Clear dropped attributes: path=%q secure=%v httponly=%v", c.Path, c.Secure, c.HttpOnly)
	}
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want negative so the browser deletes it", c.MaxAge)
	}
	if c.Value != "" {
		t.Errorf("Value = %q, want empty", c.Value)
	}
}

func TestTokenReadsTheCookie(t *testing.T) {
	c := Cookies{Name: "__Host-tightship_session"}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := c.Token(r); got != "" {
		t.Errorf("no cookie should give empty token, got %q", got)
	}
	r.AddCookie(&http.Cookie{Name: "__Host-tightship_session", Value: "abc"})
	if got := c.Token(r); got != "abc" {
		t.Errorf("Token = %q, want abc", got)
	}
}

// The migration must be embedded and reachable under the path the runner reads, or the table
// silently never gets created and every sign-in fails at run time instead of at build time.
func TestMigrationsAreEmbedded(t *testing.T) {
	b, err := fs.ReadFile(Migrations(), "migrations/0001_sessions.sql")
	if err != nil {
		t.Fatalf("migration not embedded where the runner looks: %v", err)
	}
	body := string(b)
	if !strings.Contains(body, "CREATE TABLE sessions") {
		t.Error("0001_sessions.sql does not create the sessions table")
	}
	// The table stores a hash. If this ever becomes a plain token column, a database backup
	// becomes a set of live sessions.
	if !strings.Contains(body, "token_hash") {
		t.Error("the table must store a token hash, never the token")
	}
}
