package database

import (
	"os"
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

// A comma inside a column's TYPE must not hide the NOT NULL rule. The first version of this lint
// scanned the whole statement with a pattern that could not cross a comma, so DECIMAL(10,2) and
// ENUM('a','b') — the two types most likely to carry money and status — went unreported while
// VARCHAR(8) was caught. The guarantee the tool advertises silently did not hold for them.
func TestLintSeesNotNullThroughCommasInTheType(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
		want bool // want a finding
	}{
		{"varchar", "ALTER TABLE t ADD COLUMN s VARCHAR(8) NOT NULL;", true},
		{"decimal", "ALTER TABLE invoices ADD COLUMN total DECIMAL(10,2) NOT NULL;", true},
		{"enum", "ALTER TABLE t ADD COLUMN status ENUM('open','closed') NOT NULL;", true},
		{"decimal with default is fine", "ALTER TABLE t ADD COLUMN total DECIMAL(10,2) NOT NULL DEFAULT 0;", false},
		{"enum with default is fine", "ALTER TABLE t ADD COLUMN s ENUM('a','b') NOT NULL DEFAULT 'a';", false},
		{"nullable decimal is fine", "ALTER TABLE t ADD COLUMN total DECIMAL(10,2) NULL;", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := lintOne(tc.sql)
			if tc.want && len(got) == 0 {
				t.Errorf("no finding for %q", tc.sql)
			}
			if !tc.want && len(got) != 0 {
				t.Errorf("unexpected finding for %q: %v", tc.sql, got)
			}
		})
	}
}

// A statement can break more than one guarantee, and reporting one per run means the second is
// found only after the first is fixed — one CI cycle each.
func TestLintReportsEveryProblemInOneStatement(t *testing.T) {
	got := lintOne("ALTER TABLE t DROP COLUMN a, CHANGE b c INT;")
	if len(got) != 2 {
		t.Errorf("got %d findings, want 2 (a dropped column and a CHANGE): %v", len(got), got)
	}
	var sawDrop, sawChange bool
	for _, f := range got {
		if strings.Contains(f.Why, "drops a column") {
			sawDrop = true
		}
		if strings.Contains(f.Why, "CHANGE") {
			sawChange = true
		}
	}
	if !sawDrop || !sawChange {
		t.Errorf("findings did not cover both problems: %v", got)
	}
}

// Splitting an ALTER's clauses must not split inside parentheses or quotes, or a type list
// becomes two clauses and the rules read nonsense.
func TestLintSplitsClausesWithoutBreakingTypesOrStrings(t *testing.T) {
	// Several additions in one statement, each fine on its own.
	ok := "ALTER TABLE t ADD COLUMN a DECIMAL(10,2) NULL, ADD COLUMN b ENUM('x','y') NULL, ADD COLUMN c VARCHAR(8) NOT NULL DEFAULT '';"
	if got := lintOne(ok); len(got) != 0 {
		t.Errorf("clean multi-clause ALTER produced findings: %v", got)
	}
	// One bad clause among good ones is still found, and only it.
	bad := "ALTER TABLE t ADD COLUMN a DECIMAL(10,2) NULL, ADD COLUMN b VARCHAR(8) NOT NULL, ADD COLUMN c INT NULL;"
	got := lintOne(bad)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want exactly 1: %v", len(got), got)
	}
	if !strings.Contains(got[0].Statement, "b") {
		t.Errorf("finding blamed the wrong clause: %q", got[0].Statement)
	}
}

// isAlter once tested the raw statement for the literal "ALTER TABLE", so a newline or a double
// space between the two words disabled every rule for that statement AND the clause split. A lint
// that a reformat switches off is not a guard at all.
func TestLintIsNotDefeatedByWhitespace(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE rooms DROP COLUMN guid;",
		"ALTER  TABLE rooms DROP COLUMN guid;",
		"ALTER\n  TABLE rooms DROP COLUMN guid;",
		"ALTER\tTABLE rooms DROP COLUMN guid;",
		"alter table rooms drop column guid;",
	} {
		if got := lintOne(sql); len(got) == 0 {
			t.Errorf("whitespace or case defeated the lint: %q", sql)
		}
	}
}

// IF EXISTS is MariaDB's own spelling, and this lint targets MariaDB, so the most likely
// hand-written form of a defensive drop must not pass.
func TestLintSeesDropColumnIfExists(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE rooms DROP COLUMN IF EXISTS guid;",
		"ALTER TABLE rooms DROP IF EXISTS guid;",
	} {
		if got := lintOne(sql); len(got) == 0 {
			t.Errorf("no finding for %q", sql)
		}
	}
}

// ADD CONSTRAINT and ADD INDEX contain no column, so a NOT NULL inside them must not be read as
// one. This failed a build for a correct migration, which is how a lint loses its audience.
func TestLintDoesNotMistakeConstraintsForColumns(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE t ADD CONSTRAINT chk_b CHECK (b IS NOT NULL);",
		"ALTER TABLE t ADD CHECK (b IS NOT NULL);",
		"ALTER TABLE t ADD UNIQUE KEY uq_x (a, b);",
		"ALTER TABLE t ADD INDEX idx_x (a);",
		"ALTER TABLE t ADD FOREIGN KEY (a) REFERENCES u (id);",
	} {
		if got := lintOne(sql); len(got) != 0 {
			t.Errorf("false positive on %q: %v", sql, got)
		}
	}
}

// MODIFY is deliberately unchecked, and this pins that rather than leaving it to be re-added by
// someone reading reChangeCol's advice literally.
//
// A previous version did check it, and made widening a NOT NULL column impossible by any spelling:
// MariaDB's MODIFY must restate the whole definition, so widening VARCHAR(8) to VARCHAR(16)
// necessarily restates NOT NULL, and the rule flagged it — while CHANGE, the alternative the error
// message suggests, is refused by a different rule. Blocking ordinary schema work to catch a
// narrowing the lint cannot reliably identify is the wrong trade while the schema is still being
// designed. Tightening MODIFY belongs with schema hardening, once the shape has settled.
func TestLintLeavesModifyAlone(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE rooms MODIFY COLUMN code VARCHAR(16) NOT NULL;", // widening, must restate NOT NULL
		"ALTER TABLE rooms MODIFY COLUMN label VARCHAR(128) NULL;",   // plain widening
		"ALTER TABLE rooms MODIFY COLUMN code VARCHAR(8) NOT NULL DEFAULT '';",
	} {
		if got := lintOne(sql); len(got) != 0 {
			t.Errorf("MODIFY was reported, blocking schema work: %q -> %v", sql, got)
		}
	}
}

// String data must not be read as SQL. stripSQLComments leaves literals intact on purpose, so the
// rules have to be given a masked copy — otherwise a data migration is reported as DDL, and a
// column whose ENUM happens to contain the word "default" reads as though it had a DEFAULT clause.
func TestLintDoesNotReadStringDataAsSQL(t *testing.T) {
	// False positives: keywords inside data.
	for _, sql := range []string{
		"UPDATE settings SET v = 'DROP TABLE x';",
		"INSERT INTO capabilities (name, label) VALUES ('rooms.edit', 'Change a room');",
		"INSERT INTO capabilities (name, label) VALUES ('t', 'Drop table support');",
		"INSERT INTO roles (name, description) VALUES ('admin', 'May rename table entries');",
	} {
		if got := lintOne(sql); len(got) != 0 {
			t.Errorf("string data read as SQL: %q -> %v", sql, got)
		}
	}
	// False negatives: a literal containing DEFAULT must not satisfy the default check.
	for _, sql := range []string{
		"ALTER TABLE t ADD COLUMN status ENUM('open','default') NOT NULL;",
		"ALTER TABLE t ADD COLUMN note VARCHAR(50) NOT NULL COMMENT 'no default yet';",
	} {
		if got := lintOne(sql); len(got) == 0 {
			t.Errorf("a literal containing DEFAULT suppressed the rule: %q", sql)
		}
	}
}

// MariaDB's parenthesised ADD forms are valid and were missed entirely, because the capture landed
// on the word COLUMN rather than the column name.
func TestLintSeesParenthesisedAddForms(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE rooms ADD COLUMN (code VARCHAR(8) NOT NULL);",
		"ALTER TABLE rooms ADD (code VARCHAR(8) NOT NULL);",
	} {
		if got := lintOne(sql); len(got) == 0 {
			t.Errorf("no finding for %q", sql)
		}
	}
	// And the parenthesised form with a default is still fine.
	if got := lintOne("ALTER TABLE rooms ADD COLUMN (code VARCHAR(8) NOT NULL DEFAULT '');"); len(got) != 0 {
		t.Errorf("false positive: %v", got)
	}
}

// A column named `key`, `index` or `check` is perfectly ordinary — a settings table almost always
// has one — and a backtick is the author stating unambiguously that the word is a name, not
// syntax. Treating a quoted `key` as the KEY keyword let exactly the migration this rule exists to
// catch pass silently.
func TestLintReadsBacktickedKeywordsAsColumnNames(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE t ADD COLUMN `key` VARCHAR(64) NOT NULL;",
		"ALTER TABLE t ADD COLUMN `index` INT NOT NULL;",
		"ALTER TABLE t ADD COLUMN `check` TINYINT(1) NOT NULL;",
		"ALTER TABLE t DROP COLUMN `key`;",
	} {
		if got := lintOne(sql); len(got) == 0 {
			t.Errorf("a backtick-quoted column name was read as a keyword: %q", sql)
		}
	}
	// Unquoted, those words really are syntax and must still be excluded.
	for _, sql := range []string{
		"ALTER TABLE t ADD CONSTRAINT chk_b CHECK (b IS NOT NULL);",
		"ALTER TABLE t ADD UNIQUE KEY uq_x (a, b);",
		"ALTER TABLE t DROP INDEX idx_x;",
		"ALTER TABLE t DROP PRIMARY KEY;",
	} {
		if got := lintOne(sql); len(got) != 0 {
			t.Errorf("false positive on %q: %v", sql, got)
		}
	}
}

// A table rename is the most rollback-hostile change there is, and MariaDB's shortest spelling
// omits TO entirely.
func TestLintSeesRenameWithoutToOrAs(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE rooms RENAME spaces;",
		"ALTER TABLE rooms RENAME TO spaces;",
		"ALTER TABLE rooms RENAME AS spaces;",
		"RENAME TABLE rooms TO spaces;",
		"ALTER TABLE rooms RENAME COLUMN guid TO room_guid;",
	} {
		if got := lintOne(sql); len(got) == 0 {
			t.Errorf("no finding for %q", sql)
		}
	}
}

// Whether the lint fired used to depend on a comma the author may or may not type: the column name
// had to be the last token, so any trailing modifier hid the drop.
func TestLintSeesDropColumnWithTrailingModifiers(t *testing.T) {
	for _, sql := range []string{
		"ALTER TABLE t DROP COLUMN a;",
		"ALTER TABLE t DROP COLUMN a ALGORITHM=INPLACE;",
		"ALTER TABLE t DROP COLUMN a LOCK=NONE;",
		"ALTER TABLE t DROP COLUMN IF EXISTS a CASCADE;",
		"ALTER TABLE t DROP COLUMN a, LOCK=NONE;",
	} {
		if got := lintOne(sql); len(got) == 0 {
			t.Errorf("no finding for %q", sql)
		}
	}
}

// A finding carries no line number, so the statement text is the only handle an operator has for
// locating the clause. Quoting the masked copy gave them 'xxxx', which matches nothing they can
// grep for.
func TestLintFindingsQuoteTheOriginalSQL(t *testing.T) {
	got := lintOne("ALTER TABLE t ADD COLUMN status ENUM('open','closed') NOT NULL;")
	if len(got) == 0 {
		t.Fatal("expected a finding")
	}
	if strings.Contains(got[0].Statement, "xxx") {
		t.Errorf("finding quotes masked text: %q", got[0].Statement)
	}
	if !strings.Contains(got[0].Statement, "'open'") {
		t.Errorf("finding does not quote the real SQL: %q", got[0].Statement)
	}
}

// Regex over SQL trips on real-world input far more than on invented cases. Every statement in the
// corpus is a CREATE TABLE, which is always additive, so any finding at all is a false positive.
// This is the cheapest evidence available that the lint behaves on DDL nobody wrote for it.
func TestLintProducesNoFalsePositivesOnRealDDL(t *testing.T) {
	b, err := os.ReadFile("testdata/real_ddl_corpus.sql")
	if err != nil {
		t.Fatalf("corpus missing: %v", err)
	}
	got := Lint([]Migration{{Module: "corpus", Version: 1, Name: "real_ddl", SQL: string(b)}})
	if len(got) != 0 {
		for i, f := range got {
			if i < 10 {
				t.Errorf("false positive on real DDL: %s", f)
			}
		}
		t.Fatalf("%d false positive(s) across the real-DDL corpus", len(got))
	}
}
