package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/garystansbury/tightship/internal/authz"
)

// Roles, bindings and grants.
//
// The escalation guard is the part worth reading. Anybody who can edit roles can, without it,
// give themselves every capability in the catalogue: bind auth.settings.write to a role they
// hold, and they are an administrator. The rule that prevents it is that you cannot give away a
// capability you do not hold yourself — which makes role editing a way to delegate authority
// downwards and never a way to acquire it.
//
// That rule is enforced here rather than in a handler, so a future CLI or installer is bound by
// it too, and so a reviewer can find the whole of it in one place.

// AdministratorRole is the one built-in role. It is reconciled against the capability catalogue
// at every start, so a capability shipped in a new release is held by somebody on the morning of
// the upgrade rather than after a manual step nobody remembers.
const AdministratorRole = "administrator"

// Role is a named set of capabilities.
type Role struct {
	ID           int64              `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Builtin      bool               `json:"builtin"`
	Capabilities []authz.Capability `json:"capabilities"`
}

// Grant is a role held by an account, optionally bounded.
type Grant struct {
	ID        int64       `json:"id"`
	AccountID int64       `json:"account_id"`
	Email     string      `json:"email"`
	Role      string      `json:"role"`
	Scope     authz.Scope `json:"scope"`
	GrantedAt time.Time   `json:"granted_at"`
	GrantedBy string      `json:"granted_by"`
}

var (
	ErrRoleNotFound  = errors.New("identity: no such role")
	ErrRoleProtected = errors.New("identity: that role is built in and cannot be changed here")

	// ErrWouldEscalate is the refusal that keeps role editing a delegation mechanism rather than
	// an escalation one.
	ErrWouldEscalate = errors.New("identity: you cannot grant a capability you do not hold yourself")

	// ErrLastAdministrator stops the deployment being left with nobody who can administer it —
	// the same failure mode as switching off every sign-in method, arrived at from the other side.
	ErrLastAdministrator = errors.New("identity: that would leave nobody holding the administrator role")
)

// EnsureBuiltinRoles creates the administrator role if absent and reconciles its bindings to the
// current catalogue. Called at start, after migrations, before the listener.
//
// Reconciling ADDS capabilities the catalogue has gained and REMOVES ones it no longer knows,
// which is safe because an unknown capability is inert anyway — authz.Capabilities skips it. The
// effect is that administrator always means "everything this build can enforce".
func (s *Service) EnsureBuiltinRoles(ctx context.Context) error {
	now := s.now()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO roles (name, description, builtin, created_at, updated_at)
		VALUES (?, 'Holds every capability this build enforces.', 1, ?, ?)
		ON DUPLICATE KEY UPDATE builtin = 1, updated_at = VALUES(updated_at)`,
		AdministratorRole, now, now); err != nil {
		return fmt.Errorf("identity: ensure administrator role: %w", err)
	}
	role, err := s.RoleByName(ctx, AdministratorRole)
	if err != nil {
		return err
	}

	catalog := authz.Catalog()
	held := map[authz.Capability]bool{}
	for _, c := range role.Capabilities {
		held[c] = true
	}
	want := map[authz.Capability]bool{}
	for _, c := range catalog {
		want[c] = true
	}

	for _, c := range catalog {
		if held[c] {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO role_bindings (role_id, capability, bound_at, bound_by)
			VALUES (?, ?, ?, 'tightship')
			ON DUPLICATE KEY UPDATE capability = capability`, role.ID, string(c), now); err != nil {
			return fmt.Errorf("identity: bind %s to administrator: %w", c, err)
		}
	}
	for _, c := range role.Capabilities {
		if want[c] {
			continue
		}
		if _, err := s.db.ExecContext(ctx,
			"DELETE FROM role_bindings WHERE role_id = ? AND capability = ?", role.ID, string(c)); err != nil {
			return fmt.Errorf("identity: unbind %s from administrator: %w", c, err)
		}
	}
	return nil
}

// GrantAdministrator gives an account the administrator role, district-wide. Used by first-run
// setup: an administrator who holds no capabilities cannot configure anything, which would make
// the setup wizard produce an account that can sign in and do nothing.
func (s *Service) GrantAdministrator(ctx context.Context, accountID int64, by string) error {
	role, err := s.RoleByName(ctx, AdministratorRole)
	if err != nil {
		return err
	}
	return s.grant(ctx, accountID, role.ID, authz.Scope{}, by)
}

// ---------------------------------------------------------------------------
// Reading

// RoleByName loads a role and its bindings.
func (s *Service) RoleByName(ctx context.Context, name string) (*Role, error) {
	var r Role
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, builtin FROM roles WHERE name = ?", name).
		Scan(&r.ID, &r.Name, &r.Description, &r.Builtin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("identity: role lookup: %w", err)
	}
	r.Capabilities, err = s.roleCapabilities(ctx, r.ID)
	return &r, err
}

func (s *Service) roleCapabilities(ctx context.Context, roleID int64) ([]authz.Capability, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT capability FROM role_bindings WHERE role_id = ? ORDER BY capability", roleID)
	if err != nil {
		return nil, fmt.Errorf("identity: role bindings: %w", err)
	}
	defer rows.Close()
	var out []authz.Capability
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, authz.Capability(c))
	}
	return out, rows.Err()
}

// Roles lists every role with its capabilities.
func (s *Service) Roles(ctx context.Context) ([]Role, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, builtin FROM roles ORDER BY builtin DESC, name")
	if err != nil {
		return nil, fmt.Errorf("identity: list roles: %w", err)
	}
	defer rows.Close()
	var out []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Builtin); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Capabilities, err = s.roleCapabilities(ctx, out[i].ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Bindings returns the whole role→capability map, which is what authz.Can decides with. It is
// read once per request; the table is small and stays in the buffer pool, and a cache here would
// mean a revoked capability kept working for as long as the cache lived.
func (s *Service) Bindings(ctx context.Context) (authz.Bindings, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.name, b.capability
		FROM role_bindings b JOIN roles r ON r.id = b.role_id`)
	if err != nil {
		return nil, fmt.Errorf("identity: bindings: %w", err)
	}
	defer rows.Close()
	out := authz.Bindings{}
	for rows.Next() {
		var role, cap string
		if err := rows.Scan(&role, &cap); err != nil {
			return nil, err
		}
		out[role] = append(out[role], authz.Capability(cap))
	}
	return out, rows.Err()
}

// GrantsFor returns an account's grants, for the identity resolver.
func (s *Service) GrantsFor(ctx context.Context, email string) ([]authz.Grant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.name, g.school_code, g.room_guid, g.queue
		FROM role_grants g
		JOIN roles r ON r.id = g.role_id
		JOIN accounts a ON a.id = g.account_id
		WHERE a.email = ? AND a.disabled_at IS NULL`, normaliseEmail(email))
	if err != nil {
		return nil, fmt.Errorf("identity: grants: %w", err)
	}
	defer rows.Close()
	var out []authz.Grant
	for rows.Next() {
		var g authz.Grant
		if err := rows.Scan(&g.Role, &g.Scope.School, &g.Scope.Room, &g.Scope.Queue); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// ListGrants returns every grant, for the roles screen.
func (s *Service) ListGrants(ctx context.Context) ([]Grant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.account_id, a.email, r.name, g.school_code, g.room_guid, g.queue,
		       g.granted_at, g.granted_by
		FROM role_grants g
		JOIN roles r ON r.id = g.role_id
		JOIN accounts a ON a.id = g.account_id
		ORDER BY a.email, r.name`)
	if err != nil {
		return nil, fmt.Errorf("identity: list grants: %w", err)
	}
	defer rows.Close()
	var out []Grant
	for rows.Next() {
		var g Grant
		if err := rows.Scan(&g.ID, &g.AccountID, &g.Email, &g.Role,
			&g.Scope.School, &g.Scope.Room, &g.Scope.Queue, &g.GrantedAt, &g.GrantedBy); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Writing, and the guard on it

// CreateRole adds an empty role. Capabilities are bound separately, so the escalation check has
// one place to live rather than two.
func (s *Service) CreateRole(ctx context.Context, name, description string) (*Role, error) {
	if name == "" {
		return nil, errors.New("identity: a role needs a name")
	}
	now := s.now()
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO roles (name, description, builtin, created_at, updated_at) VALUES (?, ?, 0, ?, ?)",
		name, description, now, now); err != nil {
		return nil, fmt.Errorf("identity: create role: %w", err)
	}
	return s.RoleByName(ctx, name)
}

// SetRoleCapabilities replaces a role's bindings, refusing any the actor does not hold.
//
// actorHolds is the set the actor actually has, taken from the request's identity — not from a
// role name, and not re-derived here, so this function cannot be fooled by a caller that
// describes itself generously.
func (s *Service) SetRoleCapabilities(ctx context.Context, roleName string, want []authz.Capability, actorHolds map[authz.Capability]bool, actor string) error {
	role, err := s.RoleByName(ctx, roleName)
	if err != nil {
		return err
	}
	if role.Builtin {
		return ErrRoleProtected
	}

	// Only capabilities the catalogue knows, and only ones the actor holds. Both checks report
	// everything wrong at once: fixing a role one refusal at a time is how somebody gives up and
	// asks for the administrator role instead.
	var unknown, forbidden []string
	for _, c := range want {
		if !authz.Known(c) {
			unknown = append(unknown, string(c))
			continue
		}
		if !actorHolds[c] {
			forbidden = append(forbidden, string(c))
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("identity: no such capability: %v", unknown)
	}
	if len(forbidden) > 0 {
		sort.Strings(forbidden)
		return fmt.Errorf("%w: %v", ErrWouldEscalate, forbidden)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("identity: set capabilities: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM role_bindings WHERE role_id = ?", role.ID); err != nil {
		return fmt.Errorf("identity: clear bindings: %w", err)
	}
	now := s.now()
	for _, c := range want {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO role_bindings (role_id, capability, bound_at, bound_by) VALUES (?, ?, ?, ?)",
			role.ID, string(c), now, actor); err != nil {
			return fmt.Errorf("identity: bind %s: %w", c, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("identity: set capabilities: %w", err)
	}
	s.log.Info("role capabilities changed", "role", roleName, "by", actor, "count", len(want))
	return nil
}

// GrantRole gives a role to an account within a scope.
//
// The same escalation rule applies: you cannot hand somebody a role holding capabilities you do
// not hold. Otherwise "grant a role" is a way to launder capabilities you were refused when you
// tried to bind them directly.
func (s *Service) GrantRole(ctx context.Context, email, roleName string, scope authz.Scope, actorHolds map[authz.Capability]bool, actor string) error {
	acct, err := s.AccountByEmail(ctx, email)
	if err != nil {
		return err
	}
	role, err := s.RoleByName(ctx, roleName)
	if err != nil {
		return err
	}
	var forbidden []string
	for _, c := range role.Capabilities {
		if !authz.Known(c) {
			// Inert in this build; handing it over grants nothing, so it cannot escalate.
			continue
		}
		if !actorHolds[c] {
			forbidden = append(forbidden, string(c))
		}
	}
	if len(forbidden) > 0 {
		sort.Strings(forbidden)
		return fmt.Errorf("%w: %s holds %v", ErrWouldEscalate, roleName, forbidden)
	}
	return s.grant(ctx, acct.ID, role.ID, scope, actor)
}

func (s *Service) grant(ctx context.Context, accountID, roleID int64, scope authz.Scope, actor string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO role_grants (account_id, role_id, school_code, room_guid, queue, granted_at, granted_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE granted_at = VALUES(granted_at), granted_by = VALUES(granted_by)`,
		accountID, roleID, scope.School, scope.Room, scope.Queue, s.now(), actor)
	if err != nil {
		return fmt.Errorf("identity: grant role: %w", err)
	}
	s.log.Info("role granted", "account", accountID, "role", roleID, "by", actor,
		"school", scope.School, "room", scope.Room, "queue", scope.Queue)
	return nil
}

// RevokeGrant removes one grant, refusing to remove the last administrator.
func (s *Service) RevokeGrant(ctx context.Context, grantID int64, actor string) error {
	var roleName string
	err := s.db.QueryRowContext(ctx, `
		SELECT r.name FROM role_grants g JOIN roles r ON r.id = g.role_id WHERE g.id = ?`, grantID).
		Scan(&roleName)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("identity: no such grant")
	}
	if err != nil {
		return fmt.Errorf("identity: revoke grant: %w", err)
	}
	if roleName == AdministratorRole {
		n, err := s.countAdministrators(ctx)
		if err != nil {
			return err
		}
		if n <= 1 {
			// The same failure as switching off every sign-in method, reached from the other side:
			// a deployment nobody can administer needs a database console to recover.
			return ErrLastAdministrator
		}
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM role_grants WHERE id = ?", grantID); err != nil {
		return fmt.Errorf("identity: revoke grant: %w", err)
	}
	s.log.Info("role grant revoked", "grant", grantID, "role", roleName, "by", actor)
	return nil
}

func (s *Service) countAdministrators(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT g.account_id)
		FROM role_grants g
		JOIN roles r ON r.id = g.role_id
		JOIN accounts a ON a.id = g.account_id
		WHERE r.name = ? AND a.disabled_at IS NULL`, AdministratorRole).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("identity: count administrators: %w", err)
	}
	return n, nil
}

// DeleteRole removes a role. Built-in roles are refused, and so is a role still granted to
// somebody — a delete that silently strips people's access is not a delete anyone meant.
func (s *Service) DeleteRole(ctx context.Context, name, actor string) error {
	role, err := s.RoleByName(ctx, name)
	if err != nil {
		return err
	}
	if role.Builtin {
		return ErrRoleProtected
	}
	var granted int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM role_grants WHERE role_id = ?", role.ID).Scan(&granted); err != nil {
		return fmt.Errorf("identity: delete role: %w", err)
	}
	if granted > 0 {
		return fmt.Errorf("identity: %s is still granted to %d account(s); revoke those first", name, granted)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM roles WHERE id = ?", role.ID); err != nil {
		return fmt.Errorf("identity: delete role: %w", err)
	}
	s.log.Info("role deleted", "role", name, "by", actor)
	return nil
}
