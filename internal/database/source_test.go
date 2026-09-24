package database_test

import (
	"testing"

	"github.com/garystansbury/tightship/internal/database"
	"github.com/garystansbury/tightship/internal/module"
)

// A module must be usable as a migration source without an adapter. If this stops compiling,
// the two contracts have drifted and every module would need wrapping at the call site.
func TestModuleSatisfiesSource(t *testing.T) {
	var m module.Module
	var _ database.Source = m
}
