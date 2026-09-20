package authz

import "testing"

var (
	capRead   = Register("test.read", "read a thing")
	capWrite  = Register("test.write", "write a thing")
	capGlobal = Register("test.global", "a district-wide action")
)

func TestRegisterRefusesDuplicatesAndBadNames(t *testing.T) {
	mustPanic(t, func() { Register("test.read", "again") })
	mustPanic(t, func() { Register("noDot", "no module prefix") })
	mustPanic(t, func() { Register("Upper.Case", "capitals") })
	mustPanic(t, func() { Register("a..b", "empty segment") })
}

func TestCan(t *testing.T) {
	b := Bindings{
		"tech":       {capRead, capWrite, capGlobal},
		"bookkeeper": {capRead},
	}
	district := Identity{Real: "t@x", Effective: "t@x", Grants: []Grant{{Role: "tech"}}}
	atCMS := Identity{Real: "s@x", Effective: "s@x", Grants: []Grant{{Role: "tech", Scope: Scope{School: "CMS"}}}}
	viewer := Identity{Real: "a@x", Effective: "b@x", ReadOnly: true, Grants: []Grant{{Role: "tech"}}}
	nobody := Identity{Real: "n@x", Effective: "n@x"}

	cases := []struct {
		name     string
		id       Identity
		cap      Capability
		scope    Scope
		mutating bool
		want     bool
	}{
		{"district grant covers any school", district, capWrite, Scope{School: "HHS"}, true, true},
		{"district grant covers district-wide", district, capGlobal, Scope{}, true, true},
		{"school grant covers its own school", atCMS, capWrite, Scope{School: "CMS"}, true, true},
		{"school grant does NOT cover another school", atCMS, capWrite, Scope{School: "HHS"}, true, false},
		{"school grant does NOT cover a district-wide request", atCMS, capGlobal, Scope{}, true, false},
		{"read-only refuses every mutation even when held", viewer, capWrite, Scope{}, true, false},
		{"read-only still allows reads", viewer, capRead, Scope{}, false, true},
		{"no grants, no access", nobody, capRead, Scope{}, false, false},
		{"unknown capability fails closed", district, Capability("test.nothing"), Scope{}, false, false},
		{"role without the capability", Identity{Effective: "b", Grants: []Grant{{Role: "bookkeeper"}}}, capWrite, Scope{}, true, false},
	}
	for _, c := range cases {
		got := Can(c.id, b, c.cap, c.scope, c.mutating)
		if got.Allowed != c.want {
			t.Errorf("%s: got %v (%s), want %v", c.name, got.Allowed, got.Reason, c.want)
		}
		if got.Reason == "" {
			t.Errorf("%s: a decision must carry a reason", c.name)
		}
	}
}

func TestCapabilitiesIsWhatTheUIRendersFrom(t *testing.T) {
	b := Bindings{"tech": {capRead, capWrite}, "bookkeeper": {capRead, Capability("test.unregistered")}}
	id := Identity{Effective: "x", Grants: []Grant{
		{Role: "tech", Scope: Scope{School: "CMS"}},
		{Role: "bookkeeper", Scope: Scope{School: "CMS"}},
		{Role: "bookkeeper", Scope: Scope{School: "HHS"}},
	}}
	held := Capabilities(id, b)
	want := []Held{
		{capRead, Scope{School: "CMS"}}, {capRead, Scope{School: "HHS"}},
		{capWrite, Scope{School: "CMS"}},
	}
	if len(held) != len(want) {
		t.Fatalf("got %d held, want %d: %+v", len(held), len(want), held)
	}
	for i := range want {
		if held[i] != want[i] {
			t.Errorf("held[%d] = %+v, want %+v", i, held[i], want[i])
		}
	}
}

func TestImpersonating(t *testing.T) {
	if (Identity{Real: "a", Effective: "a"}).Impersonating() {
		t.Error("same real and effective is not impersonation")
	}
	if !(Identity{Real: "a", Effective: "b"}).Impersonating() {
		t.Error("different real and effective is impersonation")
	}
}

func mustPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected a panic")
		}
	}()
	f()
}
