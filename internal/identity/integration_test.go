package identity

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// A real database, because the throttle counter, the setup token's single use and the settings
// guard are all enforced in SQL. Skipped unless TIGHTSHIP_TEST_DSN is set.
func testService(t *testing.T) (context.Context, *Service, *sql.DB) {
	t.Helper()
	raw := os.Getenv("TIGHTSHIP_TEST_DSN")
	if raw == "" {
		t.Skip("set TIGHTSHIP_TEST_DSN to run identity integration tests")
	}
	host, user, password, name := "127.0.0.1", "", "", ""
	for _, kv := range strings.Fields(raw) {
		k, v, _ := strings.Cut(kv, "=")
		switch k {
		case "host":
			host = v
		case "user":
			user = v
		case "password":
			password = v
		case "name":
			name = v
		}
	}
	// Own database: packages run in parallel and each drops its own tables.
	name += "_identity"
	base := user + ":" + password + "@tcp(" + host + ":3306)/"
	opts := "?parseTime=true&loc=UTC&multiStatements=true&time_zone=%27%2B00%3A00%27"

	admin, err := sql.Open("mysql", base+opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		admin.Close()
		t.Skipf("cannot prepare %s: %v", name, err)
	}
	admin.Close()

	db, err := sql.Open("mysql", base+name+opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	for _, tbl := range []string{"local_credentials", "setup_tokens", "settings", "accounts"} {
		if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+tbl+"`"); err != nil {
			t.Fatal(err)
		}
	}
	ddl, err := migrations.ReadFile("migrations/0001_accounts.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(ddl)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	return ctx, NewService(db, quiet), db
}

func TestAuthenticateAcceptsTheRightPassword(t *testing.T) {
	ctx, s, _ := testService(t)
	acct, err := s.CreateAccount(ctx, "Gary@Example.ORG", "Gary", "staff")
	if err != nil {
		t.Fatal(err)
	}
	// The address is normalised, so sign-in is not case-sensitive in a way nobody expects.
	if acct.Email != "gary@example.org" {
		t.Errorf("email stored as %q, want lower-cased", acct.Email)
	}
	if err := s.SetPassword(ctx, acct.ID, "a-perfectly-fine-passphrase", false); err != nil {
		t.Fatal(err)
	}
	got, mustChange, err := s.Authenticate(ctx, "GARY@example.org", "a-perfectly-fine-passphrase")
	if err != nil {
		t.Fatalf("correct password rejected: %v", err)
	}
	if got.ID != acct.ID || mustChange {
		t.Errorf("got id=%d mustChange=%v", got.ID, mustChange)
	}
}

// Every failure looks the same from outside: no account, wrong password, disabled account.
func TestEveryFailureLooksIdentical(t *testing.T) {
	ctx, s, _ := testService(t)
	live, _ := s.CreateAccount(ctx, "live@example.org", "", "staff")
	_ = s.SetPassword(ctx, live.ID, "a-perfectly-fine-passphrase", false)
	off, _ := s.CreateAccount(ctx, "off@example.org", "", "staff")
	_ = s.SetPassword(ctx, off.ID, "a-perfectly-fine-passphrase", false)
	if _, err := s.db.ExecContext(ctx, "UPDATE accounts SET disabled_at = ? WHERE id = ?", time.Now().UTC(), off.ID); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ name, email, password string }{
		{"no such account", "nobody@example.org", "whatever-long-enough"},
		{"wrong password", "live@example.org", "wrong-but-long-enough"},
		{"disabled account, right password", "off@example.org", "a-perfectly-fine-passphrase"},
		{"account with no local credential", "nopass@example.org", "whatever-long-enough"},
	} {
		_, _, err := s.Authenticate(ctx, tc.email, tc.password)
		if !errors.Is(err, ErrAuth) {
			t.Errorf("%s: err = %v, want ErrAuth — a distinguishable failure is an enumeration oracle", tc.name, err)
		}
	}
}

// The throttle counts in the database so a restart does not reset an attack in progress.
func TestRepeatedFailuresLockTheCredential(t *testing.T) {
	ctx, s, _ := testService(t)
	acct, _ := s.CreateAccount(ctx, "gary@example.org", "", "staff")
	_ = s.SetPassword(ctx, acct.ID, "a-perfectly-fine-passphrase", false)

	for i := range s.maxFailures {
		if _, _, err := s.Authenticate(ctx, "gary@example.org", "wrong-but-long-enough"); !errors.Is(err, ErrAuth) {
			t.Fatalf("attempt %d: err = %v, want ErrAuth", i, err)
		}
	}
	// Now locked — and the RIGHT password must also be refused, or the lock protects nothing.
	if _, _, err := s.Authenticate(ctx, "gary@example.org", "a-perfectly-fine-passphrase"); !errors.Is(err, ErrLockedOut) {
		t.Errorf("after %d failures err = %v, want ErrLockedOut", s.maxFailures, err)
	}
	// It lifts on its own.
	s.now = func() time.Time { return time.Now().UTC().Add(s.lockFor + time.Minute) }
	if _, _, err := s.Authenticate(ctx, "gary@example.org", "a-perfectly-fine-passphrase"); err != nil {
		t.Errorf("the lock did not lift: %v", err)
	}
}

// A successful sign-in clears the counter, or a person who mistypes nine times over a term is
// locked out by their tenth attempt months later.
func TestSuccessClearsTheFailureCount(t *testing.T) {
	ctx, s, db := testService(t)
	acct, _ := s.CreateAccount(ctx, "gary@example.org", "", "staff")
	_ = s.SetPassword(ctx, acct.ID, "a-perfectly-fine-passphrase", false)

	for range 3 {
		_, _, _ = s.Authenticate(ctx, "gary@example.org", "wrong-but-long-enough")
	}
	if _, _, err := s.Authenticate(ctx, "gary@example.org", "a-perfectly-fine-passphrase"); err != nil {
		t.Fatal(err)
	}
	var failed int
	if err := db.QueryRowContext(ctx,
		"SELECT failed_attempts FROM local_credentials WHERE account_id = ?", acct.ID).Scan(&failed); err != nil {
		t.Fatal(err)
	}
	if failed != 0 {
		t.Errorf("failed_attempts = %d after a success, want 0", failed)
	}
}

// Resetting a password has to clear the lock, or a reset cannot recover a locked-out account —
// which is the situation a reset exists for.
func TestSettingAPasswordClearsTheLock(t *testing.T) {
	ctx, s, _ := testService(t)
	acct, _ := s.CreateAccount(ctx, "gary@example.org", "", "staff")
	_ = s.SetPassword(ctx, acct.ID, "a-perfectly-fine-passphrase", false)
	for range s.maxFailures {
		_, _, _ = s.Authenticate(ctx, "gary@example.org", "wrong-but-long-enough")
	}
	if _, _, err := s.Authenticate(ctx, "gary@example.org", "a-perfectly-fine-passphrase"); !errors.Is(err, ErrLockedOut) {
		t.Fatal("expected the account to be locked")
	}
	if err := s.SetPassword(ctx, acct.ID, "a-completely-different-passphrase", true); err != nil {
		t.Fatal(err)
	}
	if _, mustChange, err := s.Authenticate(ctx, "gary@example.org", "a-completely-different-passphrase"); err != nil {
		t.Errorf("reset did not clear the lock: %v", err)
	} else if !mustChange {
		t.Error("must_change was not carried through a reset")
	}
}

// ---------------------------------------------------------------------------
// The knob and its guard

// The guard that matters: local sign-in cannot be switched off until SSO has actually carried a
// sign-in. Otherwise a wrong redirect URI locks a district out of its own deployment.
func TestLocalCannotBeDisabledUntilSSOHasActuallyWorked(t *testing.T) {
	ctx, s, _ := testService(t)

	// A fresh database permits local sign-in; that default is what makes a new deployment
	// reachable at all.
	got, err := s.AuthSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got.LocalEnabled || got.SSOEnabled || got.SSOProven {
		t.Fatalf("fresh default = %+v, want local on, sso off, unproven", got)
	}

	// Turning SSO on is fine.
	if err := s.SetAuthSettings(ctx, AuthSettings{LocalEnabled: true, SSOEnabled: true}, "gary@example.org"); err != nil {
		t.Fatalf("enabling SSO alongside local was refused: %v", err)
	}
	// Turning local off while SSO has never worked is not.
	err = s.SetAuthSettings(ctx, AuthSettings{LocalEnabled: false, SSOEnabled: true}, "gary@example.org")
	if err == nil {
		t.Fatal("local sign-in was switched off before SSO had ever carried a sign-in")
	}
	if !strings.Contains(err.Error(), "successfully signed in with SSO") {
		t.Errorf("refusal does not explain itself: %v", err)
	}

	// Once somebody has actually signed in through SSO, it is allowed.
	if err := s.MarkSSOProven(ctx, "first.sso.user@example.org"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAuthSettings(ctx, AuthSettings{LocalEnabled: false, SSOEnabled: true}, "gary@example.org"); err != nil {
		t.Errorf("still refused after SSO was proven: %v", err)
	}
}

// Nobody gets to switch off every way in.
func TestCannotDisableEveryMethod(t *testing.T) {
	ctx, s, _ := testService(t)
	if err := s.SetAuthSettings(ctx, AuthSettings{LocalEnabled: false, SSOEnabled: false}, "gary@example.org"); err == nil {
		t.Error("a configuration with no way to sign in was accepted")
	}
}

// SSOProven is evidence, not policy: a caller must not be able to assert it and thereby unlock
// the guard that depends on it.
func TestSSOProvenCannotBeAssertedByACaller(t *testing.T) {
	ctx, s, _ := testService(t)
	err := s.SetAuthSettings(ctx,
		AuthSettings{LocalEnabled: false, SSOEnabled: true, SSOProven: true}, "attacker@example.org")
	if err == nil {
		t.Fatal("a caller claimed SSOProven and disabled local sign-in in one request")
	}
	got, _ := s.AuthSettings(ctx)
	if got.SSOProven {
		t.Error("SSOProven was written from a caller's request body")
	}
}

// With local sign-in off, a correct password is still refused — the knob has to bind the
// authentication path, not just the user interface.
func TestDisablingLocalActuallyRefusesCorrectPasswords(t *testing.T) {
	ctx, s, _ := testService(t)
	acct, _ := s.CreateAccount(ctx, "gary@example.org", "", "staff")
	_ = s.SetPassword(ctx, acct.ID, "a-perfectly-fine-passphrase", false)
	if err := s.MarkSSOProven(ctx, "someone@example.org"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAuthSettings(ctx, AuthSettings{LocalEnabled: false, SSOEnabled: true}, "gary@example.org"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Authenticate(ctx, "gary@example.org", "a-perfectly-fine-passphrase"); !errors.Is(err, ErrLocalDisabled) {
		t.Errorf("err = %v, want ErrLocalDisabled — the knob did not reach the auth path", err)
	}
}

// An unreadable settings row must not quietly become the permissive default.
func TestUnreadableSettingsAreAnErrorNotADefault(t *testing.T) {
	ctx, s, db := testService(t)
	if _, err := db.ExecContext(ctx,
		"INSERT INTO settings (name, value, updated_at) VALUES ('auth', '{not json', ?)", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthSettings(ctx); err == nil {
		t.Error("a corrupt settings row was silently replaced by the permissive default")
	}
}

// ---------------------------------------------------------------------------
// First-run setup

func TestSetupTokenIsSingleUseAndDiesOnceAnAccountExists(t *testing.T) {
	ctx, s, _ := testService(t)
	token, err := s.IssueSetupToken(ctx, time.Hour)
	if err != nil || token == "" {
		t.Fatalf("no setup token on an empty database: %q %v", token, err)
	}
	if _, err := s.CompleteSetup(ctx, token, "gary@example.org", "Gary", "a-perfectly-fine-passphrase"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	// Replay.
	if _, err := s.CompleteSetup(ctx, token, "attacker@example.org", "X", "another-fine-passphrase"); err == nil {
		t.Error("the setup token worked a second time")
	}
	// And no new token is issued now that an account exists, so none can leak.
	again, err := s.IssueSetupToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if again != "" {
		t.Error("a setup token was issued for a database that already has accounts")
	}
}

// Reissuing retires the previous token, so a restarted deployment does not leave a trail of
// working links in a log aggregator.
func TestIssuingANewSetupTokenRetiresTheOldOne(t *testing.T) {
	ctx, s, _ := testService(t)
	first, _ := s.IssueSetupToken(ctx, time.Hour)
	second, _ := s.IssueSetupToken(ctx, time.Hour)
	if first == second || first == "" || second == "" {
		t.Fatalf("expected two distinct tokens, got %q and %q", first, second)
	}
	if _, err := s.CompleteSetup(ctx, first, "gary@example.org", "Gary", "a-perfectly-fine-passphrase"); err == nil {
		t.Error("the superseded setup token still worked")
	}
	if _, err := s.CompleteSetup(ctx, second, "gary@example.org", "Gary", "a-perfectly-fine-passphrase"); err != nil {
		t.Errorf("the current setup token did not work: %v", err)
	}
}

func TestExpiredSetupTokenIsRefused(t *testing.T) {
	ctx, s, _ := testService(t)
	token, _ := s.IssueSetupToken(ctx, time.Hour)
	s.now = func() time.Time { return time.Now().UTC().Add(2 * time.Hour) }
	if _, err := s.CompleteSetup(ctx, token, "gary@example.org", "Gary", "a-perfectly-fine-passphrase"); err == nil {
		t.Error("an expired setup token was accepted")
	}
}

// The token is stored as a hash, so a database dump does not contain a working setup link.
func TestSetupTokenIsStoredOnlyAsAHash(t *testing.T) {
	ctx, s, db := testService(t)
	token, _ := s.IssueSetupToken(ctx, time.Hour)
	var n int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM setup_tokens WHERE token_hash = ?", token).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("the raw setup token is in the database")
	}
}

// Setup must apply the password policy; the first account is the one most worth attacking.
func TestSetupEnforcesThePasswordPolicy(t *testing.T) {
	ctx, s, _ := testService(t)
	token, _ := s.IssueSetupToken(ctx, time.Hour)
	if _, err := s.CompleteSetup(ctx, token, "gary@example.org", "Gary", "short"); err == nil {
		t.Fatal("a weak password was accepted for the first administrator")
	}
	// And the token survives a rejected attempt, or one typo burns the only way in.
	if _, err := s.CompleteSetup(ctx, token, "gary@example.org", "Gary", "a-perfectly-fine-passphrase"); err != nil {
		t.Errorf("the token was consumed by a rejected attempt: %v", err)
	}
}
