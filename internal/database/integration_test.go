package database

import (
	"context"
	"database/sql"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/garystansbury/tightship/internal/config"
)

// These tests need a real MariaDB, because what they check cannot be faked: MariaDB does not roll
// back DDL, GET_LOCK is a server behaviour, and "the migration ran and the ledger says so" is a
// claim about two systems agreeing. They skip unless TIGHTSHIP_TEST_DSN names a database the test
// may create and drop tables in.
//
//	TIGHTSHIP_TEST_DSN='host=127.0.0.1 port=3306 user=x password=y name=tightship_test' go test ./internal/database/
func testConfig(t *testing.T) (config.Database, string) {
	t.Helper()
	raw := os.Getenv("TIGHTSHIP_TEST_DSN")
	if raw == "" {
		t.Skip("set TIGHTSHIP_TEST_DSN to run database integration tests")
	}
	c := config.Database{Host: "127.0.0.1", Port: 3306, ConnectTimeout: config.Duration(10 * time.Second),
		MaxOpenConns: 4, MaxIdleConns: 4, ConnMaxLifetime: config.Duration(time.Minute), ConnMaxIdleTime: config.Duration(time.Minute)}
	var password string
	for _, kv := range strings.Fields(raw) {
		k, v, _ := strings.Cut(kv, "=")
		switch k {
		case "host":
			c.Host = v
		case "user":
			c.User = v
		case "password":
			password = v
		case "name":
			c.Name = v
		}
	}
	// `go test ./...` runs packages in parallel, and this package's tests drop every table in
	// their schema. Sharing one database with another package's integration tests means whichever
	// starts second finds its tables gone — so the DSN names a base and each package works in its
	// own database beside it.
	c.Name = c.Name + "_database"
	if err := ensureSchema(c, password); err != nil {
		t.Skipf("cannot prepare %s: %v", c.Name, err)
	}
	return c, password
}

// ensureSchema creates this package's database if it is not there yet.
func ensureSchema(c config.Database, password string) error {
	admin := c
	admin.Name = ""
	db, err := sql.Open("mysql", dsn(admin, password, false))
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + c.Name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	return err
}

// freshDB hands back a connected pool with no tightship tables in it, so each test starts from a
// known state rather than from whatever the last one left.
func freshDB(t *testing.T) (context.Context, config.Database, string, *sql.DB) {
	t.Helper()
	c, password := testConfig(t)
	ctx := context.Background()
	db, err := Open(ctx, c, password)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	dropAll(t, ctx, db, c.Name)
	return ctx, c, password, db
}

func dropAll(t *testing.T, ctx context.Context, db *sql.DB, schema string) {
	t.Helper()
	rows, err := db.QueryContext(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema = ?", schema)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	rows.Close()
	for _, n := range names {
		if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS `"+n+"`"); err != nil {
			t.Fatalf("drop %s: %v", n, err)
		}
	}
}

func fsMod(name string, files map[string]string) Source {
	m := fstest.MapFS{}
	for n, body := range files {
		m["migrations/"+n] = &fstest.MapFile{Data: []byte(body)}
	}
	return sourceOf{name: name, fs: m}
}

var quiet = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

func TestMigrateAppliesOnceAndIsIdempotent(t *testing.T) {
	ctx, c, pw, db := freshDB(t)
	src := []Source{fsMod("core", map[string]string{
		"0001_rooms.sql": "CREATE TABLE rooms (id INT PRIMARY KEY, guid CHAR(36) NOT NULL)",
		"0002_label.sql": "ALTER TABLE rooms ADD COLUMN label VARCHAR(64) NULL",
	})}

	n, err := Migrate(ctx, c, pw, src, quiet)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if n != 2 {
		t.Errorf("first run applied %d, want 2", n)
	}

	// The second start of the same binary must do nothing. This is the common case — every
	// restart — and getting it wrong means every restart re-runs DDL.
	n, err = Migrate(ctx, c, pw, src, quiet)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if n != 0 {
		t.Errorf("second run applied %d, want 0", n)
	}

	var cols int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='rooms'`,
		c.Name).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	if cols != 3 {
		t.Errorf("rooms has %d columns, want 3", cols)
	}
}

// An edited migration means two deployments reporting the same version have different schemas.
// The checksum is how that is caught, and it must be caught before anything else is applied.
func TestMigrateRefusesAnEditedMigration(t *testing.T) {
	ctx, c, pw, _ := freshDB(t)
	before := []Source{fsMod("core", map[string]string{"0001_rooms.sql": "CREATE TABLE rooms (id INT PRIMARY KEY)"})}
	if _, err := Migrate(ctx, c, pw, before, quiet); err != nil {
		t.Fatal(err)
	}
	after := []Source{fsMod("core", map[string]string{"0001_rooms.sql": "CREATE TABLE rooms (id BIGINT PRIMARY KEY)"})}
	_, err := Migrate(ctx, c, pw, after, quiet)
	if err == nil || !strings.Contains(err.Error(), "has changed since it was applied") {
		t.Errorf("edited migration accepted, err = %v", err)
	}
}

// MariaDB cannot roll back DDL, so a migration that fails partway leaves the schema in a state
// nobody can infer. The runner must notice on the next start and refuse, rather than applying
// later migrations on top of it.
func TestMigrateRefusesToContinueAfterAPartialFailure(t *testing.T) {
	ctx, c, pw, db := freshDB(t)
	broken := []Source{fsMod("core", map[string]string{
		"0001_ok.sql":     "CREATE TABLE rooms (id INT PRIMARY KEY)",
		"0002_broken.sql": "CREATE TABLE rooms_2 (id INT PRIMARY KEY); THIS IS NOT SQL",
	})}
	if _, err := Migrate(ctx, c, pw, broken, quiet); err == nil {
		t.Fatal("a broken migration reported success")
	}

	// The ledger should now hold a started-but-not-applied row for 0002.
	var started, applied sql.NullTime
	if err := db.QueryRowContext(ctx,
		"SELECT started_at, applied_at FROM schema_migrations WHERE module='core' AND version=2").
		Scan(&started, &applied); err != nil {
		t.Fatalf("expected a half-written ledger row: %v", err)
	}
	if !started.Valid || applied.Valid {
		t.Fatalf("ledger row is started=%v applied=%v; want started set and applied NULL", started.Valid, applied.Valid)
	}

	// A later start must refuse and say what to do, not silently carry on.
	_, err := Migrate(ctx, c, pw, broken, quiet)
	if err == nil || !strings.Contains(err.Error(), "started but never completed") {
		t.Errorf("did not refuse after a partial failure, err = %v", err)
	}
}

// Two instances starting together — a rolling restart — must not both apply the same migration.
func TestMigrateIsSerialisedByTheAdvisoryLock(t *testing.T) {
	ctx, c, pw, _ := freshDB(t)
	src := []Source{fsMod("core", map[string]string{
		"0001_rooms.sql": "CREATE TABLE rooms (id INT PRIMARY KEY)",
	})}

	type result struct {
		n   int
		err error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			n, err := Migrate(ctx, c, pw, src, quiet)
			results <- result{n, err}
		}()
	}
	var total int
	for range 2 {
		r := <-results
		if r.err != nil {
			t.Errorf("concurrent migrate failed: %v", r.err)
		}
		total += r.n
	}
	// Exactly one of them did the work; the other waited for the lock and found nothing to do.
	if total != 1 {
		t.Errorf("two concurrent runs applied %d migrations in total, want 1", total)
	}
}

// The pool must hand back UTC. Rule 9 is "store UTC, display local", and the half that breaks
// quietly is the server's own now() on a host set to local time.
func TestSessionTimeZoneIsUTC(t *testing.T) {
	ctx, _, _, db := freshDB(t)
	var offset string
	if err := db.QueryRowContext(ctx, "SELECT @@session.time_zone").Scan(&offset); err != nil {
		t.Fatal(err)
	}
	if offset != "+00:00" {
		t.Errorf("session time_zone = %q, want +00:00", offset)
	}

	var now time.Time
	if err := db.QueryRowContext(ctx, "SELECT NOW()").Scan(&now); err != nil {
		t.Fatal(err)
	}
	if now.Location() != time.UTC {
		t.Errorf("NOW() scanned in %v, want UTC", now.Location())
	}
	if d := time.Since(now).Abs(); d > 2*time.Minute {
		t.Errorf("NOW() is %v from the client's clock; the session zone is probably not UTC", d)
	}
}

// The application pool must not be able to run two statements in one call, so that a
// query-building mistake cannot become a second statement.
func TestApplicationPoolRefusesMultipleStatements(t *testing.T) {
	ctx, _, _, db := freshDB(t)
	_, err := db.ExecContext(ctx, "SELECT 1; SELECT 2")
	if err == nil {
		t.Error("the application pool executed two statements in one call")
	}
}

// Strict mode must be on: a silently truncated value is the kind of defect that ships green.
func TestStrictModeRejectsTruncation(t *testing.T) {
	ctx, _, _, db := freshDB(t)
	if _, err := db.ExecContext(ctx, "CREATE TABLE strict_probe (v VARCHAR(4) NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO strict_probe (v) VALUES ('far too long')"); err == nil {
		t.Error("an over-long value was accepted; strict mode is not in force")
	}
}

var _ fs.FS = fstest.MapFS{}

// SELECT 1 needs no privilege on any table and no table to exist, so it stays green against a
// server the application cannot read a row from. Health has to fail in that case, which is the
// only case it is for.
func TestHealthFailsWhenTheSchemaIsUnreadable(t *testing.T) {
	ctx, c, pw, db := freshDB(t)
	if _, err := Migrate(ctx, c, pw, nil, quiet); err != nil {
		t.Fatalf("migrate (ledger only): %v", err)
	}
	if err := Health(ctx, db); err != nil {
		t.Fatalf("health failed on a good database: %v", err)
	}

	// Drop the ledger: stands in for any state where the schema is not what the binary expects.
	if _, err := db.ExecContext(ctx, "DROP TABLE schema_migrations"); err != nil {
		t.Fatal(err)
	}
	if err := Health(ctx, db); err == nil {
		t.Error("health reported ok with no schema_migrations table")
	}

	// And SELECT 1 would still have passed, which is the point.
	var one int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
		t.Errorf("SELECT 1 should still succeed here; got %d, %v", one, err)
	}
}

// After any successful start the ledger exists, even for a build with no modules — health reads
// it, and tooling should not have to special-case a database that has never migrated.
func TestMigrateCreatesTheLedgerEvenWithNoMigrations(t *testing.T) {
	ctx, c, pw, db := freshDB(t)
	n, err := Migrate(ctx, c, pw, nil, quiet)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("applied %d migrations from no sources", n)
	}
	var name string
	if err := db.QueryRowContext(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema=? AND table_name='schema_migrations'",
		c.Name).Scan(&name); err != nil {
		t.Fatalf("schema_migrations was not created: %v", err)
	}
}

// The exact case that was broken: a later-sorting module half-applied, an earlier-sorting module
// added in the next release. Nothing must be applied.
func TestMigrateRefusesEverythingWhenAnyMigrationIsUnsafe(t *testing.T) {
	ctx, c, pw, db := freshDB(t)
	broken := []Source{fsMod("core", map[string]string{
		"0001_ok.sql":     "CREATE TABLE core_one (id INT PRIMARY KEY)",
		"0002_broken.sql": "CREATE TABLE core_two (id INT PRIMARY KEY); THIS IS NOT SQL",
	})}
	if _, err := Migrate(ctx, c, pw, broken, quiet); err == nil {
		t.Fatal("expected the broken migration to fail")
	}
	next := append([]Source{fsMod("assets", map[string]string{
		"0001_assets.sql": "CREATE TABLE assets_table (id INT PRIMARY KEY)",
	})}, broken...)

	n, err := Migrate(ctx, c, pw, next, quiet)
	if err == nil {
		t.Fatal("a run with a half-applied migration reported success")
	}
	if n != 0 {
		t.Errorf("applied %d migrations despite refusing the run", n)
	}
	var found int
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name='assets_table'",
		c.Name).Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found != 0 {
		t.Error("an earlier-sorting module's DDL was applied before the refusal fired")
	}
}

// Preflight reports every fault at once, so a broken deployment is diagnosed in one start.
func TestPreflightReportsEveryFaultAtOnce(t *testing.T) {
	migrations := []Migration{
		{Module: "a", Version: 1, Name: "x", Checksum: "aaa"},
		{Module: "b", Version: 1, Name: "y", Checksum: "bbb"},
	}
	state := map[string]applied{
		"a/1": {checksum: "aaa", done: false},      // started, never completed
		"b/1": {checksum: "different", done: true}, // edited since applied
	}
	err := preflight(migrations, state)
	if err == nil {
		t.Fatal("preflight passed a ledger with two faults")
	}
	msg := err.Error()
	for _, want := range []string{"a/0001_x", "b/0001_y", "started but never completed", "has changed since"} {
		if !strings.Contains(msg, want) {
			t.Errorf("preflight message omits %q:\n%s", want, msg)
		}
	}
}
