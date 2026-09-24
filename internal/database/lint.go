package database

import (
	"fmt"
	"regexp"
	"strings"
)

// A deployment rolls back by pinning the previous tag, which means the previous release's code
// has to keep working against this release's schema for one release. That is a policy nobody can
// hold in their head across a year of migrations, so it is checked instead: Lint refuses the
// statements that break it, and `tightship check` runs it.
//
// The rule is one-directional. Adding a table, a nullable column, an index or a default is
// invisible to older code. Removing or renaming anything it reads, or adding a column it must
// populate but does not know about, is not — the rollback comes up broken, during an incident,
// which is the worst possible moment to discover it.
//
// Removing a column is still allowed eventually; it just takes two releases. Stop writing it in
// release N, drop it in release N+1, by which point no deployment can roll back to code that
// still reads it.

// Finding is one lint problem.
type Finding struct {
	Migration string
	Statement string // the offending fragment, trimmed
	Why       string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s: %s (%s)", f.Migration, f.Why, f.Statement)
}

var (
	// Each pattern names the thing a rollback would trip over.
	reDropTable = regexp.MustCompile(`(?is)\bDROP\s+TABLE\b`)
	// IF EXISTS is MariaDB's own spelling and this lint targets MariaDB, so leaving it out meant
	// the most likely hand-written form of a defensive column drop passed silently.
	reDropColumn = regexp.MustCompile(`(?is)\bDROP\s+(?:COLUMN\s+)?(?:IF\s+EXISTS\s+)?(` + "`?" + `)(\w+)`)
	reRename     = regexp.MustCompile(`(?is)\bRENAME\b`)
	// CHANGE can rename, so it is refused. MODIFY, which cannot, is deliberately NOT checked at
	// all: telling a safe widening from a breaking narrowing needs the column's current
	// definition, which a lint reading migration files does not have. An earlier version did check
	// it and made widening a NOT NULL column impossible by any spelling — MariaDB's MODIFY must
	// restate the full definition, so a widening necessarily restates NOT NULL, and the rule
	// flagged it. Blocking ordinary schema work to catch something it cannot identify reliably is
	// the wrong trade during active design; tightening MODIFY belongs with schema hardening, once
	// the shape has settled.
	reChangeCol = regexp.MustCompile(`(?is)\bCHANGE\s+(?:COLUMN\s+)?` + "`?" + `\w+`)
	// Matched against a single clause, not a whole statement. An earlier version scanned the
	// statement with `[^,;]*` around NOT NULL, which cannot cross a comma — so any column whose
	// TYPE contains one escaped the rule entirely. DECIMAL(10,2) and ENUM('a','b') are the common
	// cases, and both went unreported while VARCHAR(8) was caught, so the guarantee this lint
	// advertises quietly did not hold for the two types most likely to carry money and status.
	// Captures the identifier after ADD so the caller can check it is a column name rather than a
	// keyword introducing a constraint or an index. Go's regexp is RE2 and has no negative
	// lookahead, so the exclusion is a lookup rather than a pattern — which is clearer anyway.
	// Without it, ADD CONSTRAINT chk CHECK (b IS NOT NULL) was reported as "adds a NOT NULL column
	// with no default": a statement that adds no column at all, failing a build that was correct.
	reNotNullAdd  = regexp.MustCompile(`(?is)\bADD\s+(?:COLUMN\s+)?\(?\s*(` + "`?" + `)(\w+)`)
	reHasDefault  = regexp.MustCompile(`(?is)\bDEFAULT\b|\bAUTO_INCREMENT\b`)
	reNotNullWord = regexp.MustCompile(`(?is)\bNOT\s+NULL\b`)
)

// Lint checks a set of migrations and returns every problem, so a migration is fixed in one pass
// rather than one failed CI run per mistake.
func Lint(migrations []Migration) []Finding {
	var out []Finding
	for _, m := range migrations {
		sql := stripSQLComments(m.SQL)
		for _, stmt := range splitStatements(sql) {
			trimmed := collapse(stmt)
			if trimmed == "" {
				continue
			}
			// Collapsed, not raw. Testing the raw statement meant a newline or a double space
			// between ALTER and TABLE set this false, which disabled both isAlter-gated rules AND
			// the clause split — so reformatting a migration turned the whole lint off for it.
			// Mask string data before any rule reads it. stripSQLComments deliberately leaves
			// literals intact — a comment marker inside a quoted string is not a comment — but the
			// rules must not then read that data as SQL. Without this, a data migration writing
			// 'DROP TABLE x' is reported as dropping a table, and a column declared
			// ENUM('open','default') reads as though it had a DEFAULT clause.
			// The rules read a masked copy; the message quotes the original. A finding carries no
			// line number, so the statement text is the only handle an operator has for locating
			// the clause — and "ENUM('xxxx','xxxxxx')" matches nothing they can grep for.
			masked := maskStringLiterals(stmt)
			trimmed = collapse(masked)
			isAlter := strings.Contains(strings.ToUpper(trimmed), "ALTER TABLE")

			// One ALTER may carry several clauses, and each is a separate decision: ADD this,
			// DROP that. Splitting on top-level commas — respecting parentheses and quotes, so a
			// DECIMAL(10,2) or an ENUM('a','b') stays in one piece — lets each clause be judged on
			// its own, and lets a statement report every problem it has rather than only the first.
			clauses, originals := []string{masked}, []string{stmt}
			if isAlter {
				clauses = splitTopLevelCommas(masked)
				// Split the original the same way. Masking preserves length and structure, so the
				// two lists correspond clause for clause.
				originals = splitTopLevelCommas(stmt)
			}
			for i, clause := range clauses {
				c := collapse(clause)
				if c == "" {
					continue
				}
				shown := c
				if i < len(originals) {
					shown = collapse(originals[i])
				}
				add := func(why string) {
					out = append(out, Finding{Migration: m.ID(), Statement: excerpt(shown), Why: why})
				}
				// Not a switch: a clause can break more than one guarantee, and reporting one
				// problem per run means the next one is found only after the first is fixed.
				if reDropTable.MatchString(clause) {
					add("drops a table; the previous release may still read it")
				}
				if reRename.MatchString(clause) {
					add("renames; a rename is a drop and an add to the previous release")
				}
				if reChangeCol.MatchString(clause) {
					add("uses CHANGE, which can rename; use MODIFY to alter a type in place")
				}
				if isAlter && dropsAColumn(clause) {
					add("drops a column; stop writing it in one release and drop it in the next")
				}
				// A new NOT NULL column with no default breaks the previous release's INSERTs,
				// which do not know to supply it. With a default, the old code's INSERT still
				// works. Checked per clause, so a comma inside the column's type cannot hide it.
				if isAlter && addsAColumn(clause) &&
					reNotNullWord.MatchString(clause) && !reHasDefault.MatchString(clause) {
					add("adds a NOT NULL column with no default; the previous release's INSERTs omit it")
				}
				// MODIFY is deliberately NOT checked. See the note above reChangeCol.
			}
		}
	}
	return out
}

// stripSQLComments removes -- line comments and /* */ blocks before any pattern is matched.
// Without this a migration whose comment explains why it does NOT drop a table would be reported
// as dropping one, and — worse — a reviewer who saw that false positive would learn to ignore the
// lint. String literals are left alone deliberately: a comment marker inside a quoted string is
// rare in DDL, and treating it as a comment would hide real SQL.
func stripSQLComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var inLine, inBlock, inSingle, inDouble, inBacktick bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inLine:
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
		case inBlock:
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				inBlock = false
				i++
			}
		case inSingle:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			} else if c == '\'' {
				inSingle = false
			}
		case inDouble:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			} else if c == '"' {
				inDouble = false
			}
		case inBacktick:
			b.WriteByte(c)
			if c == '`' {
				inBacktick = false
			}
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			inLine = true
			i++
		case c == '#':
			inLine = true
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			inBlock = true
			i++
		case c == '\'':
			inSingle = true
			b.WriteByte(c)
		case c == '"':
			inDouble = true
			b.WriteByte(c)
		case c == '`':
			inBacktick = true
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// splitStatements cuts on semicolons that are not inside a quoted string or identifier. It is not
// a SQL parser and does not need to be: it runs on comment-stripped DDL that this repository
// wrote, and its only job is to keep one statement's keywords from being attributed to another.
func splitStatements(s string) []string {
	var out []string
	var cur strings.Builder
	var inSingle, inDouble, inBacktick bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\\' && i+1 < len(s) {
				cur.WriteByte(c)
				i++
				c = s[i]
			} else if c == '\'' {
				inSingle = false
			}
		case inDouble:
			if c == '\\' && i+1 < len(s) {
				cur.WriteByte(c)
				i++
				c = s[i]
			} else if c == '"' {
				inDouble = false
			}
		case inBacktick:
			if c == '`' {
				inBacktick = false
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '`':
			inBacktick = true
		case c == ';':
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}

// maskStringLiterals replaces the CONTENTS of single- and double-quoted strings with a filler
// character, keeping the quotes and the length so that positions and clause splitting are
// unaffected. Backtick-quoted identifiers are left alone: those are column and table names, which
// the rules are supposed to read.
func maskStringLiterals(s string) string {
	b := []byte(s)
	var inSingle, inDouble bool
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case inSingle:
			if c == '\\' && i+1 < len(b) {
				b[i+1] = 'x'
				i++
				continue
			}
			if c == '\'' {
				inSingle = false
				continue
			}
			b[i] = 'x'
		case inDouble:
			if c == '\\' && i+1 < len(b) {
				b[i+1] = 'x'
				i++
				continue
			}
			if c == '"' {
				inDouble = false
				continue
			}
			b[i] = 'x'
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		}
	}
	return string(b)
}

// splitTopLevelCommas cuts an ALTER's clause list on commas that are not inside parentheses or a
// quoted string. The parenthesis depth is what keeps DECIMAL(10,2) and ENUM('a','b') whole; the
// quote handling is what keeps a comma inside a DEFAULT string from splitting a clause in two.
func splitTopLevelCommas(s string) []string {
	var out []string
	var cur strings.Builder
	var depth int
	var inSingle, inDouble, inBacktick bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\\' && i+1 < len(s) {
				cur.WriteByte(c)
				i++
				c = s[i]
			} else if c == '\'' {
				inSingle = false
			}
		case inDouble:
			if c == '\\' && i+1 < len(s) {
				cur.WriteByte(c)
				i++
				c = s[i]
			} else if c == '"' {
				inDouble = false
			}
		case inBacktick:
			if c == '`' {
				inBacktick = false
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '`':
			inBacktick = true
		case c == '(':
			depth++
		case c == ')':
			if depth > 0 {
				depth--
			}
		case c == ',' && depth == 0:
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}

// notAColumnName are the words that can follow ADD without a column being added.
var notAColumnName = map[string]bool{
	"constraint": true, "check": true, "index": true, "key": true, "foreign": true,
	"unique": true, "primary": true, "fulltext": true, "spatial": true, "column": true,
	"partition": true, "system": true,
	// Also the words that can follow DROP without a column being dropped.
	"table": true, "if": true,
}

// addsAColumn reports whether an ALTER clause adds a column, as opposed to a constraint, an index
// or a partition — all of which can contain the words NOT NULL without introducing a column.
func addsAColumn(clause string) bool {
	return namesAColumn(reNotNullAdd.FindStringSubmatch(clause))
}

// dropsAColumn is the same question for a DROP clause: DROP TABLE, DROP INDEX and DROP PRIMARY KEY
// all begin the same way and none of them drops a column.
func dropsAColumn(clause string) bool {
	return namesAColumn(reDropColumn.FindStringSubmatch(clause))
}

// namesAColumn interprets the (quote, identifier) pair both patterns capture.
//
// A backtick is the decisive part. `key`, `index` and `check` are perfectly ordinary column names —
// a settings table almost always has one — and quoting is the author stating unambiguously that
// the word is an identifier, not syntax. Treating a quoted `key` as the KEY keyword let exactly the
// migration this rule exists to catch pass silently.
func namesAColumn(m []string) bool {
	if m == nil {
		return false
	}
	if m[1] != "" {
		return true // backtick-quoted: it is a name, whatever the word is
	}
	return !notAColumnName[strings.ToLower(m[2])]
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func excerpt(s string) string {
	if len(s) > 90 {
		return s[:90] + "…"
	}
	return s
}
