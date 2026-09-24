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

// migrateLockPrefix names the advisory lock every instance takes before migrating. Two binaries
// starting at the same moment — a rolling restart, a systemd retry — must not both decide the
// same migration is pending and both run it.
//
// GET_LOCK names are scoped to the SERVER, not to the schema, so the lock is qualified with the
// database name. Without that, a staging deployment sharing a server with production waits on
// production's migration and fails after the timeout, having nothing to do with it.
const migrateLockPrefix = "tightship:migrate:"

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

	release, err := lock(ctx, conn, migrateLockPrefix+c.Name)
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
	// Half-written rows are read from the ledger itself rather than by walking the current
	// migration set, because a migration can vanish from the build: disable the module in config,
	// or deploy without it, and a row left mid-apply becomes invisible to a check that only looks
	// at migrations it can still see. The schema is no less half-migrated for the module having
	// gone away.
	dirty, err := loadDirty(ctx, conn)
	if err != nil {
		return 0, err
	}

	// Check EVERY migration before applying ANY of them.
	//
	// These checks used to sit inside the apply loop, which made them fire only when iteration
	// reached the offending migration. Because modules apply alphabetically, a half-applied
	// migration in a late-sorting module let an earlier module's DDL run first — and MariaDB
	// cannot roll that back. The refusal then arrived after the schema had already moved, which is
	// precisely the "later migrations stacked on a schema nobody can describe" this is here to
	// prevent. A pre-flight pass is the only ordering where the refusal costs nothing.
	if err := preflight(migrations, state, dirty); err != nil {
		return 0, err
	}

	var ran int
	for _, m := range migrations {
		if _, seen := state[ledgerKey(m)]; seen {
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

// preflight refuses the whole run if any migration is in a state that makes applying anything
// unsafe. It reports every problem it finds, so a broken deployment is diagnosed in one pass
// rather than one failed start per fault.
func preflight(migrations []Migration, state map[string]applied, dirty []string) error {
	var problems []string

	// Anything left mid-apply, whether or not this build still contains it.
	for _, id := range dirty {
		problems = append(problems, fmt.Sprintf(
			"%s was started but never completed — a previous run died partway. MariaDB cannot "+
				"roll back DDL, so inspect the schema, finish or undo it by hand, then delete "+
				"its row from schema_migrations", id))
	}

	// The highest version each module has already applied. A pending migration below it would
	// apply in a different order here than on a fresh install — two deployments reporting the same
	// version with different schemas, which is the hazard the checksum rule exists to prevent,
	// reached by another route. It happens when two branches each add a migration and the
	// higher-numbered one merges first.
	highest := map[string]int{}
	for key, a := range state {
		if !a.done {
			continue
		}
		module, versionText, ok := strings.Cut(key, "/")
		if !ok {
			continue
		}
		v, err := strconv.Atoi(versionText)
		if err != nil {
			continue
		}
		if v > highest[module] {
			highest[module] = v
		}
	}

	for _, m := range migrations {
		prev, seen := state[ledgerKey(m)]
		if !seen {
			if h := highest[m.Module]; m.Version < h {
				problems = append(problems, fmt.Sprintf(
					"%s has not been applied, but %s is already at version %d. Applying it now "+
						"would give this deployment a different schema from a fresh install at the "+
						"same version; renumber it above %d",
					m.ID(), m.Module, h, h))
			}
			continue
		}
		if !prev.done {
			// Already reported from the ledger sweep above.
			continue
		}
		if prev.checksum != m.Checksum {
			// A released migration is immutable: editing one means two deployments reporting the
			// same version have different schemas.
			problems = append(problems, fmt.Sprintf(
				"%s has changed since it was applied (recorded %s, now %s). A released migration "+
					"is immutable; add a new one instead",
				m.ID(), short(prev.checksum), short(m.Checksum)))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("migrations: refusing to apply anything:\n  %s", strings.Join(problems, "\n  "))
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
func lock(ctx context.Context, conn *sql.Conn, name string) (func(), error) {
	var got sql.NullInt64
	// 30s: long enough for a rolling restart's second instance to wait out a short migration,
	// short enough that a stuck one fails the deployment instead of hanging it.
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 30)", name).Scan(&got); err != nil {
		return nil, fmt.Errorf("migrations: acquire lock: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		return nil, errors.New("migrations: another instance is migrating (lock timeout)")
	}
	return func() {
		// Best effort: the lock is released by the session ending in any case.
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "SELECT RELEASE_LOCK(?)", name)
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

// loadDirty returns the identifiers of every half-written ledger row, independent of what this
// build contains.
func loadDirty(ctx context.Context, conn *sql.Conn) ([]string, error) {
	rows, err := conn.QueryContext(ctx,
		"SELECT module, version, name FROM schema_migrations WHERE applied_at IS NULL ORDER BY module, version")
	if err != nil {
		return nil, fmt.Errorf("migrations: read schema_migrations: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var module, name string
		var version int
		if err := rows.Scan(&module, &version, &name); err != nil {
			return nil, err
		}
		out = append(out, fmt.Sprintf("%s/%04d_%s", module, version, name))
	}
	return out, rows.Err()
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
