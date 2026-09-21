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
- [x] Roles, bindings and grants: capabilities bound to roles, roles granted with scope, the
      administrator role reconciled to the catalogue at every start, and the escalation guard —
      you cannot give away a capability you do not hold. `/me` now returns real capabilities and
      the auth-settings screens are reachable.

Next, in order:
1. **Credential store** (layer 2). AES-GCM with the master key from `secrets.master_key_*`;
   upload, test-connection, expiry, owner module, audit; the admin screen for it. Before SSO,
   because a client secret entered in the interface has to live somewhere.
2. **SSO: Google and Microsoft Entra.** Not one provider but a set — both are OpenID Connect, and
   they differ in discovery URL, claim names and failure modes rather than in protocol. Configured
   through the interface with a wizard per provider, not in the deployment file. A provider is
   `proven` the first time it actually carries a sign-in, and that is what releases the guard on
   switching local sign-in off.

   Three things the design has to get right, all of them Entra-specific:
   - **The tenant is pinned and the `tid` claim is validated against it.** Pointing Entra at
     `/common/` lets any Microsoft account in any tenant sign in. It is the most common Entra
     misconfiguration, it passes testing perfectly, and it is a full authentication bypass.
   - **Claim mapping is per provider.** Google reliably gives `email` and `email_verified`; Entra
     often gives only `preferred_username` or `upn`, neither guaranteed to be an email address nor
     verified. A shared claim map would quietly trust the wrong field.
   - **Linking pins `sub`, not email.** Matching an OIDC identity to an existing account by email
     alone means anyone who can make an IdP assert that address becomes that user. The provider's
     subject is recorded on first link, and the domain must be allow-listed.
3. **The rest of the identity module.** `audit_log` with real and effective actor; account
   management screens. Impersonation: start, stop, status; lower-privilege targets only,
   read-only.
4. **First real module: Schools and Rooms.** Small enough to prove the path — migrations, routes,
   capabilities, navigation, a list page, a detail page, a form — and needed by everything after.
5. **Roles screen.** Grant roles to people with scope; bind capabilities to roles from the
   catalogue. Gate behind `roles.bind`. Only after the catalogue has settled.
6. **Job scheduler.** In-process, DB leader lease, run log, health beat per job, catch-up policy
   per job. Retention jobs are the first customers — sessions already has the sweep waiting.
7. **Demo mode.** Integrations behind interfaces with fixture implementations, a seed, and a
   way to view the app as each seeded role. This is also the screenshot-test harness.
8. **Observability.** Structured logs with request IDs; a health endpoint that proves the
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
