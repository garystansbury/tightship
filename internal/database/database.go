// Package database opens the MariaDB pool and runs migrations. It is layer 3's front door: the
// deployment file says where the database is and how big the pool may be, the credential store
// (later) or a bootstrap environment variable supplies the password, and everything else in the
// binary receives a *sql.DB that is already configured correctly.
//
// Two decisions are made here once so no module has to remember them:
//
//   - Times are UTC on the wire. The session time zone is pinned to +00:00 and the driver parses
//     DATETIME into time.Time as UTC, so a server whose system zone is America/New_York cannot
//     make NOW() mean something different from what the application stores. Display-local is the
//     UI's job, from org.timezone.
//   - The application pool cannot execute multiple statements in one call. Migrations need that
//     and get their own connection with it enabled; the pool every request uses does not, so a
//     query-building mistake cannot become a second statement.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/garystansbury/tightship/internal/config"
)

// Open connects the application pool and verifies it before returning. A binary that starts
// without a working database and only discovers it on the first request has moved a deployment
// failure into a user-facing one.
func Open(ctx context.Context, c config.Database, password string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn(c, password, false))
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(c.ConnMaxLifetime)
	db.SetConnMaxIdleTime(c.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, c.ConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database: connect to %s: %w", net.JoinHostPort(c.Host, strconv.Itoa(c.Port)), err)
	}
	return db, nil
}

// openForMigrations returns a separate pool that may send multiple statements per call. It is
// deliberately not the pool the application uses: migration SQL is a compile-time constant
// embedded in the binary, so multi-statement is safe there, while in the request path it would
// turn any query-building slip into an opportunity to append a second statement.
func openForMigrations(ctx context.Context, c config.Database, password string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn(c, password, true))
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	// One connection: the migration lock is held per-session, so the runner must not be handed a
	// different connection halfway through and silently lose it.
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	pingCtx, cancel := context.WithTimeout(ctx, c.ConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database: connect for migrations: %w", err)
	}
	return db, nil
}

// dsn builds the go-sql-driver connection string.
func dsn(c config.Database, password string, multiStatements bool) string {
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	cfg.DBName = c.Name
	cfg.Timeout = c.ConnectTimeout
	cfg.MultiStatements = multiStatements

	// ParseTime with Loc=UTC makes the driver hand back time.Time in UTC rather than strings.
	// Without it every timestamp is a []byte each caller parses, and they will not all parse it
	// the same way.
	cfg.ParseTime = true
	cfg.Loc = time.UTC

	// Interpolation is off (the default) so arguments travel as bound parameters.
	cfg.InterpolateParams = false

	cfg.Params = map[string]string{
		// Pin the session zone. Rule 9 is "store UTC, display local", and the half of it that is
		// easy to get wrong is the server's own idea of now().
		"time_zone": "'+00:00'",
		// utf8mb4 throughout: a name with an emoji or a non-BMP character must round-trip, not
		// raise "Incorrect string value" on the row that finally contains one.
		"charset": "utf8mb4",
		// Reject silent truncation and zero dates. A warning the server would otherwise swallow
		// should be an error; the suite this replaces is full of defects that shipped green.
		//
		// CONCAT rather than a bare assignment, so this TIGHTENS the server's defaults instead of
		// replacing them. Assigning outright drops NO_ENGINE_SUBSTITUTION, under which a
		// CREATE TABLE ... ENGINE=InnoDB silently falls back to another engine with only a warning
		// if InnoDB is unavailable — precisely the class of failure this setting exists to close.
		"sql_mode": "CONCAT(@@sql_mode,',STRICT_ALL_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO')",
	}
	return cfg.FormatDSN()
}

// Health reports whether the database is actually usable, which is a stronger claim than that it
// answers. It reads schema_migrations rather than running SELECT 1, because SELECT 1 needs no
// privilege on any table and no table to exist: it succeeds against a server the application
// cannot read a single row from, which is the failure a health endpoint is for.
//
// Reading the ledger proves the connection, the schema, and this user's ability to read it, and
// costs one primary-key-ordered scan of the smallest table in the database.
func Health(ctx context.Context, db *sql.DB) error {
	var version sql.NullInt64
	err := db.QueryRowContext(ctx, "SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err != nil {
		return fmt.Errorf("database: health: %w", err)
	}
	return nil
}
