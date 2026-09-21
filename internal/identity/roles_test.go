package identity

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/garystansbury/tightship/internal/authz"
)

// rolesFixture is testService without the handle on the raw database, which these tests do not
// need. Built-in roles are already reconciled: testService mirrors the binary's start-up order.
func rolesFixture(t *testing.T) (context.Context, *Service) {
	t.Helper()
	ctx, s, _ := testService(t)
	return ctx, s
}

func holds(caps ...authz.Capability) map[authz.Capability]bool {
	m := map[authz.Capability]bool{}
	for _, c := range caps {
		m[c] = true
	}
	return m
}

// The built-in role must cover the whole catalogue, and must keep covering it as the catalogue
// grows — otherwise a capability shipped in a release is held by nobody on the morning of the
// upgrade, and the screen that needs it looks broken.
func TestAdministratorHoldsTheWholeCatalogue(t *testing.T) {
	ctx, s := rolesFixture(t)
	role, err := s.RoleByName(ctx, AdministratorRole)
	if err != nil {
		t.Fatal(err)
	}
	if !role.Builtin {
		t.Error("administrator is not marked built-in")
	}
	catalog := authz.Catalog()
	if len(role.Capabilities) != len(catalog) {
		t.Fatalf("administrator holds %d capabilities, catalogue has %d", len(role.Capabilities), len(catalog))
	}
	held := map[authz.Capability]bool{}
	for _, c := range role.Capabilities {
		held[c] = true
	}
	for _, c := range catalog {
		if !held[c] {
			t.Errorf("administrator does not hold %s", c)
		}
	}
}

// Reconciling has to be idempotent: it runs on every start.
func TestReconcilingBuiltinRolesIsIdempotent(t *testing.T) {
	ctx, s := rolesFixture(t)
	before, _ := s.RoleByName(ctx, AdministratorRole)
	for range 3 {
		if err := s.EnsureBuiltinRoles(ctx); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := s.RoleByName(ctx, AdministratorRole)
	if len(before.Capabilities) != len(after.Capabilities) {
		t.Errorf("capability count drifted from %d to %d across reconciles",
			len(before.Capabilities), len(after.Capabilities))
	}
}

// Reconciling removes a binding the catalogue no longer has, so administrator means exactly
// "everything this build enforces" rather than accumulating retired capabilities forever.
func TestReconcilingDropsRetiredCapabilities(t *testing.T) {
	ctx, s := rolesFixture(t)
	role, _ := s.RoleByName(ctx, AdministratorRole)
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO role_bindings (role_id, capability, bound_at) VALUES (?, 'retired.capability', ?)",
		role.ID, s.now()); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureBuiltinRoles(ctx); err != nil {
		t.Fatal(err)
	}
	after, _ := s.RoleByName(ctx, AdministratorRole)
	for _, c := range after.Capabilities {
		if c == "retired.capability" {
			t.Error("a capability the catalogue no longer has survived reconciliation")
		}
	}
}

// THE escalation guard. Without it, anybody who can edit roles is one request away from every
// capability in the catalogue.
func TestCannotBindACapabilityYouDoNotHold(t *testing.T) {
	ctx, s := rolesFixture(t)
	if _, err := s.CreateRole(ctx, "helpdesk", ""); err != nil {
		t.Fatal(err)
	}
	actor := holds(CapRoleBind, CapRoleRead, CapAccountRead)

	err := s.SetRoleCapabilities(ctx, "helpdesk",
		[]authz.Capability{CapAccountRead, CapAuthSettingsWrite}, actor, "tech@example.org")
	if !errors.Is(err, ErrWouldEscalate) {
		t.Fatalf("err = %v, want ErrWouldEscalate", err)
	}
	if !strings.Contains(err.Error(), string(CapAuthSettingsWrite)) {
		t.Errorf("refusal does not name the offending capability: %v", err)
	}
	// Nothing was written: a refused request must not half-apply.
	role, _ := s.RoleByName(ctx, "helpdesk")
	if len(role.Capabilities) != 0 {
		t.Errorf("a refused binding still wrote %d capabilities", len(role.Capabilities))
	}
}

// Delegation downwards must still work, or the guard makes role editing useless.
func TestCanBindWhatYouDoHold(t *testing.T) {
	ctx, s := rolesFixture(t)
	if _, err := s.CreateRole(ctx, "helpdesk", ""); err != nil {
		t.Fatal(err)
	}
	actor := holds(CapRoleBind, CapAccountRead, CapRoleRead)
	if err := s.SetRoleCapabilities(ctx, "helpdesk",
		[]authz.Capability{CapAccountRead, CapRoleRead}, actor, "tech@example.org"); err != nil {
		t.Fatalf("delegating a held capability was refused: %v", err)
	}
	role, _ := s.RoleByName(ctx, "helpdesk")
	if len(role.Capabilities) != 2 {
		t.Errorf("role holds %d capabilities, want 2", len(role.Capabilities))
	}
}

// Granting a role is the other way to launder capabilities: hand somebody a role holding more
// than you do, and you have created an account more powerful than your own.
func TestCannotGrantARoleHoldingMoreThanYouDo(t *testing.T) {
	ctx, s := rolesFixture(t)
	acct, err := s.CreateAccount(ctx, "tech@example.org", "Tech", "staff")
	if err != nil {
		t.Fatal(err)
	}
	_ = acct
	actor := holds(CapRoleGrant, CapAccountRead)

	err = s.GrantRole(ctx, "tech@example.org", AdministratorRole, authz.Scope{}, actor, "tech@example.org")
	if !errors.Is(err, ErrWouldEscalate) {
		t.Errorf("err = %v, want ErrWouldEscalate", err)
	}
}

// Every problem at once, so a role is fixed in one edit.
func TestEscalationRefusalNamesEveryOffendingCapability(t *testing.T) {
	ctx, s := rolesFixture(t)
	if _, err := s.CreateRole(ctx, "helpdesk", ""); err != nil {
		t.Fatal(err)
	}
	err := s.SetRoleCapabilities(ctx, "helpdesk",
		[]authz.Capability{CapAuthSettingsWrite, CapPasswordSetOther, CapAccountDisable},
		holds(CapRoleBind), "tech@example.org")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	for _, want := range []authz.Capability{CapAuthSettingsWrite, CapPasswordSetOther, CapAccountDisable} {
		if !strings.Contains(err.Error(), string(want)) {
			t.Errorf("refusal omits %s: %v", want, err)
		}
	}
}

// A capability the catalogue does not know cannot be bound: bindings are data, but they are data
// drawn from a fixed set, and a typo must not become a permission nobody can find.
func TestCannotBindAnUnknownCapability(t *testing.T) {
	ctx, s := rolesFixture(t)
	if _, err := s.CreateRole(ctx, "helpdesk", ""); err != nil {
		t.Fatal(err)
	}
	err := s.SetRoleCapabilities(ctx, "helpdesk",
		[]authz.Capability{"not.a.real.capability"}, holds(CapRoleBind), "gary@example.org")
	if err == nil || !strings.Contains(err.Error(), "no such capability") {
		t.Errorf("err = %v, want a complaint about an unknown capability", err)
	}
}

// The built-in role cannot be edited or deleted through the ordinary path, or somebody removes
// the capability that lets them put it back.
func TestBuiltinRoleIsProtected(t *testing.T) {
	ctx, s := rolesFixture(t)
	all := map[authz.Capability]bool{}
	for _, c := range authz.Catalog() {
		all[c] = true
	}
	if err := s.SetRoleCapabilities(ctx, AdministratorRole,
		[]authz.Capability{CapAccountRead}, all, "gary@example.org"); !errors.Is(err, ErrRoleProtected) {
		t.Errorf("editing the built-in role: err = %v, want ErrRoleProtected", err)
	}
	if err := s.DeleteRole(ctx, AdministratorRole, "gary@example.org"); !errors.Is(err, ErrRoleProtected) {
		t.Errorf("deleting the built-in role: err = %v, want ErrRoleProtected", err)
	}
}

// The same failure as switching off every sign-in method, reached from the other side.
func TestCannotRevokeTheLastAdministrator(t *testing.T) {
	ctx, s := rolesFixture(t)
	first, err := s.CreateAccount(ctx, "gary@example.org", "Gary", "staff")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.GrantAdministrator(ctx, first.ID, "setup"); err != nil {
		t.Fatal(err)
	}
	grants, err := s.ListGrants(ctx)
	if err != nil || len(grants) != 1 {
		t.Fatalf("expected one grant, got %d (%v)", len(grants), err)
	}
	if err := s.RevokeGrant(ctx, grants[0].ID, "gary@example.org"); !errors.Is(err, ErrLastAdministrator) {
		t.Fatalf("err = %v, want ErrLastAdministrator", err)
	}

	// With a second administrator, revoking the first is fine.
	second, _ := s.CreateAccount(ctx, "deputy@example.org", "Deputy", "staff")
	if err := s.GrantAdministrator(ctx, second.ID, "gary@example.org"); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeGrant(ctx, grants[0].ID, "gary@example.org"); err != nil {
		t.Errorf("revoking one of two administrators was refused: %v", err)
	}
}

// A disabled administrator does not count towards the last-administrator check, or disabling
// everybody leaves a deployment that looks administrable and is not.
func TestDisabledAdministratorsDoNotCount(t *testing.T) {
	ctx, s := rolesFixture(t)
	a, _ := s.CreateAccount(ctx, "gary@example.org", "", "staff")
	b, _ := s.CreateAccount(ctx, "deputy@example.org", "", "staff")
	_ = s.GrantAdministrator(ctx, a.ID, "setup")
	_ = s.GrantAdministrator(ctx, b.ID, "setup")
	if _, err := s.db.ExecContext(ctx, "UPDATE accounts SET disabled_at = ? WHERE id = ?", s.now(), b.ID); err != nil {
		t.Fatal(err)
	}
	grants, _ := s.ListGrants(ctx)
	var target int64
	for _, g := range grants {
		if g.Email == "gary@example.org" {
			target = g.ID
		}
	}
	if err := s.RevokeGrant(ctx, target, "gary@example.org"); !errors.Is(err, ErrLastAdministrator) {
		t.Errorf("a disabled administrator was counted as one: err = %v", err)
	}
}

// Grants carry scope, and scope has to survive the round trip or a school-bounded grant silently
// becomes district-wide.
func TestGrantsCarryScope(t *testing.T) {
	ctx, s := rolesFixture(t)
	acct, _ := s.CreateAccount(ctx, "tech@example.org", "Tech", "staff")
	_ = acct
	if _, err := s.CreateRole(ctx, "technician", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SetRoleCapabilities(ctx, "technician",
		[]authz.Capability{CapAccountRead}, holds(CapAccountRead), "gary@example.org"); err != nil {
		t.Fatal(err)
	}
	scope := authz.Scope{School: "NHS"}
	if err := s.GrantRole(ctx, "tech@example.org", "technician", scope, holds(CapAccountRead), "gary@example.org"); err != nil {
		t.Fatal(err)
	}
	got, err := s.GrantsFor(ctx, "tech@example.org")
	if err != nil || len(got) != 1 {
		t.Fatalf("got %d grants (%v)", len(got), err)
	}
	if got[0].Scope.School != "NHS" {
		t.Errorf("scope.School = %q, want NHS", got[0].Scope.School)
	}

	// And the scope actually bounds: a school-scoped grant does not satisfy a district-wide ask.
	bindings, _ := s.Bindings(ctx)
	id := authz.Identity{Real: "tech@example.org", Effective: "tech@example.org", Grants: got}
	if d := authz.Can(id, bindings, CapAccountRead, authz.Scope{School: "NHS"}, false); !d.Allowed {
		t.Errorf("school-scoped request refused: %s", d.Reason)
	}
	if d := authz.Can(id, bindings, CapAccountRead, authz.Scope{}, false); d.Allowed {
		t.Error("a school-scoped grant satisfied a district-wide request")
	}
	if d := authz.Can(id, bindings, CapAccountRead, authz.Scope{School: "EHS"}, false); d.Allowed {
		t.Error("a grant for one school satisfied a request about another")
	}
}

// A disabled account holds nothing, whatever it was granted.
func TestDisabledAccountsHoldNoGrants(t *testing.T) {
	ctx, s := rolesFixture(t)
	acct, _ := s.CreateAccount(ctx, "gone@example.org", "", "staff")
	_ = s.GrantAdministrator(ctx, acct.ID, "setup")
	if got, _ := s.GrantsFor(ctx, "gone@example.org"); len(got) != 1 {
		t.Fatalf("setup: expected one grant, got %d", len(got))
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE accounts SET disabled_at = ? WHERE id = ?", s.now(), acct.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GrantsFor(ctx, "gone@example.org"); len(got) != 0 {
		t.Errorf("a disabled account still holds %d grant(s)", len(got))
	}
}

// Deleting a role that people still hold would strip access silently.
func TestCannotDeleteARoleStillGranted(t *testing.T) {
	ctx, s := rolesFixture(t)
	acct, _ := s.CreateAccount(ctx, "tech@example.org", "", "staff")
	_ = acct
	if _, err := s.CreateRole(ctx, "technician", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.GrantRole(ctx, "tech@example.org", "technician", authz.Scope{}, holds(), "gary@example.org"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRole(ctx, "technician", "gary@example.org"); err == nil {
		t.Error("deleted a role that was still granted")
	}
}

// Bindings feed authz.Can directly, so the map has to come back shaped as that expects.
func TestBindingsMapRoleNamesToCapabilities(t *testing.T) {
	ctx, s := rolesFixture(t)
	b, err := s.Bindings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	caps, ok := b[AdministratorRole]
	if !ok {
		t.Fatal("bindings do not contain the administrator role")
	}
	if len(caps) != len(authz.Catalog()) {
		t.Errorf("administrator maps to %d capabilities, catalogue has %d", len(caps), len(authz.Catalog()))
	}
}
