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
	reDropTable  = regexp.MustCompile(`(?is)\bDROP\s+TABLE\b`)
	reDropColumn = regexp.MustCompile(`(?is)\bDROP\s+(?:COLUMN\s+)?` + "`?" + `\w+` + "`?" + `\s*(?:,|;|$)`)
	reRename     = regexp.MustCompile(`(?is)\bRENAME\s+(?:TABLE|COLUMN|TO|AS)\b`)
	reChangeCol  = regexp.MustCompile(`(?is)\bCHANGE\s+(?:COLUMN\s+)?` + "`?" + `\w+`)
	// Matched against a single clause, not a whole statement. An earlier version scanned the
	// statement with `[^,;]*` around NOT NULL, which cannot cross a comma — so any column whose
	// TYPE contains one escaped the rule entirely. DECIMAL(10,2) and ENUM('a','b') are the common
	// cases, and both went unreported while VARCHAR(8) was caught, so the guarantee this lint
	// advertises quietly did not hold for the two types most likely to carry money and status.
	reNotNullAdd  = regexp.MustCompile(`(?is)\bADD\s+(?:COLUMN\s+)?` + "`?" + `\w+`)
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
			isAlter := strings.Contains(strings.ToUpper(stmt), "ALTER TABLE")

			// One ALTER may carry several clauses, and each is a separate decision: ADD this,
			// DROP that. Splitting on top-level commas — respecting parentheses and quotes, so a
			// DECIMAL(10,2) or an ENUM('a','b') stays in one piece — lets each clause be judged on
			// its own, and lets a statement report every problem it has rather than only the first.
			clauses := []string{stmt}
			if isAlter {
				clauses = splitTopLevelCommas(stmt)
			}
			for _, clause := range clauses {
				c := collapse(clause)
				if c == "" {
					continue
				}
				add := func(why string) {
					out = append(out, Finding{Migration: m.ID(), Statement: excerpt(c), Why: why})
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
				if isAlter && reDropColumn.MatchString(clause) {
					add("drops a column; stop writing it in one release and drop it in the next")
				}
				// A new NOT NULL column with no default breaks the previous release's INSERTs,
				// which do not know to supply it. With a default, the old code's INSERT still
				// works. Checked per clause, so a comma inside the column's type cannot hide it.
				if isAlter && reNotNullAdd.MatchString(clause) &&
					reNotNullWord.MatchString(clause) && !reHasDefault.MatchString(clause) {
					add("adds a NOT NULL column with no default; the previous release's INSERTs omit it")
				}
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

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func excerpt(s string) string {
	if len(s) > 90 {
		return s[:90] + "…"
	}
	return s
}
