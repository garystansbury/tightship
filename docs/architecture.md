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
cmd/tightship/        the binary: serve, check, routes, version
internal/authz/       capabilities, grants, scope, Can, Capabilities
internal/httpapi/     the route table, the middleware, /api/v1/me
internal/config/      layer 1
internal/module/      the Module contract
internal/webui/       the embedded web app (dist/ is the Vite build output)
web/                  the TypeScript app (Vite + React)
deploy/               example config and systemd unit, shipped with each release
docs/                 this
```

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
