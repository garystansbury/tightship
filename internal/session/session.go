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
	"time"
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

// Effective is who this session acts as. It is Subject until impersonation exists, and the
// indirection is here now so the call sites that matter are already asking the right question.
func (s Session) Effective() string { return s.Subject }

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

	now func() time.Time // injectable so expiry can be tested without sleeping
}

// New builds a Store. idle is the sliding window, life the absolute ceiling.
func New(db *sql.DB, idle, life time.Duration) *Store {
	return &Store{
		db:         db,
		idle:       idle,
		life:       life,
		touchAfter: granularity(idle),
		now:        func() time.Time { return time.Now().UTC() },
	}
}

// granularity picks how often the sliding window is actually written back: a fortieth of the idle
// window, clamped to something sane. At the default 8h idle that is 12 minutes, so a session in
// continuous use is written about five times a day instead of thousands.
func granularity(idle time.Duration) time.Duration {
	g := idle / 40
	if g < time.Minute {
		g = time.Minute
	}
	if g > 15*time.Minute {
		g = 15 * time.Minute
	}
	return g
}

// Create issues a session and returns the token to put in the cookie. The token is returned once
// and never stored, so it cannot be recovered from the database or from a backup of it.
func (s *Store) Create(ctx context.Context, subject string, kind Kind, ip net.IP, userAgent string) (string, *Session, error) {
	if subject == "" {
		return "", nil, errors.New("session: subject is required")
	}
	token, err := newToken()
	if err != nil {
		return "", nil, err
	}
	now := s.now()
	sess := &Session{
		Subject: subject, Kind: kind,
		CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(s.life),
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, subject, kind, created_at, last_seen_at, expires_at, created_ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		hashToken(token), subject, string(kind), now, now, sess.ExpiresAt, ipBytes(ip), truncate(userAgent, 255))
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
// The two limits are checked in SQL rather than in Go so that a session cannot be resurrected by
// a clock difference between the application and the database: one clock decides.
func (s *Store) Lookup(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	now := s.now()
	idleCutoff := now.Add(-s.idle)

	var sess Session
	var kind string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, subject, kind, created_at, last_seen_at, expires_at
		FROM sessions
		WHERE token_hash = ?
		  AND revoked_at IS NULL
		  AND expires_at > ?
		  AND last_seen_at > ?`,
		hashToken(token), now, idleCutoff,
	).Scan(&sess.ID, &sess.Subject, &kind, &sess.CreatedAt, &sess.LastSeenAt, &sess.ExpiresAt)
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
		`UPDATE sessions SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		s.now(), hashToken(token))
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
		`UPDATE sessions SET revoked_at = ? WHERE subject = ? AND revoked_at IS NULL`, s.now(), subject)
	if err != nil {
		return 0, fmt.Errorf("session: revoke all: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ListFor returns a subject's live sessions, newest first, so a person can see where they are
// signed in. The token is not among the fields, because it is not stored.
func (s *Store) ListFor(ctx context.Context, subject string) ([]Session, error) {
	now := s.now()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subject, kind, created_at, last_seen_at, expires_at
		FROM sessions
		WHERE subject = ? AND revoked_at IS NULL AND expires_at > ? AND last_seen_at > ?
		ORDER BY last_seen_at DESC`,
		subject, now, now.Add(-s.idle))
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
	cutoff := s.now().Add(-retain)
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE expires_at < ?
		   OR (revoked_at IS NOT NULL AND revoked_at < ?)
		LIMIT ?`, cutoff, cutoff, batch)
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
