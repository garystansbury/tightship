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
- [x] Sessions: server-side, one cookie for every kind of user, token stored only as a hash,
      sliding idle window under an absolute lifetime (both from config, both enforced in SQL),
      revocation by row, and a session-backed identity resolver
- [x] Local sign-in: accounts, Argon2id passwords, database-backed lockout, the auth-settings knob
      with the "local cannot be disabled until SSO has actually worked" guard, and one-time
      first-run setup

Next, in order:
1. **Roles and bindings.** `roles`, `role_grants` (with scope), `role_bindings` (role →
   capability), seeded so the first administrator actually holds capabilities. Until this lands
   `/me` returns an empty set and every capability-bearing route refuses — including the
   auth-settings screens, which is why it comes before the rest of identity.
2. **Credential store** (layer 2). AES-GCM with the master key from `secrets.master_key_*`;
   upload, test-connection, expiry, owner module, audit; the admin screen for it. Before SSO,
   because a client secret entered in the interface has to live somewhere.
3. **Sign-in with Google.** OpenID Connect for the staff domain, configured through the interface
   rather than the deployment file; a sign-in route that stores the return URL the app saved and
   sends the browser back to it. Writes `sso_proven` on the first success, which is what releases
   the guard on switching local sign-in off.
4. **The rest of the identity module.** `audit_log` with real and effective actor; account
   management screens. Impersonation: start, stop, status; lower-privilege targets only,
   read-only.
5. **First real module: Schools and Rooms.** Small enough to prove the path — migrations, routes,
   capabilities, navigation, a list page, a detail page, a form — and needed by everything after.
6. **Roles screen.** Grant roles to people with scope; bind capabilities to roles from the
   catalogue. Gate behind `roles.bind`. Only after the catalogue has settled.
7. **Job scheduler.** In-process, DB leader lease, run log, health beat per job, catch-up policy
   per job. Retention jobs are the first customers — sessions already has the sweep waiting.
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

## Open decisions

- React or Preact (same code either way; React unless bundle size on old devices bites).
- sqlc versus hand-written queries for the database layer.
- Whether impersonation ever gets write-through with attribution (the audit column exists from
  day one either way).
