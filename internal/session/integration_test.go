package session

import (
	"context"
	"database/sql"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// These need a real database: expiry is decided in SQL so that one clock rules, revocation is a
// row, and "the token is not stored" is a claim about what is actually on disk. Skipped unless
// TIGHTSHIP_TEST_DSN is set.
func testStore(t *testing.T, idle, life time.Duration) (context.Context, *Store, *sql.DB) {
	t.Helper()
	raw := os.Getenv("TIGHTSHIP_TEST_DSN")
	if raw == "" {
		t.Skip("set TIGHTSHIP_TEST_DSN to run session integration tests")
	}
	var host, user, password, name string
	host = "127.0.0.1"
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
	// Own database, not shared: `go test ./...` runs packages in parallel and the database
	// package's tests drop every table in their schema. See internal/database/integration_test.go.
	name += "_session"
	base := user + ":" + password + "@tcp(" + host + ":3306)/"
	opts := "?parseTime=true&loc=UTC&time_zone=%27%2B00%3A00%27"

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
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS sessions"); err != nil {
		t.Fatal(err)
	}
	ddl, err := readMigration()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatalf("create sessions table: %v", err)
	}
	return ctx, New(db, idle, life), db
}

func readMigration() (string, error) {
	b, err := migrations.ReadFile("migrations/0001_sessions.sql")
	return string(b), err
}

// at pins the store's clock so expiry can be tested without sleeping through it.
func at(s *Store, when time.Time) { s.now = func() time.Time { return when } }

func TestCreateAndLookupRoundTrip(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 24*time.Hour)
	token, created, err := store.Create(ctx, "teacher@example.org", KindStaff, net.ParseIP("10.1.2.3"), "Firefox")
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Lookup(ctx, token)
	if err != nil {
		t.Fatalf("a freshly created session did not look up: %v", err)
	}
	if got.Subject != "teacher@example.org" || got.Kind != KindStaff {
		t.Errorf("got %s/%s, want teacher@example.org/staff", got.Subject, got.Kind)
	}
	if got.ID != created.ID {
		t.Errorf("lookup returned id %d, create said %d", got.ID, created.ID)
	}
}

// The single most important property in this package: a database dump must not contain anything
// that can be replayed as a session.
func TestTheTokenIsNeverStored(t *testing.T) {
	ctx, store, db := testStore(t, time.Hour, 24*time.Hour)
	token, _, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	var found int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sessions WHERE token_hash = ?", token).Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found != 0 {
		t.Fatal("the raw token is in the database; a backup would be a set of live sessions")
	}
	// And the hash that IS stored must not be reversible to the token by simple presence.
	var stored string
	if err := db.QueryRowContext(ctx, "SELECT token_hash FROM sessions").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, token) || stored == token {
		t.Error("stored value contains the token")
	}
	if stored != hashToken(token) {
		t.Error("stored value is not the token's hash")
	}
}

func TestLookupRejectsAnUnknownToken(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 24*time.Hour)
	if _, err := store.Lookup(ctx, "not-a-real-token"); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if _, err := store.Lookup(ctx, ""); err != ErrNotFound {
		t.Errorf("empty token: err = %v, want ErrNotFound", err)
	}
}

// The idle window: unused for longer than the timeout and the session is gone.
func TestSessionIdlesOut(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 30*24*time.Hour)
	start := time.Now().UTC().Add(-48 * time.Hour)
	at(store, start)
	token, _, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	at(store, start.Add(59*time.Minute))
	if _, err := store.Lookup(ctx, token); err != nil {
		t.Fatalf("session died before the idle window elapsed: %v", err)
	}

	at(store, start.Add(2*time.Hour))
	if _, err := store.Lookup(ctx, token); err != ErrNotFound {
		t.Errorf("session survived past the idle window: err = %v", err)
	}
}

// Using a session slides the window: the point of an idle timeout is that activity refreshes it.
func TestUseSlidesTheIdleWindow(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 30*24*time.Hour)
	start := time.Now().UTC().Add(-48 * time.Hour)
	at(store, start)
	token, _, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	// Touch it every 45 minutes for six hours — well past the one-hour idle window in total.
	for i := 1; i <= 8; i++ {
		at(store, start.Add(time.Duration(i)*45*time.Minute))
		if _, err := store.Lookup(ctx, token); err != nil {
			t.Fatalf("session expired at step %d despite continuous use: %v", i, err)
		}
	}
}

// The reason there are two limits: continuous use must NOT extend a session forever. This is what
// bounds a stolen cookie.
func TestAbsoluteLifetimeBeatsContinuousUse(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 4*time.Hour)
	start := time.Now().UTC().Add(-48 * time.Hour)
	at(store, start)
	token, _, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	// Used every half hour, so the idle window never lapses.
	var lastOK time.Duration
	for i := 1; i <= 12; i++ {
		elapsed := time.Duration(i) * 30 * time.Minute
		at(store, start.Add(elapsed))
		if _, err := store.Lookup(ctx, token); err != nil {
			break
		}
		lastOK = elapsed
	}
	// Expiry is exclusive: the session is dead AT expires_at, not after it. So the last touch
	// inside a 4h lifetime is the one at 3h30m, and the touch landing exactly on 4h fails.
	if lastOK != 3*time.Hour+30*time.Minute {
		t.Errorf("last successful touch was %v in; the 4h absolute lifetime did not hold", lastOK)
	}
}

// Expiry is exclusive, and the boundary is worth pinning rather than inferring from a loop: a
// session must be dead at expires_at, not one tick later. An off-by-one here is a session that
// outlives its own limit.
func TestExpiryBoundaryIsExclusive(t *testing.T) {
	ctx, store, db := testStore(t, time.Hour, 4*time.Hour)
	start := time.Now().UTC().Add(-48 * time.Hour)
	at(store, start)
	token, sess, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	// Keep the idle window open so only the absolute limit can end this.
	if _, err := db.ExecContext(ctx, "UPDATE sessions SET last_seen_at = ? WHERE id = ?",
		sess.ExpiresAt.Add(-time.Minute), sess.ID); err != nil {
		t.Fatal(err)
	}

	at(store, sess.ExpiresAt.Add(-time.Second))
	if _, err := store.Lookup(ctx, token); err != nil {
		t.Errorf("session was dead one second before expires_at: %v", err)
	}
	at(store, sess.ExpiresAt)
	if _, err := store.Lookup(ctx, token); err != ErrNotFound {
		t.Errorf("session was alive exactly at expires_at: err = %v", err)
	}
}

// Revocation by row, and the row survives for the audit trail.
func TestRevokeEndsTheSessionButKeepsTheRow(t *testing.T) {
	ctx, store, db := testStore(t, time.Hour, 24*time.Hour)
	token, _, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Lookup(ctx, token); err != ErrNotFound {
		t.Errorf("revoked session still resolves: err = %v", err)
	}
	var n int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sessions WHERE revoked_at IS NOT NULL").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("revoked rows = %d, want 1 kept for the audit trail", n)
	}
}

// Sign out everywhere — and only for that subject.
func TestRevokeAllForOneSubjectLeavesOthersAlone(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 24*time.Hour)
	var mine []string
	for range 3 {
		tok, _, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
		if err != nil {
			t.Fatal(err)
		}
		mine = append(mine, tok)
	}
	other, _, err := store.Create(ctx, "someone.else@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	n, err := store.RevokeAllFor(ctx, "teacher@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("revoked %d, want 3", n)
	}
	for i, tok := range mine {
		if _, err := store.Lookup(ctx, tok); err != ErrNotFound {
			t.Errorf("session %d survived sign-out-everywhere", i)
		}
	}
	if _, err := store.Lookup(ctx, other); err != nil {
		t.Errorf("another subject's session was revoked too: %v", err)
	}
}

func TestListForShowsLiveSessionsOnly(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 24*time.Hour)
	keep, _, _ := store.Create(ctx, "teacher@example.org", KindStaff, nil, "Firefox")
	drop, _, _ := store.Create(ctx, "teacher@example.org", KindStaff, nil, "Chrome")
	if err := store.Revoke(ctx, drop); err != nil {
		t.Fatal(err)
	}
	list, err := store.ListFor(ctx, "teacher@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("listed %d sessions, want 1 live", len(list))
	}
	got, _ := store.Lookup(ctx, keep)
	if list[0].ID != got.ID {
		t.Error("listed the wrong session")
	}
}

// Different kinds of user share one table and one cookie — D3's "one interface for every kind of
// user" has to be true at this level before sign-in can rely on it.
func TestOneTableHoldsEveryKindOfUser(t *testing.T) {
	ctx, store, _ := testStore(t, time.Hour, 24*time.Hour)
	for _, k := range []Kind{KindStaff, KindStudent, KindContractor} {
		tok, _, err := store.Create(ctx, string(k)+"@example.org", k, nil, "")
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		got, err := store.Lookup(ctx, tok)
		if err != nil {
			t.Fatalf("%s: lookup: %v", k, err)
		}
		if got.Kind != k {
			t.Errorf("kind round-tripped as %q, want %q", got.Kind, k)
		}
	}
}

// The retention sweep removes what can no longer authenticate anything, keeps recent corpses for
// "why was I signed out", and stays inside its batch so it cannot lock the table for long.
func TestDeleteExpiredIsBoundedAndRespectsRetention(t *testing.T) {
	ctx, store, db := testStore(t, time.Hour, time.Hour)
	old := time.Now().UTC().Add(-30 * 24 * time.Hour)
	at(store, old)
	for range 5 {
		if _, _, err := store.Create(ctx, "old@example.org", KindStaff, nil, ""); err != nil {
			t.Fatal(err)
		}
	}
	at(store, time.Now().UTC())
	fresh, _, err := store.Create(ctx, "now@example.org", KindStaff, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	// A batch of 2 must delete exactly 2, not everything.
	n, err := store.DeleteExpired(ctx, 7*24*time.Hour, 2)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("deleted %d with batch=2, want 2", n)
	}
	// Drain.
	for {
		n, err := store.DeleteExpired(ctx, 7*24*time.Hour, 100)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			break
		}
	}
	var left int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 1 {
		t.Errorf("%d rows left, want 1 (the live one)", left)
	}
	if _, err := store.Lookup(ctx, fresh); err != nil {
		t.Errorf("the sweep removed a live session: %v", err)
	}
}

// Two sessions must never collide on the unique token hash, and creating many must not error.
func TestConcurrentCreatesDoNotCollide(t *testing.T) {
	ctx, store, db := testStore(t, time.Hour, 24*time.Hour)
	errs := make(chan error, 20)
	for i := range 20 {
		go func(i int) {
			_, _, err := store.Create(ctx, "teacher@example.org", KindStaff, nil, "")
			errs <- err
		}(i)
	}
	for range 20 {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent create failed: %v", err)
		}
	}
	var distinct, total int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT token_hash), COUNT(*) FROM sessions").Scan(&distinct, &total); err != nil {
		t.Fatal(err)
	}
	if distinct != total || total != 20 {
		t.Errorf("distinct=%d total=%d, want 20 and 20", distinct, total)
	}
}
