# RBAC: the capability layer

Status: **adopted as the permission model; implemented in `internal/authz` and enforced by
`internal/httpapi`.** This page keeps the reasoning; the code keeps the rules.

Originally written for the mid-2026 prototype, when the production suite it generalises had
about 75 hardcoded role-check call sites (`require_role('admin')`, `is_admin || is_tech`, …)
spread across its endpoints. Role *resolution* was centralised there; the role→capability
mapping was not. That is the gap this closes.

## Why a capability layer at all

The original assessment (a single internal tool, eight stable roles, a handful of admins) said a
capability layer was over-engineering. Productisation breaks that premise: a deploying district
must be able to re-map *who can do what* without forking code.

## The line that holds

The permission **vocabulary is part of the application contract, enforced by code** — a
capability means something only because a route calls for it. So:

- **In scope (data, editable per deployment):** which roles hold which *existing* capabilities;
  composing custom roles from *existing* capabilities.
- **Out of scope (always a code change, by definition):** inventing new capabilities. A
  capability nothing enforces is meaningless, so a runtime "create a permission" feature is
  incoherent, not merely risky. It is not built and will not be.

Two policies stay **first-class context, not capabilities** — a naive permission-flag model
loses both:

1. **Scope** — a grant may be bounded to a school, a room or a queue. `Can` refuses a bounded
   grant for a request outside its bound, and for a request that names no bound at all
   (district-wide actions need district-wide grants).
2. **Read-only overlay** — impersonation, and roles that exist to observe, force read-only even
   when the effective identity co-holds a mutating capability. This is an overlay over
   capabilities, not a capability itself.

`Can` answers "is this action permitted for this identity, here?"; scope and read-only are
separate inputs it combines with the bindings.

## How it is built

- **Catalogue in code.** `authz.Register("ticket.comment", "…")` in a module's `init()`. Names
  are `module.action`; duplicates panic.
- **Bindings as data.** `role_bindings(role, capability)`, seeded by migration from the
  deployment's chosen defaults, edited in the Roles screen (gated by `roles.bind`, audited).
- **One decision point.** `httpapi.Router.Handle(method, pattern, capability, scope, handler)`.
  A mutating route with no capability panics at registration; a test walks the table.
- **The UI renders from the same answer.** `GET /api/v1/me` returns the effective identity's
  capabilities with their scopes. Navigation and controls appear when a capability is held;
  the route refuses the same capability; a rendered-but-refused control cannot exist.

## What not to do

- Do not add a role name to a handler or to the UI. Ask for a capability.
- Do not add a second allow-list "for safety". The route table is the list.
- Do not wrap a UI around the binding editor until the catalogue has settled in real use —
  a churning vocabulary behind a management screen trains operators to distrust it.
