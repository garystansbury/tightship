package identity_test

import (
	"testing"

	"github.com/garystansbury/tightship/internal/database"
	"github.com/garystansbury/tightship/internal/identity"
)

// Each module proves its own migrations are rollback-safe. `tightship check` does this across the
// whole build, but a module that breaks the rule should fail its own package's tests first — the
// failure then names the module rather than arriving as a surprise in a deployment check.
func TestMigrationsAreRollbackSafe(t *testing.T) {
	ms, err := database.Collect([]database.Source{identity.Module{}})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(ms) == 0 {
		t.Fatal("no migrations found; check the embed path")
	}
	if findings := database.Lint(ms); len(findings) > 0 {
		for _, f := range findings {
			t.Errorf("%s", f)
		}
	}
}
