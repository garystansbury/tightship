# Working in this repository

TightShip is an IT-operations platform for K-12 school systems, rebuilt as one Go binary and one
web app from a production PHP suite. Read `docs/architecture.md` first, then `docs/roadmap.md`
for what is next. This file is the short version of the rules; the docs carry the reasoning.

## This repository is public

Nothing district-specific goes in it: no hostnames, IP addresses, account names, ticket numbers,
vendor contract details or people's names. Every such fact is configuration (layer 1), a credential
(layer 2), or a row (layer 3). If a change needs a real-world value to make sense, use the
`example.org` family. The production deployment that drives this work has its own private
repository; this one never references it by name.

## Rules that hold the design together

1. **A mutating route declares a capability, or registration panics.** `internal/httpapi` enforces
   it and `TestEveryMutatingRouteDeclaresACapability` walks the table. Never add a second
   allow-list anywhere.
2. **Capabilities are code; bindings are data.** `authz.Register("module.action", "…")` in a
   module's `init()`. No runtime "create a permission". Roles hold capabilities via
   `role_bindings`.
3. **The UI renders from `GET /api/v1/me` and nothing else.** No role name ever appears in
   `web/`. A control appears when the capability is held; the route refuses the same capability.
4. **Every screen state is a URL.** Tabs, filters, selected records, impersonation. A reload or a
   re-login lands where you were.
5. **One data layer** (`web/src/api.ts`). A 401 is handled in exactly one place.
6. **Scope is context, read-only is an overlay.** A school-bounded grant never satisfies a
   district-wide request. Impersonation is read-only and every audit line names the real user.
7. **A module owns its tables.** Migrations live in the module; no other module writes them.
8. **No request a person waits on runs an aggregate over an events table.** Summaries are computed
   by a worker into summary rows.
9. **Store UTC, display local.** The deployment file carries the display timezone.
10. **Configuration is strict.** Unknown keys are errors; validation reports every problem at once.
11. **A tag is a release; a deployment pins a tag.** Fixes go upstream, never into a fork.

## Build, test, run

    make web build           # Go 1.27+, Node 22+
    make test                # go test ./...
    ./bin/tightship check  --config deploy/config.example.yaml
    ./bin/tightship routes --config deploy/config.example.yaml

`deploy/config.dev.yaml` is git-ignored; copy the example there with `dev.allow_debug_identity:
true` and `make run`, then send `X-Tightship-User` / `X-Tightship-Roles: tech@CMS` headers.

## Conventions

- Go: stdlib first. Current dependencies: `gopkg.in/yaml.v3`. Add a dependency when it carries
  real weight (the MariaDB driver, a migration runner, the Google API clients), not for
  convenience.
- Comments explain why, and name the failure the code prevents. The production suite's history
  is full of defects that shipped green; a test here should assert behaviour at a boundary, not
  arithmetic about it.
- Tests that assert on source text strip comments first. A docblock quoting the spelling under
  test is how a source assertion passes against broken code.
- Every module is added the same way: `internal/<module>/` satisfying `module.Module`; its
  capabilities registered in `init()`; its routes registered through `httpapi.Router.Handle`;
  its migrations embedded; its navigation entries declared. Wire it in `cmd/tightship` behind
  `config.ModuleEnabled`.
- Frontend: TypeScript, React, Vite. One list pattern, one form pattern, one detail pattern,
  shared. Keep a route screenshot-tested with fixture data once the harness exists.
- Commit messages say what changed and why in the first line, then the reasoning. Pull requests
  to `main`; CI must be green.
