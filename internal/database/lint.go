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
	reNotNullAdd = regexp.MustCompile(`(?is)\bADD\s+(?:COLUMN\s+)?` + "`?" + `\w+` + "`?" + `[^,;]*\bNOT\s+NULL\b[^,;]*`)
	reHasDefault = regexp.MustCompile(`(?is)\bDEFAULT\b|\bAUTO_INCREMENT\b`)
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
			add := func(why string) {
				out = append(out, Finding{Migration: m.ID(), Statement: excerpt(trimmed), Why: why})
			}
			switch {
			case reDropTable.MatchString(stmt):
				add("drops a table; the previous release may still read it")
			case reRename.MatchString(stmt):
				add("renames; a rename is a drop and an add to the previous release")
			case reChangeCol.MatchString(stmt):
				add("uses CHANGE, which can rename; use MODIFY to alter a type in place")
			case reDropColumn.MatchString(stmt) && strings.Contains(strings.ToUpper(stmt), "ALTER TABLE"):
				add("drops a column; stop writing it in one release and drop it in the next")
			}
			// A new NOT NULL column with no default breaks the previous release's INSERTs, which
			// do not know to supply it. With a default, the old code's INSERT still works.
			for _, addition := range reNotNullAdd.FindAllString(stmt, -1) {
				if !reHasDefault.MatchString(addition) {
					out = append(out, Finding{
						Migration: m.ID(), Statement: excerpt(collapse(addition)),
						Why: "adds a NOT NULL column with no default; the previous release's INSERTs omit it",
					})
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

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func excerpt(s string) string {
	if len(s) > 90 {
		return s[:90] + "…"
	}
	return s
}
