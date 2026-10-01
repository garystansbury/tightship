// Package session is the server-side session store. A signed-in browser holds one cookie
// containing a random token; everything else about the session lives in a row here, so a session
// can be revoked, listed, and expired by the server rather than merely by asking the browser
// nicely to forget something.
//
// One cookie for every kind of user. Staff arriving through Google, students through the IdP and
// contractors on a magic link all get a row in the same table with a different `kind`. D3 says
// there is one interface for every kind of user; this is the part of it that has to be true
// before sign-in can be written.
//
// Nothing here decides what a session may do. Grants and capabilities are the identity module's
// business; a session answers only "who is this, and is this still valid".
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Name identifies this store to the migration runner. It is not a module — it has no routes, no
// capabilities and no navigation — but it owns a table, and database.Source is deliberately
// narrow enough to say exactly that.
func Name() string { return "session" }

// Migrations is the embedded SQL the core migrator applies.
func Migrations() fs.FS { return migrations }

// Source adapts this package to database.Source without importing it, so the dependency runs one
// way: cmd wires them together and neither package knows the other.
type Source struct{}

func (Source) Name() string      { return Name() }
func (Source) Migrations() fs.FS { return Migrations() }

// Kind distinguishes the populations that share the table and the cookie.
type Kind string

const (
	KindStaff      Kind = "staff"
	KindStudent    Kind = "student"
	KindContractor Kind = "contractor"
)

// ErrNotFound is returned for a token that names no live session — absent, expired, idle out, or
// revoked. It is deliberately one error: telling a caller which of those it was would tell an
// attacker holding a stolen cookie whether the session ever existed.
var ErrNotFound = errors.New("session: no live session for that token")

// Session is one row, as callers see it. The token is not here: it exists once, at creation, and
// is never readable again.
type Session struct {
	ID         int64
	Subject    string
	Kind       Kind
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

// Store issues and validates sessions.
type Store struct {
	db   *sql.DB
	idle time.Duration
	life time.Duration

	// touchAfter is how stale last_seen_at must be before Lookup writes it back. Updating the row
	// on every request would turn every page load — every poll, every asset fetch that carries the
	// cookie — into a write, and on a fleet this size that is the busiest write in the system for
	// no benefit. A granularity well under the idle window keeps the sliding behaviour honest.
	touchAfter time.Duration

	// fixed pins the clock, for tests. It is nil in production, and then every timestamp this
	// store reads or writes is the DATABASE's, not the application's — see dbNow.
	fixed *time.Time
}

// dbNow is the expression every time-dependent statement uses for "now". Its parameter is nil in
// production, so COALESCE falls through to the database server's own clock; a test passes a
// pinned time instead.
//
// One clock has to decide, and it cannot be the application's. With several instances behind a
// load balancer, a node whose clock lags would keep accepting a session past expires_at, and one
// whose clock runs ahead would write a last_seen_at in the future and stretch the idle window for
// every node. The database is the one clock they all share.
//
// The CAST is load-bearing. With a parameter supplied, COALESCE's result type is a string, not a
// DATETIME — so it would not scan as a time, and the comparisons it feeds would be
// datetime-against-string. Casting makes it the same type whichever branch supplies the value.
const dbNow = "CAST(COALESCE(?, UTC_TIMESTAMP(3)) AS DATETIME(3))"

// clock is the argument for a dbNow placeholder.
func (s *Store) clock() any {
	if s.fixed == nil {
		return nil
	}
	return *s.fixed
}

// New builds a Store. idle is the sliding window, life the absolute ceiling.
func New(db *sql.DB, idle, life time.Duration) *Store {
	return &Store{db: db, idle: idle, life: life, touchAfter: granularity(idle)}
}

// granularity picks how often the sliding window is actually written back: a fortieth of the idle
// window, clamped to something sane. At the default 8h idle that is 12 minutes, so a session in
// continuous use is written about five times a day instead of thousands.
//
// It must also stay well under the idle window itself. With a one-minute floor and an idle
// timeout of a minute or less, a session in constant use was never written back before it idled
// out — so the window did not slide at all, and the session died one idle period after creation.
// Capping at half the window guarantees at least one write-back inside it.
func granularity(idle time.Duration) time.Duration {
	g := idle / 40
	if g < time.Minute {
		g = time.Minute
	}
	if g > 15*time.Minute {
		g = 15 * time.Minute
	}
	if half := idle / 2; g > half {
		g = half
	}
	return g
}

// Column limits, in characters. utf8mb4 VARCHAR lengths count characters, not bytes.
const (
	maxSubject   = 191
	maxUserAgent = 255
)

// Create issues a session and returns the token to put in the cookie. The token is returned once
// and never stored, so it cannot be recovered from the database or from a backup of it.
func (s *Store) Create(ctx context.Context, subject string, kind Kind, ip net.IP, userAgent string) (string, *Session, error) {
	if subject == "" {
		return "", nil, errors.New("session: subject is required")
	}
	// Refused here rather than left to the database. Under strict mode an over-long or malformed
	// subject is an opaque "Incorrect string value" from the INSERT; this says what is wrong.
	if !utf8.ValidString(subject) || utf8.RuneCountInString(subject) > maxSubject {
		return "", nil, fmt.Errorf("session: subject must be valid UTF-8 of at most %d characters", maxSubject)
	}
	token, err := newToken()
	if err != nil {
		return "", nil, err
	}

	var now time.Time
	if err := s.db.QueryRowContext(ctx, "SELECT "+dbNow, s.clock()).Scan(&now); err != nil {
		return "", nil, fmt.Errorf("session: read clock: %w", err)
	}
	// DATETIME(3) keeps milliseconds. Truncating here makes what is returned exactly what is
	// stored, so a caller comparing against ExpiresAt is comparing against the real boundary.
	now = now.UTC().Truncate(time.Millisecond)
	sess := &Session{
		Subject: subject, Kind: kind,
		CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(s.life),
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, subject, kind, created_at, last_seen_at, expires_at, created_ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		hashToken(token), subject, string(kind), now, now, sess.ExpiresAt, ipBytes(ip),
		clean(userAgent, maxUserAgent))
	if err != nil {
		return "", nil, fmt.Errorf("session: create: %w", err)
	}
	if id, err := res.LastInsertId(); err == nil {
		sess.ID = id
	}
	return token, sess, nil
}

// Lookup validates a token and slides the idle window. It returns ErrNotFound for anything that
// is not a live session.
//
// Both limits are decided by the database's clock, in the same statement that finds the row, and
// the "now" it used comes back with the row so the write-back below uses that same instant.
func (s *Store) Lookup(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, ErrNotFound
	}

	var sess Session
	var kind string
	var now time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.subject, s.kind, s.created_at, s.last_seen_at, s.expires_at, clk.n
		FROM sessions s, (SELECT `+dbNow+` AS n) clk
		WHERE s.token_hash = ?
		  AND s.revoked_at IS NULL
		  AND s.expires_at > clk.n
		  AND s.last_seen_at > TIMESTAMPADD(MICROSECOND, ?, clk.n)`,
		s.clock(), hashToken(token), -s.idle.Microseconds(),
	).Scan(&sess.ID, &sess.Subject, &kind, &sess.CreatedAt, &sess.LastSeenAt, &sess.ExpiresAt, &now)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("session: lookup: %w", err)
	}
	sess.Kind = Kind(kind)

	if now.Sub(sess.LastSeenAt) >= s.touchAfter {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE sessions SET last_seen_at = ? WHERE id = ? AND revoked_at IS NULL`, now, sess.ID); err != nil {
			// The session is valid; failing the request because the sliding window could not be
			// written would turn a transient write error into a sign-out.
			return &sess, nil
		}
		sess.LastSeenAt = now
	}
	return &sess, nil
}

// Revoke ends one session. Revoking is a write, not a delete, so the row stays for the audit
// trail and a returning cookie can still be recognised as revoked rather than merely unknown.
func (s *Store) Revoke(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = `+dbNow+` WHERE token_hash = ? AND revoked_at IS NULL`,
		s.clock(), hashToken(token))
	if err != nil {
		return fmt.Errorf("session: revoke: %w", err)
	}
	return nil
}

// RevokeAllFor ends every live session for a subject and reports how many. This is "sign out
// everywhere", and it is also what runs when an account is disabled or a password is reset —
// the moment an account stops being trusted, its sessions have to stop too.
func (s *Store) RevokeAllFor(ctx context.Context, subject string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = `+dbNow+` WHERE subject = ? AND revoked_at IS NULL`,
		s.clock(), subject)
	if err != nil {
		return 0, fmt.Errorf("session: revoke all: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ListFor returns a subject's live sessions, newest first, so a person can see where they are
// signed in. The token is not among the fields, because it is not stored.
func (s *Store) ListFor(ctx context.Context, subject string) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.subject, s.kind, s.created_at, s.last_seen_at, s.expires_at
		FROM sessions s, (SELECT `+dbNow+` AS n) clk
		WHERE s.subject = ? AND s.revoked_at IS NULL
		  AND s.expires_at > clk.n
		  AND s.last_seen_at > TIMESTAMPADD(MICROSECOND, ?, clk.n)
		ORDER BY s.last_seen_at DESC`,
		s.clock(), subject, -s.idle.Microseconds())
	if err != nil {
		return nil, fmt.Errorf("session: list: %w", err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var sess Session
		var kind string
		if err := rows.Scan(&sess.ID, &sess.Subject, &kind, &sess.CreatedAt, &sess.LastSeenAt, &sess.ExpiresAt); err != nil {
			return nil, err
		}
		sess.Kind = Kind(kind)
		out = append(out, sess)
	}
	return out, rows.Err()
}

// DeleteExpired removes rows that can no longer authenticate anything, in bounded batches so the
// sweep cannot take a long lock on a table the request path reads. It returns how many it
// deleted; the caller loops until it returns zero. The job scheduler is its intended caller.
//
// retain keeps recently-dead sessions around for that long after they die, so "why was I signed
// out" is answerable.
func (s *Store) DeleteExpired(ctx context.Context, retain time.Duration, batch int) (int64, error) {
	if batch <= 0 {
		batch = 1000
	}
	cutoff := "TIMESTAMPADD(MICROSECOND, ?, " + dbNow + ")"
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE expires_at < `+cutoff+`
		   OR (revoked_at IS NOT NULL AND revoked_at < `+cutoff+`)
		LIMIT ?`,
		-retain.Microseconds(), s.clock(), -retain.Microseconds(), s.clock(), batch)
	if err != nil {
		return 0, fmt.Errorf("session: delete expired: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// newToken returns 256 bits of CSPRNG output, URL-safe. rand.Read is documented never to return
// a short read, and an error from it means the system has no entropy source — which must fail the
// sign-in rather than produce a guessable token.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ipBytes normalises to 16 bytes so v4 and v6 share one column, and returns nil for an absent or
// unparseable address rather than storing something misleading.
func ipBytes(ip net.IP) []byte {
	if ip == nil {
		return nil
	}
	return ip.To16()
}

// clean makes a client-supplied string safe to store in a utf8mb4 column of n characters.
//
// The User-Agent header is whatever the client sends. Cutting it to n BYTES could split a
// multi-byte character and leave invalid UTF-8, and a header that was invalid UTF-8 to begin with
// is the same problem without the cut — either way, strict mode rejects the INSERT and that
// browser can never sign in. So invalid sequences are replaced first, and the cut is made on a
// character boundary, at n characters, which is what the column actually counts.
func clean(s string, n int) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	i, count := 0, 0
	for i = range s {
		if count == n {
			break
		}
		count++
	}
	return s[:i]
}
