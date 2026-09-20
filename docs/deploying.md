# Deploying TightShip

TightShip is a product. A school system running it is a **deployment**: a pinned release, a
configuration file, credentials, and a place to run it. The two never mix.

## The rule

**A deployment never runs a commit that is not a tag.** Every tag `v*` builds a static binary on
GitHub Actions, tests it, and attaches it to a release with a checksum, the example config and a
systemd unit. If a deployment needs a change in the product, the change is made upstream and
arrives as the next tag — a patch release takes minutes to cut. Because a fix goes upstream first,
every deployment gets it, which is the point of a product.

Do not fork and rebase. A fork is a second copy of the code, and a second copy that must be kept
in step is the failure this suite was rebuilt to escape.

## What a deployment repository holds

Keep it small and keep it in the deploying organisation's own account, so the district can
reproduce its installation without the product author:

- the pinned tag (`TIGHTSHIP_VERSION=v1.4.2`)
- `config.yaml` (layer 1 — no secrets)
- the systemd unit and the reverse-proxy vhost
- the script that fetches the release asset for the pinned tag, verifies its checksum, installs it
  and restarts the unit
- the location of the master key (never the key)

## Install

1. Create a system user and directories:
   `useradd -r -s /usr/sbin/nologin tightship; install -d -o root -g tightship -m 750 /etc/tightship`
2. Fetch the release binary and its checksum from the release page, verify, install to
   `/usr/local/bin/tightship`.
3. Copy `config.example.yaml` to `/etc/tightship/config.yaml` and edit. `tightship check --config
   /etc/tightship/config.yaml` reports every problem at once.
4. Generate the master key: `head -c 32 /dev/urandom > /etc/tightship/master.key; chmod 640 …`.
5. Put the bootstrap database password in `/etc/tightship/env` as `TIGHTSHIP_DB_PASSWORD=…`.
6. Install `tightship.service`, `systemctl enable --now tightship`, front it with your reverse
   proxy terminating TLS, and list the proxy addresses in `server.trusted_proxies`.
7. Sign in and upload credentials in Admin → Credentials.

## Upgrade

Change the pinned tag, run the fetch script, restart. Migrations run at start; a release note names
the migrations it carries. Roll back by pinning the previous tag — migrations are written to be
backward-compatible for one release for that reason.
