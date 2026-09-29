# Roadmap

Phases are ordered so the architecture is proven end to end on something small before the largest
module is attempted, and so the "one interface for every kind of user" win lands as early as it
can. The production suite this replaces keeps running beside it; each module cuts over when it is
ready, and a cut-over module's tables get one writer.

## Phase 1 — Foundations (in progress)

Done:
- [x] Capability layer: `authz.Register`, `Can`, `Capabilities`, scope, read-only overlay
- [x] Route table with the mutating-route rule; `GET /api/v1/me`
- [x] Strict deployment-file loader with all-problems validation
- [x] Module contract
- [x] Embedded web app; Vite + React + TypeScript shell rendering navigation from `/me`
- [x] CI; tag → release with static binary, checksum, example config, systemd unit
- [x] Database layer: pool sized from config, per-module embedded migrations applied at start,
      `schema_migrations` with checksums and partial-failure detection, advisory lock, and a
      rollback-safety lint that `tightship check` runs

Next, in order:
1. **Sessions.** Server-side, stored in the database, one cookie for every kind of user. Sliding
   idle window plus absolute lifetime, both from config. Revocation by row.
2. **Sign-in.** Google OpenID Connect for the staff domain (the first identity kind); a sign-in
   route that stores the return URL the app saved and sends the browser back to it. The identity
   resolver replaces the debug header outside development.
3. **Identity module.** `users`, `roles`, `role_grants` (with scope), `role_bindings`
   (role → capability), `audit_log` with real and effective actor. `/me` starts returning real
   capabilities. Impersonation: start, stop, status; lower-privilege targets only; read-only.
4. **Credential store** (layer 2). AES-GCM with the master key from `secrets.master_key_*`;
   upload, test-connection, expiry, owner module, audit; the admin screen for it.
5. **First real module: Schools and Rooms.** Small enough to prove the path — migrations, routes,
   capabilities, navigation, a list page, a detail page, a form — and needed by everything after.
6. **Roles screen.** Grant roles to people with scope; bind capabilities to roles from the
   catalogue. Gate behind `roles.bind`. Only after the catalogue has settled.
7. **Job scheduler.** In-process, DB leader lease, run log, health beat per job, catch-up policy
   per job. Retention jobs are the first customers.
8. **Demo mode.** Integrations behind interfaces with fixture implementations, a seed, and a
   way to view the app as each seeded role. This is also the screenshot-test harness.
9. **Observability.** Structured logs with request IDs; a health endpoint that proves the
   credential store as well as the database. (`/healthz` already proves the database: it reads
   `schema_migrations`, because `SELECT 1` stays green against a server the application cannot
   read a row from.)

## Phase 2 — Help Desk

Tickets, queues, categories, comments, attachments; the submitter view (any signed-in staff
member, no roles) replaces the separate intake app; invoices for bookkeepers in the same shell; a
contractor guest identity (magic link plus one-time code). Largest module, so second, not first.

## Phase 3 — Chromebooks

Device and Student object pages; assign, release, move, labels, history; the room navigator and
per-room assigner grants; a scanner component so the phone-width layout replaces the separate
technician app. Carries the fleet invariants (the assignment store is authoritative; the ledger
mirrors every device write).

## Phase 4 — Identity and the IdP

User management, staff account lifecycle, recovery-key lookup; the student MFA roster, activity,
sessions and policy screens; then the student sign-in flow itself moves into this binary behind
the existing SAML terminator, factor by factor, pilot organisational unit first. Native SAML is an
optional later step, only after a term of the flow running here.

## Phase 5 — Operations

DHCP reservations, per-user Wi-Fi keys, managed file transfer, roster sync, job schedule, status
and observability, reporting and published queries. Then the original suite is a set of links,
and is retired.

## Coexistence with the suite being replaced

Recorded in [coexistence](coexistence.md), and load-bearing for every module after the first.

Both interfaces stay functional indefinitely; people move across because TightShip is the better
place to work, not because a date arrived. A module talks to a store interface with a native and a
legacy implementation, exactly one system writes a given domain at a time, and when a domain flips,
the legacy suite becomes a **client** of TightShip for it — reading its tables directly with a
SELECT-only account where one connection can reach both, writing through its API.

Unbuilt, and forced by the first module that needs live data from an incumbent system: the
per-module backend in config; the rule that a module's configured backend, never the presence of
its tables, decides authority; the read-only legacy pool and an authenticated health endpoint that
names dependencies the public one must not; `Identity.Via` and a service account that may assert an
actor; and an API versioning policy, since another system depending on the API makes it a
compatibility surface rather than an internal detail.

The adapters themselves are not built here. A deployment with a legacy suite to migrate composes
its own binary in a private repository that imports this one as a library — composition, not a
fork, so D7 holds.

## Open decisions

- React or Preact (same code either way; React unless bundle size on old devices bites).
- sqlc versus hand-written queries for the database layer.
- Whether impersonation ever gets write-through with attribution (the audit column exists from
  day one either way).
