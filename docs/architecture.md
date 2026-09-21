# Architecture

TightShip is one process: an HTTP API, an embedded single-page web app, database migrations and a
job scheduler, built into a single static binary. This page is the shape of that process and the
decisions behind it. It is written for someone about to add a module.

## Decisions

| | Decision | Why |
|---|---|---|
| D1 | **Go, one static binary.** | A deployment is a binary and a config file. No web-executable scripts, no include paths, no cron lines that die silently: jobs are goroutines under one leader lease with structured logs. |
| D2 | **One TypeScript single-page app.** | Every tool is a route in one app sharing one component library and one data layer. No iframes, no injected fragments, no side channels for state. |
| D3 | **One interface for every kind of user**, rendered from the capability set. | A submitter and an administrator differ only in what `GET /api/v1/me` returns. Impersonation is "swap the effective identity and re-render". |
| D4 | **Capabilities in code, bindings as data, one decision point.** | The route table declares what each endpoint needs; `authz.Can` decides; the UI asks the same function through `/me`. |
| D5 | **Object pages, not tool pages.** | Devices, people, tickets, rooms and schools are the destinations. "Where do I go to do X" stops being a question when you go to the thing. |
| D6 | **Districts are configuration and modules.** | Four layers, below. Nothing in the core knows a vendor or a district name. |
| D7 | **A deployment pins a tag.** | Fixes go upstream first and arrive as the next release. No forks. |

## The request path

```
request ──► route table ──► authz.Can(identity, bindings, capability, scope, mutating) ──► handler
             (declares the           │                                                     (audits real → effective)
              capability)            └──► GET /api/v1/me returns the same capability set ──► the UI renders from it
```

- **Identity** is resolved once per request: `Real` (who signed in), `Effective` (who they act
  as), their `Grants` (role, optionally bounded to a school, room or queue) and a `ReadOnly`
  overlay. Every permission question is asked of Effective; every audit line names Real.
- **Capabilities** are a fixed catalogue registered by modules at init (`authz.Register`). A
  capability means something only because a route enforces it, so there is no runtime "create a
  permission" and never will be.
- **Bindings** map roles to capabilities and live in the database, seeded by migration and edited
  in the Roles screen. A district composes its own roles from the catalogue without a code change.
- **Scope** is context, not a capability. A grant bounded to a school never satisfies a request
  that names no school: district-wide actions need district-wide grants.
- **Read-only** refuses every mutation before grants are consulted. It is how impersonation is
  safe and how observer roles exist.
- **The rule that holds it together:** a mutating route must declare a capability. Registration
  panics otherwise, and `internal/httpapi` has a test that walks the table. That single rule
  replaces every per-role allow-list.

## Configuration: four layers

1. **The deployment file** (`/etc/tightship/config.yaml`, `internal/config`). Environment-specific
   and non-secret: organisation, listen address, database location, account domains, mail relay,
   where the master key lives, which modules are on. Decoded strictly; unknown keys are errors;
   validation reports every problem at once.
2. **The credential store.** Google service accounts, directory bind accounts, vendor API keys,
   SMTP passwords: uploaded through the running app, encrypted at rest with the master key, each
   with an owner module, an expiry, a "test connection" action and an audit trail.
3. **Tenant data.** Schools, rooms, organisational units, roles and bindings, queues, categories:
   rows, seeded by the installer, edited in the app.
4. **Modules.** Each feature area is a package satisfying `module.Module`: its name, the
   capabilities it enforces, its routes, its embedded migrations, its navigation entries. A
   deployment lists the modules it runs; a module whose integration is not configured does not
   register. Integrations sit behind interfaces with a fixture implementation, so the whole
   application runs in demo mode with no credentials.

## Layout

```
cmd/tightship/        the binary: serve, check, routes, migrate, version
internal/authz/       capabilities, grants, scope, Can, Capabilities
internal/httpapi/     the route table, the middleware, /api/v1/me
internal/config/      layer 1
internal/database/    the pool and the migration runner
internal/module/      the Module contract
internal/session/     server-side sessions: the store, the cookie, the identity resolver
internal/identity/    accounts, local passwords, auth settings, first-run setup
internal/webui/       the embedded web app (dist/ is the Vite build output)
web/                  the TypeScript app (Vite + React)
deploy/               example config and systemd unit, shipped with each release
docs/                 this
```

## The database layer

One pool, opened at start and proven before the listener comes up. A binary that starts without a
working database and discovers it on the first request has turned a deployment failure into a
user-facing one.

Four things are decided once here so no module repeats them:

- **UTC on the wire.** The session time zone is pinned to `+00:00` and the driver parses DATETIME
  into `time.Time` as UTC. Rule 9 is "store UTC, display local", and the half that breaks quietly
  is the server's own `NOW()` on a host set to local time.
- **Strict SQL mode.** `STRICT_ALL_TABLES` plus the zero-date and division rules. A silently
  truncated value is precisely the defect that ships green.
- **The application pool cannot run two statements in one call.** Migrations need that and get
  their own single connection with it enabled; the request path does not, so a query-building
  mistake cannot become a second statement.
- **The pool is sized in the deployment file.** `max_connections` is shared with every other
  client on that server, and several instances restarting at once must not exhaust it.

### Migrations

A module owns a `migrations/` directory in its embedded FS, holding `NNNN_description.sql`.
Modules apply alphabetically, versions ascend within a module. Order between modules is
alphabetical rather than wiring order because wiring order is easy to change by accident; rule 7
is what makes that sufficient, since a module needing another's tables first would already be
breaking it.

`schema_migrations` records module, version, name, checksum, `started_at` and `applied_at`. Three
properties come out of that shape:

- **MariaDB cannot roll back DDL.** A migration that fails halfway leaves a schema no retry can
  fix. The row is written *before* the SQL runs and completed after, so a row with a null
  `applied_at` means a previous run died partway — the next start refuses rather than stacking
  later migrations on a schema nobody can describe.
- **A released migration is immutable.** The checksum is compared on every start; editing one
  would mean two deployments reporting the same version with different schemas.
- **One instance migrates at a time.** `GET_LOCK` is held for the run, so a rolling restart does
  not have two binaries applying the same migration. It is session-scoped, so a crashed migrator
  releases it instead of blocking the next start forever.

### Migrations stay rollback-safe for one release

A deployment rolls back by pinning the previous tag, so the previous release's code has to keep
working against this release's schema. That is a policy nobody can hold in their head across a
year of migrations, so `tightship check` enforces it: dropping a table or column, any rename, and
adding a `NOT NULL` column with no default are all refused.

Removing a column is still possible, it just takes two releases — stop writing it in release N,
drop it in N+1, by which point no deployment can roll back to code that reads it.

## Sessions

A signed-in browser holds one cookie containing a random token. Everything else lives in a row, so
a session can be revoked, listed and expired by the server rather than by asking the browser
nicely to forget something.

**One cookie for every kind of user.** Staff arriving through Google, students through the IdP and
contractors on a magic link all get a row in the same table with a different `kind`. D3 says there
is one interface for every kind of user; this is the part that has to be true before sign-in can
be written.

**The token is never stored.** The table holds its SHA-256, so a database that leaks — or a backup
of one — is not a set of live sessions. A plain hash is right here where a password hash would not
be: the token is 256 bits of CSPRNG output, so there is no guess to slow down, and running bcrypt
on every request would be a denial-of-service surface.

**Two limits, because they answer different questions.** `idle_timeout` is how long a session may
sit unused, which is what protects an unattended browser on a shared cart. `absolute_lifetime` is
how long it may live at all, which is what bounds a stolen cookie no matter how often it is used.
Continuous use slides the first and never extends the second. Both come from the deployment file,
because a district on shared devices and one issuing staff laptops want different numbers.

Both are checked in SQL rather than in Go, so a session cannot be resurrected by a clock
difference between the application and the database — one clock decides. Expiry is exclusive: a
session is dead *at* `expires_at`, not after it.

**The sliding window is written back at a granularity**, a fortieth of the idle window, not on
every request. Otherwise every page load, poll and asset fetch carrying the cookie becomes a write,
and on a fleet this size that would be the busiest write in the system for no benefit.

**Revocation is a row.** `revoked_at` is set, never deleted, so a revoked session stays auditable
and a returning cookie is recognised as revoked rather than merely unknown. `RevokeAllFor` is
"sign out everywhere", and it is also what runs the moment an account stops being trusted.

The cookie is `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, no `Domain`, and carries the
`__Host-` prefix — which the browser enforces, so no sibling subdomain can set a session cookie
this application would then trust. `Lax` rather than `Strict` because Strict drops the cookie on
the top-level navigation back from the identity provider, landing the user on a signed-out page
immediately after signing in.

## Signing in

Local passwords exist so a deployment can be reached before SSO is configured, and after SSO
breaks. They are the bootstrap and the break-glass path, not the everyday one — which makes them
*more* worth protecting, not less: a break-glass account is the one an attacker most wants and the
one whose use nobody notices for months.

**Argon2id**, parameters stored with each hash in PHC format so they can be raised later and old
hashes keep verifying — a scheme whose parameters are compiled in is one nobody ever increases. A
sign-in with a weaker stored hash re-hashes at the current cost, which is the only moment the
plaintext exists to do it with.

**Every failure is the same failure.** No such account, wrong password, disabled account: one
message, one status. An unknown address also burns the same Argon2 computation against a dummy
hash generated at start, because a form that answers faster for addresses that do not exist is an
account-enumeration endpoint.

**The throttle counts in the database**, not in memory, so a restart does not reset an attack and
instances behind a load balancer share one count. Lockout is deliberately slow — an attacker who
can quickly lock out the break-glass account has taken away the thing that recovers a broken SSO
configuration.

**Policy is length-first.** Composition rules push people towards `Password1!` and towards writing
it down; NIST dropped them in 2017. What is left is a real minimum, a byte ceiling so a megabyte
is never fed to a memory-hard hash, and a check against the few passwords tried first. Every
problem is reported at once, for the same reason the config loader does it.

### The knob, and the guard on it

Whether local passwords are accepted at all is a setting an administrator changes in the
interface — layer 3, a row, not the deployment file, because it is a decision the district makes
and revisits. Per-account credentials are separate: one break-glass account can keep a password
after everyone else has moved to SSO, which is what you want the day Google is down.

Local sign-in **cannot be switched off until at least one person has actually signed in through
SSO.** Not "SSO is configured" — configured is an intention. A wrong redirect URI or an
unpublished consent screen is how a district locks itself out of its own deployment, and the only
account that could fix it is the one that can no longer get in. `sso_proven` is written by the
sign-in path, never by a request body, so a caller cannot assert it and unlock the guard in the
same call that uses it. Turning off every method at once is refused outright.

### First run

A fresh deployment has no accounts, so nobody can sign in to create the first one. The binary
prints a one-time link at start. Three things bound it: the token is 256 bits and stored only as a
hash, it expires, and — the control that actually matters — setup refuses once any account exists.
A token sitting in a log aggregator stops working the moment setup is done, and reissuing on each
start retires the previous one, so a restarted deployment does not leave a trail of working links.

A rejected attempt does not consume the token: one mistyped password should not burn the only way
in.

## Roles, bindings and grants

Capabilities are code; bindings are data. A district composes its own roles out of the catalogue
without a code change, and nothing in the database can invent a capability, because a capability
means something only because a route enforces it. A binding naming a capability this build does
not have is inert rather than an error — a module turned off, or a capability retired in a later
release, must not stop the rest of a role working.

**The escalation guard.** Anyone who can edit roles is otherwise one request away from every
capability in the catalogue: bind `auth.settings.write` to a role you already hold, and you are an
administrator. The rule is that **you cannot give away a capability you do not hold yourself** —
which makes role editing a way to delegate authority downwards and never a way to acquire it. It
applies to binding capabilities to a role and to granting a role to a person, because otherwise
granting is just a slower way to launder the capabilities binding refused you.

It is enforced in the service rather than a handler, so a future CLI or installer is bound by it
too, and the refusal names every offending capability at once — fixing a role one refusal at a
time is how somebody gives up and asks for the administrator role instead. The catalogue endpoint
reports `held_by_you` per capability, so the interface can grey out what this person cannot
delegate rather than letting them discover it by being refused.

**One built-in role.** `administrator` is reconciled against the catalogue at every start: it
gains capabilities a release adds and loses ones it retires, so it always means "everything this
build enforces". Without that, a capability shipped on Tuesday is held by nobody on Tuesday
morning and the screen needing it appears broken. It cannot be edited or deleted through the
interface, and the last grant of it cannot be revoked — that is the same failure as switching off
every sign-in method, reached from the other side, and it also needs a database console to undo.

**Scope is on the grant, not the role.** The same role means different things bounded to
different schools. The columns are `school_code` and `room_guid` because that is what the data
already uses — `school_code` appears in nineteen tables of the production schema and `room_guid`
in eleven. Empty means "anywhere" and is stored as the empty string rather than NULL, because in
MySQL two NULLs are not duplicates and the unique key would stop constraining.

**A disabled account holds nothing**, whatever it was granted, and disabled administrators do not
count towards the last-administrator check.

## Interface principles

- Every screen state is a URL: tabs, filters, selected records, impersonation.
- The UI never contains a role name. It asks whether a capability is present in `/me`.
- One list pattern, one form pattern, one detail pattern, shared by every module.
- Impersonation is a banner, not a mode; writes are refused server-side and disabled client-side
  from the same read-only flag.
- No request a person is waiting on runs an aggregate over an events table. Summaries are computed
  by a worker into summary rows.
- Global search is the front door: a serial, an ID, an email or a ticket number lands on the object.
- Routes are screenshot-tested with fixture data. Markup being correct and rendering being correct
  are different properties.
