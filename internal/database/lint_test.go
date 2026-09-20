package database

import (
	"strings"
	"testing"
)

func lintOne(sql string) []Finding {
	return Lint([]Migration{{Module: "core", Version: 1, Name: "t", SQL: sql}})
}

// Each of these breaks a rollback to the previous release, which is the only reason the lint
// exists. Asserting the categories separately means a regression names which guarantee it broke.
func TestLintRefusesWhatBreaksARollback(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"drop table", "DROP TABLE rooms;", "drops a table"},
		{"drop column", "ALTER TABLE rooms DROP COLUMN guid;", "drops a column"},
		{"drop column without keyword", "ALTER TABLE rooms DROP guid;", "drops a column"},
		{"rename table", "RENAME TABLE rooms TO spaces;", "renames"},
		{"alter rename", "ALTER TABLE rooms RENAME TO spaces;", "renames"},
		{"change column", "ALTER TABLE rooms CHANGE guid room_guid CHAR(36);", "CHANGE"},
		{"not null no default", "ALTER TABLE rooms ADD COLUMN code VARCHAR(8) NOT NULL;", "NOT NULL column with no default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := lintOne(tc.sql)
			if len(got) == 0 {
				t.Fatalf("accepted %q", tc.sql)
			}
			if !strings.Contains(got[0].Why, tc.want) {
				t.Errorf("reason = %q, want it to mention %q", got[0].Why, tc.want)
			}
		})
	}
}

// The additive half must pass, or the lint blocks ordinary work and gets switched off.
func TestLintAllowsAdditiveChanges(t *testing.T) {
	for _, sql := range []string{
		"CREATE TABLE rooms (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, guid CHAR(36) NOT NULL);",
		"ALTER TABLE rooms ADD COLUMN label VARCHAR(64) NULL;",
		"ALTER TABLE rooms ADD COLUMN code VARCHAR(8) NOT NULL DEFAULT '';",
		"CREATE INDEX idx_rooms_guid ON rooms (guid);",
		"ALTER TABLE rooms MODIFY COLUMN label VARCHAR(128) NULL;",
		"INSERT INTO rooms (guid) VALUES ('x');",
	} {
		if got := lintOne(sql); len(got) != 0 {
			t.Errorf("rejected additive change %q: %v", sql, got)
		}
	}
}

// A NOT NULL column inside a CREATE TABLE is fine: there is no previous release inserting into a
// table that did not exist. Only ADD COLUMN on an existing table is the problem.
func TestLintDistinguishesCreateTableFromAddColumn(t *testing.T) {
	if got := lintOne("CREATE TABLE t (code VARCHAR(8) NOT NULL);"); len(got) != 0 {
		t.Errorf("NOT NULL in CREATE TABLE should be fine: %v", got)
	}
	if got := lintOne("ALTER TABLE t ADD COLUMN code VARCHAR(8) NOT NULL;"); len(got) == 0 {
		t.Error("NOT NULL in ADD COLUMN should be refused")
	}
}

// CLAUDE.md: a source assertion has to strip comments first, or a docblock quoting the spelling
// under test passes the assertion against broken code. The same hazard inverted applies here — a
// comment explaining why a migration does NOT drop a table would be reported as dropping one, and
// a lint with false positives is a lint people learn to ignore.
func TestLintIgnoresSQLComments(t *testing.T) {
	for _, sql := range []string{
		"-- We deliberately do not DROP TABLE rooms here; see docs/rbac.md\nCREATE TABLE rooms (id INT);",
		"/* RENAME TABLE was considered and rejected */ CREATE TABLE rooms (id INT);",
		"# DROP COLUMN guid would break rollback\nALTER TABLE rooms ADD COLUMN x INT NULL;",
	} {
		if got := lintOne(sql); len(got) != 0 {
			t.Errorf("comment tripped the lint: %q -> %v", sql, got)
		}
	}
}

// A comment must not be able to hide a real statement either: stripping happens before matching,
// so the code after a comment is still checked.
func TestLintStillSeesCodeAfterAComment(t *testing.T) {
	if got := lintOne("-- harmless note\nDROP TABLE rooms;"); len(got) == 0 {
		t.Error("a DROP TABLE after a comment was missed")
	}
}

// Statements are split so one statement's keywords are not attributed to its neighbour. A file
// that creates a table and drops another must report the drop and only the drop.
func TestLintAttributesFindingsToTheRightStatement(t *testing.T) {
	got := lintOne("CREATE TABLE a (id INT);\nDROP TABLE b;\nCREATE TABLE c (id INT);")
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(got), got)
	}
	if !strings.Contains(strings.ToUpper(got[0].Statement), "DROP TABLE B") {
		t.Errorf("finding blamed the wrong statement: %q", got[0].Statement)
	}
}

// A semicolon inside a string literal is not a statement boundary.
func TestLintDoesNotSplitInsideStringLiterals(t *testing.T) {
	got := lintOne("INSERT INTO settings (v) VALUES ('a;b');")
	if len(got) != 0 {
		t.Errorf("a semicolon in a literal produced findings: %v", got)
	}
}

// Every problem in one pass, so a migration is fixed in one edit rather than one CI run each.
func TestLintReportsEveryProblem(t *testing.T) {
	got := lintOne("DROP TABLE a;\nALTER TABLE b DROP COLUMN c;\nRENAME TABLE d TO e;")
	if len(got) != 3 {
		t.Errorf("got %d findings, want 3: %v", len(got), got)
	}
}
