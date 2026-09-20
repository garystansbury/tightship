# TightShip

An IT-operations platform for K-12 school systems: help desk and device-replacement invoicing,
Chromebook fleet management, staff and student account operations, device inventory, network
provisioning and a student single-sign-on identity provider, in one role-based console. One Go
binary, one deployment file, one database.

**Status: foundations.** The capability layer, the route table and the configuration loader are
real and tested; the feature modules are being ported one at a time from the production suite this
generalises. See [docs/architecture.md](docs/architecture.md) for the shape and
[docs/history.md](docs/history.md) for where it comes from.

## The idea in five lines

- **Everyone enters the same app.** What renders is decided by the capabilities the signed-in
  person holds, returned by `GET /api/v1/me`. A ticket submitter, a bookkeeper and a technician
  are the same application with different answers to that call.
- **Permissions are decided in one place.** A route declares the capability it needs when it is
  registered; the build fails if a write declares none. There is no second list.
- **Objects, not tools.** Devices, people, tickets, rooms and schools are the destinations; the
  tools are actions on them.
- **Districts are configuration.** Nothing district-specific is in this repository. A deployment
  is a pinned release tag plus a config file plus credentials uploaded through the running app.
- **Every screen state is a URL.** Reload, share, bookmark and re-login all land where you were.

## Build

    make web build        # needs Go 1.27+ and Node 22+; produces bin/tightship
    make test
    ./bin/tightship check --config deploy/config.example.yaml

`go build` alone works too — the binary then serves the API and a page saying the web app was not
built.

## Run locally

    cp deploy/config.example.yaml deploy/config.dev.yaml   # set dev.allow_debug_identity: true
    make run
    curl -H 'X-Tightship-User: you@example.org' -H 'X-Tightship-Roles: tech@CMS' localhost:8090/api/v1/me

The debug identity header only works when the deployment file allows it, and `serve` logs a
warning every start while it does.

## Deploy

A deployment never runs a commit that is not a tag. Tags build a static `linux/amd64` binary on
GitHub Actions and attach it to a release with its checksum, the example config and a systemd
unit. See [docs/deploying.md](docs/deploying.md).

## Licence

Apache-2.0. See [LICENSE](LICENSE).
