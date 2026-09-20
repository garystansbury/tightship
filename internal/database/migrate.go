package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/garystansbury/tightship/internal/config"
)

// Migrations live in each module, under a `migrations/` directory in its embedded FS, named
// `NNNN_what_it_does.sql`. The number orders them within the module and never changes once
// released; the name is for people.
const migrationsDir = "migrations"

// migrateLockName is the advisory lock every instance takes before migrating. Two binaries
// starting at the same moment — a rolling restart, a systemd retry — must not both decide the
// same migration is pending and both run it.
const migrateLockName = "tightship:migrate"

// Migration is one file from one module.
type Migration struct {
	Module   string
	Version  int
	Name     string
	SQL      string
	Checksum string // SHA-256 of SQL, hex
}

// ID is how a migration is named in logs and errors.
func (m Migration) ID() string { return fmt.Sprintf("%s/%04d_%s", m.Module, m.Version, m.Name) }

// Source is what the runner needs from a module. module.Module satisfies it; keeping the
// dependency this narrow means the migration tests do not have to build a whole module.
type Source interface {
	Name() string
	Migrations() fs.FS
}

// Collect reads every module's migrations and returns them in the order they will be applied:
// modules alphabetically, versions ascending within a module.
//
// Order between modules is alphabetical rather than the order they were wired, because wiring
// order is easy to change by accident and a migration set whose meaning depends on it is a trap.
// Rule 7 — a module owns its tables — is what makes alphabetical sufficient: a module that needed
// another module's tables to exist first would already be breaking it.
func Collect(sources []Source) ([]Migration, error) {
	var out []Migration
	var problems []string

	ordered := append([]Source(nil), sources...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name() < ordered[j].Name() })

	for _, s := range ordered {
		fsys := s.Migrations()
		if fsys == nil {
			continue
		}
		entries, err := fs.ReadDir(fsys, migrationsDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue // a module with no tables yet is fine
		}
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", s.Name(), err))
			continue
		}

		seen := map[int]string{}
		var mods []Migration
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
				continue
			}
			version, name, err := parseFilename(e.Name())
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", s.Name(), err))
				continue
			}
			if prev, dup := seen[version]; dup {
				// Two files claiming the same version is ambiguous, and the ambiguity would be
				// resolved differently depending on filesystem order. Refuse.
				problems = append(problems, fmt.Sprintf(
					"%s: version %d appears twice (%s and %s)", s.Name(), version, prev, e.Name()))
				continue
			}
			seen[version] = e.Name()

			body, err := fs.ReadFile(fsys, path.Join(migrationsDir, e.Name()))
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s/%s: %v", s.Name(), e.Name(), err))
				continue
			}
			sum := sha256.Sum256(body)
			mods = append(mods, Migration{
				Module:   s.Name(),
				Version:  version,
				Name:     name,
				SQL:      string(body),
				Checksum: hex.EncodeToString(sum[:]),
			})
		}
		sort.Slice(mods, func(i, j int) bool { return mods[i].Version < mods[j].Version })
		out = append(out, mods...)
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("migrations: %s", strings.Join(problems, "; "))
	}
	return out, nil
}

// parseFilename splits `0007_add_room_guid.sql` into 7 and "add_room_guid".
func parseFilename(fn string) (int, string, error) {
	base := strings.TrimSuffix(fn, ".sql")
	num, rest, found := strings.Cut(base, "_")
	if !found || num == "" || rest == "" {
		return 0, "", fmt.Errorf("%q must be named NNNN_description.sql", fn)
	}
	v, err := strconv.Atoi(num)
	if err != nil || v <= 0 {
		return 0, "", fmt.Errorf("%q must start with a positive version number", fn)
	}
	return v, rest, nil
}

// applied is one row of schema_migrations.
type applied struct {
	checksum string
	done     bool
}

// Migrate brings the database up to date and returns how many migrations it ran.
//
// The hazard this function exists to manage is that MariaDB does not have transactional DDL. A
// CREATE TABLE cannot be rolled back, so a migration that fails halfway leaves the schema in a
// state no amount of retrying will fix. The runner therefore records that a migration has STARTED
// before running it and marks it done afterwards; a row left half-written means a previous run
// died mid-migration, and the next start refuses to continue rather than applying later
// migrations on top of a schema that is not what anyone thinks it is.
func Migrate(ctx context.Context, c config.Database, password string, sources []Source, log *slog.Logger) (int, error) {
	migrations, err := Collect(sources)
	if err != nil {
		return 0, err
	}
	// Continue even with nothing to apply: the ledger is created unconditionally so that after any
	// successful start, schema_migrations exists. Health reads it, and a build with no modules yet
	// should still leave a database that later tooling can rely on.

	db, err := openForMigrations(ctx, c, password)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("migrations: %w", err)
	}
	defer conn.Close()

	release, err := lock(ctx, conn)
	if err != nil {
		return 0, err
	}
	defer release()

	if err := ensureLedger(ctx, conn); err != nil {
		return 0, err
	}
	state, err := loadLedger(ctx, conn)
	if err != nil {
		return 0, err
	}

	var ran int
	for _, m := range migrations {
		prev, seen := state[ledgerKey(m)]
		switch {
		case seen && !prev.done:
			// Refuse rather than retry. Whether the DDL got partway through is not knowable from
			// here, and guessing wrong corrupts the schema quietly.
			return ran, fmt.Errorf(
				"migrations: %s was started but never completed — a previous run died partway. "+
					"MariaDB cannot roll back DDL, so inspect the schema, finish or undo it by hand, "+
					"then delete its row from schema_migrations", m.ID())
		case seen && prev.checksum != m.Checksum:
			// A released migration is immutable. Editing one means two deployments that both
			// report the same version have different schemas.
			return ran, fmt.Errorf(
				"migrations: %s has changed since it was applied (recorded %s, now %s). "+
					"A released migration is immutable; add a new one instead",
				m.ID(), short(prev.checksum), short(m.Checksum))
		case seen:
			continue
		}

		log.Info("applying migration", "migration", m.ID())
		if err := apply(ctx, conn, m); err != nil {
			return ran, err
		}
		ran++
	}
	return ran, nil
}

func ledgerKey(m Migration) string { return m.Module + "/" + strconv.Itoa(m.Version) }

func short(sum string) string {
	if len(sum) > 12 {
		return sum[:12]
	}
	return sum
}

// lock takes the advisory lock and returns the release. GET_LOCK is session-scoped, so the
// connection is held for the whole migration run and the lock disappears on its own if the
// process dies — which is the behaviour wanted, since a crashed migrator must not block the next
// start forever.
func lock(ctx context.Context, conn *sql.Conn) (func(), error) {
	var got sql.NullInt64
	// 30s: long enough for a rolling restart's second instance to wait out a short migration,
	// short enough that a stuck one fails the deployment instead of hanging it.
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 30)", migrateLockName).Scan(&got); err != nil {
		return nil, fmt.Errorf("migrations: acquire lock: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		return nil, errors.New("migrations: another instance is migrating (lock timeout)")
	}
	return func() {
		// Best effort: the lock is released by the session ending in any case.
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "SELECT RELEASE_LOCK(?)", migrateLockName)
	}, nil
}

// ensureLedger creates schema_migrations. It is the one piece of schema the core owns outright,
// so it is created here rather than by a module migration — nothing else can run until it exists.
func ensureLedger(ctx context.Context, conn *sql.Conn) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  module      VARCHAR(64)  NOT NULL,
  version     INT UNSIGNED NOT NULL,
  name        VARCHAR(255) NOT NULL,
  checksum    CHAR(64)     NOT NULL,
  started_at  DATETIME(3)  NOT NULL,
  applied_at  DATETIME(3)  NULL,
  PRIMARY KEY (module, version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrations: create schema_migrations: %w", err)
	}
	return nil
}

func loadLedger(ctx context.Context, conn *sql.Conn) (map[string]applied, error) {
	rows, err := conn.QueryContext(ctx, "SELECT module, version, checksum, applied_at FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migrations: read schema_migrations: %w", err)
	}
	defer rows.Close()
	state := map[string]applied{}
	for rows.Next() {
		var module, checksum string
		var version int
		var appliedAt sql.NullTime
		if err := rows.Scan(&module, &version, &checksum, &appliedAt); err != nil {
			return nil, err
		}
		state[module+"/"+strconv.Itoa(version)] = applied{checksum: checksum, done: appliedAt.Valid}
	}
	return state, rows.Err()
}

// apply runs one migration and records it. The write-before/write-after pair around the DDL is
// what makes a partial failure detectable; see Migrate.
func apply(ctx context.Context, conn *sql.Conn, m Migration) error {
	now := time.Now().UTC()
	if _, err := conn.ExecContext(ctx,
		`INSERT INTO schema_migrations (module, version, name, checksum, started_at, applied_at)
		 VALUES (?, ?, ?, ?, ?, NULL)`,
		m.Module, m.Version, m.Name, m.Checksum, now); err != nil {
		return fmt.Errorf("migrations: %s: record start: %w", m.ID(), err)
	}
	if _, err := conn.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("migrations: %s failed: %w — schema may be partially changed; "+
			"MariaDB cannot roll back DDL", m.ID(), err)
	}
	if _, err := conn.ExecContext(ctx,
		`UPDATE schema_migrations SET applied_at = ? WHERE module = ? AND version = ?`,
		time.Now().UTC(), m.Module, m.Version); err != nil {
		return fmt.Errorf("migrations: %s: applied but not recorded: %w", m.ID(), err)
	}
	return nil
}
