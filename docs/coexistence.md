# Running beside the suite it replaces

TightShip does not arrive on a green field, and — this is the part that shapes everything else —
the suite it replaces does not go away. Both interfaces stay fully functional indefinitely. People
move across as TightShip becomes the better place to work, not because a date arrived and their
tools stopped working.

That rules out the obvious reading of a strangler migration. Nobody is going to accept raising
tickets in one system and assigning devices in another, so "this domain has moved" cannot mean
"this domain has left the other interface". It means the data moved; both interfaces still show it.

This page is how that works without either dragging the other down: without TightShip inheriting an
undocumented schema it then has to sell to somebody else, and without the deployment ending up with
two systems holding different answers to the same question.

> Throughout, **the legacy suite** means whatever system a deployment is replacing. This repository
> is public and does not name the one driving the work; the specifics — its schema, the measured
> shape of its data, the adapters written against it — live in that deployment's private
> repository.

## The two constraints, which pull in opposite directions

1. **One source of truth.** A device assigned in one system is assigned in the other, at the same
   instant, because it is the same row. Synchronisation is not an acceptable answer: a sync job is
   a second source of truth with a delay, and the interesting cases are exactly the ones where it
   is mid-flight.
2. **TightShip is a free-standing product.** A district buying it in three years gets a clean
   schema with declared relationships. It does not inherit the shape of one particular legacy
   system, and nothing in the core knows that system exists.

Reading the legacy tables directly satisfies the first and destroys the second. Copying the data
satisfies the second and destroys the first.

## The resolution: a module talks to a store, not to tables

Each module declares the operations it needs as an interface — `DeviceStore`, `TicketStore`,
`RoomStore` — in terms of TightShip's own domain model. Two implementations exist:

- **native** — TightShip's own tables, created by the module's migrations.
- **legacy** — the incumbent system's tables, adapted.

A deployment chooses per module: legacy-backed for every domain that has not cut over, native for
the ones that have. A customer with nothing to migrate runs native everywhere and never loads an
adapter at all.

This is the pattern the architecture already uses for integrations — an interface with a fixture
implementation, so the application runs in demo mode with no credentials. A legacy backend is the
same shape pointed at a different problem.

```
    module code ──► DeviceStore (TightShip's domain model)
                         ├── native  → TightShip tables (this module's migrations)
                         └── legacy  → the incumbent's tables (adapter absorbs the shape)
```

The adapter is where the legacy schema's quirks live, and they stay there. Nothing above the store
interface knows about them, so nothing above it ships to a customer who does not have them.

### What an adapter has to absorb

These are the classes of problem an adapter for a long-lived suite will meet. They are not
hypothetical — each was measured against the deployment driving this work, and the figures are
recorded in its private repository — but the point here is the shape, because a different
deployment's legacy system will have its own instances of the same four.

- **Relationships are real but implicit.** A schema of that age typically declares almost no
  foreign keys; the joins live in application code, where they are enforced inconsistently or not
  at all. Every relationship TightShip needs, the adapter establishes for itself.
- **One concept, several identities.** Staff and students are commonly separate populations in
  separate tables with unrelated keys — an enrolment identifier against a directory identifier —
  with no column in common to join on.
- **One entity, several spellings.** A vendor mirror sitting beside a locally maintained table
  will key the same device differently in each, and nothing in the schema says the two agree.
- **Records that outlive their subject.** A withdrawals or deletions log keeps rows whose subject
  has been removed from the table it references. Most of its rows point at nothing, and that is
  correct behaviour — an adapter that treats the reference as a foreign key rejects most of the
  table.

## One writer per domain — which makes the legacy suite a client

At any moment, for any domain, exactly one system writes. Devices stay legacy-written until the
Chromebooks module is ready; then they flip.

The flip is the part that is easy to get wrong. If both interfaces stay fully usable — and they
must — then both can be used to *change* a device assignment, and "one writer" collapses. The
resolution is not to take the screen away. It is that **the legacy suite stops being a peer and
becomes a client**: for a cut-over domain, its write path changes from direct SQL to a call into
TightShip's API. The screen is unchanged; the data access underneath it is not.

So a technician assigning a device in the old interface gets TightShip's validation, authorisation
and audit without knowing it, and the row they change is the row TightShip owns. There is still
exactly one writer, and there is still nothing to reconcile.

The alternative — both systems writing the same tables under an agreed convention — is the
configuration that diverges. The legacy application's invariants were written on the assumption
that it owned those rows, and they are mostly not in the schema; a convention that lives in two
codebases and no constraint is a convention that is already broken somewhere nobody has looked.

### Reads go straight to the database; writes go through the API

Not symmetric, on purpose.

**Reads** are direct SQL against TightShip's tables with a SELECT-only account. There is no
divergence risk — TightShip is the only writer — and the legacy system's list and report screens
stay one query rather than becoming a paginated HTTP call.

This assumes both schemas are reachable from one connection: the same server, or federated storage,
or a replica of TightShip's tables the legacy system can reach. Where that does not hold, the read
path has to go through the API as well, and the cost lands on exactly the screens least able to
absorb it — long lists and cross-domain reports. **Establish which case a deployment is in before
planning its first cut-over**, because it changes the work substantially.

The other cost is coupling: the tables a cut-over domain exposes become a compatibility surface, so
renaming a column there breaks a screen in the other system. Treat them as published, or publish
views in front of them.

**Writes** go through the API, because that is where the invariants, the capability checks and the
audit live. A write that reached the tables directly would bypass all three, and the module that
owns those tables would no longer be able to reason about what is in them.

### Which means the caller's identity has to be carried, not replaced

The legacy suite authenticates with a service-account credential proving which system it is, and
names the person it is acting for on each request.

The wrinkle is that this must not be modelled as impersonation. `Identity.ReadOnly` refuses every
mutation, and impersonation sets it — that is what makes impersonation safe. A technician assigning
a device through the old interface is not impersonating anybody and must be able to write. It is
also not the legacy suite acting on its own behalf: the human did it, and the audit trail has to
say so, because the actions people ask about afterwards are exactly these.

So the channel is recorded alongside the actor rather than in place of it:

- `Real` is the person — the technician signed into the other interface.
- `Via` is the channel, naming the calling system, and is new.
- `Effective` and the read-only overlay keep their existing meanings, untouched.

Two things bound what the service account can assert, because "this system says the caller is
Gary" is a claim TightShip is trusting:

- Asserting an actor at all is a capability the service account holds and nothing else does, so the
  power is visible in the catalogue rather than implicit in being a service account.
- It may only assert an account that exists and is enabled, and authorisation is then decided
  against **that person's** capabilities. A technician cannot do more through the old interface
  than they could do signed into TightShip directly.

The boundary this creates is worth stating plainly: TightShip is trusting the calling system's word
about who is asking, so that system's own authentication becomes part of TightShip's security
boundary. A shared identity provider removes that trust — the user's own token would be forwarded
and TightShip would verify it rather than the caller's assertion — and is the reason the SSO work
is worth finishing before many domains have moved.

## Cutting a domain over

A cut-over is a config change, a backfill and a flip — not an event. It takes work on both sides,
and the legacy half is not optional: a domain whose data has moved but whose old screens have not
been repointed is a domain that has silently stopped working for everyone still using that
interface.

1. **Ship the native store** alongside the legacy one, unused, and run its migrations. The tables
   now exist; the deployment still reads and writes through the legacy backend. Their existence
   does not make them authoritative — the module's configured backend decides that, and nothing
   else should be read as deciding it.
2. **Backfill** legacy rows into the native tables, and keep backfilling on a schedule. The native
   tables are a warm read-only copy, which is also how the migration is rehearsed: the backfill
   failing on real data is what tells you the adapter's assumptions are wrong.
3. **Repoint the legacy suite's READS**, and only its reads, at TightShip's tables. Its screens can
   then be exercised against real, current data while the old path still carries every write.

   Its writes stay where they are. Repointing writes before the flip would send them to a copy that
   is not authoritative and that the next backfill overwrites — a technician's change would be
   accepted, disappear, and never reach the system that is still the source of truth. That failure
   is silent, which makes it the worst available way to get this wrong.
4. **Flip the writer, and repoint the legacy suite's writes, together.** TightShip's module becomes
   native-backed; the legacy direct write path for that domain is removed and replaced by API
   calls. These are one change, not two: any gap between them either loses writes or admits two
   writers.

### What is and is not reversible

Step 4 is small — one module, one config value — and TightShip's own release stays reversible in
the usual way: the previous tag runs against the same schema, because migrations stay
backward-compatible for one release and the lint enforces it.

**The cut-over itself is not reversible by rolling TightShip back.** Rolling back the binary does
not restore the legacy system's direct write path, and by then TightShip's tables hold writes the
legacy tables never saw. Undoing a flip means repointing the legacy suite again *and* backfilling
in the opposite direction. That is a real procedure someone should write down per domain before the
flip, not an assumed property of the release mechanism.

Nothing is removed at the end. A domain's legacy store implementation goes away once that domain
has flipped — it has no reader — but the other interface's screens, and TightShip's obligation to
keep serving them, do not.

## Consequences that need building

None of this is implemented yet. Recording what it costs, so it is not discovered later:

- **Config grows a per-module backend.** Something like `modules: {chromebooks: {backend: legacy}}`,
  defaulting to native. Strictly decoded like everything else in layer 1.
- **The backend, not the schema, decides authority.** A module's native tables exist from the
  moment its migrations run, which is well before it owns anything — the backfill needs them. So
  nothing may infer "this is authoritative" from a table being present. The configured backend is
  the only answer, and the migration runner keeps applying every enabled module's migrations
  regardless of backend.
- **The legacy connection is a second pool.** Different database, possibly different credentials,
  certainly different health. A deployment whose legacy database is unreachable is down for every
  domain that has not cut over, and that has to be visible.
- **Health detail belongs behind authentication.** `/healthz` is public, and naming backends in it
  tells an unauthenticated caller what systems a district runs. The public response stays a status;
  a named, per-dependency breakdown goes on an authenticated endpoint. (The current generic
  `database` check is fine; the mistake would be adding one named after the incumbent suite.)
- **Read-only enforcement belongs in the adapter, not in review.** A legacy store for a domain
  TightShip does not own should be physically incapable of writing — a connection with SELECT-only
  grants, so the one-writer rule is enforced by the database rather than by everyone remembering.
- **The adapter does not live in this repository.** It is specific to one legacy suite, and this
  repository is public and free-standing. It lives in that deployment's private repository as a
  **downstream Go module that imports this one as a library**, with its own `cmd/` wiring public
  modules together with private adapters into its own binary.

  That is composition, not a fork, so D7 still holds: the private repository pins a tag of this
  one, fixes go upstream and arrive as the next release, and no change to the product is made by
  editing a copy of it. What this repository provides is the store interfaces; what it must not
  contain is any implementation of them that knows a particular legacy system.
- **`Identity` gains `Via`**, and the route middleware learns to populate it from an authenticated
  service account's asserted actor. Every audit row records it.
- **The API becomes a compatibility surface.** Once another system depends on it, breaking it
  breaks the district's daily work — a different obligation from the one a first-party web app
  creates, because the two are no longer released together. It needs a version in the path, a
  deprecation policy, and the same one-release rule the schema already has.
- **The tables a cut-over domain exposes are published**, wherever direct reads are in use. Either
  document them as stable or put views in front of them; what does not work is treating them as
  private and discovering otherwise when a rename takes a screen down in the other system.

## What this does not solve

**Cross-domain reporting.** A query spanning devices (cut over) and tickets (not) crosses two
schemas and two shapes, and no abstraction makes that free. Reporting stays wherever the joins are
cheapest until the domains it spans have all moved — which is a reason to cut over in an order
driven by how the data is actually queried, not by which module is easiest to write.

**Two sign-ins, until SSO.** Somebody using both interfaces signs into both. That is tolerable
while few domains have moved and increasingly not as more do, which is the practical argument for
finishing SSO early rather than treating it as a later phase.

**Feature drift between the two interfaces.** This is the one with no purely technical answer. If
people move across because TightShip is better, then TightShip gains capabilities the older
interface does not have, and a user there eventually cannot do something a colleague can. That is
the intended outcome — it is what makes the migration voluntary rather than forced — but it means
the two interfaces are not equivalent, only both *functional*.

Drift is not only cosmetic, either: once the older interface is an API client, a required field
added to a domain it writes will break it unless it is updated in step. That is the versioning
obligation above, arriving as a product decision rather than a technical one.
