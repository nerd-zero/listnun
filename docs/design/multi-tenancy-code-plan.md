# Multi-tenancy: code-level implementation plan

Companion to [`multi-tenancy.md`](./multi-tenancy.md) (architecture/research) and
[`multi-tenancy-erd.drawio`](./multi-tenancy-erd.drawio) (schema diagrams). That
doc has the *why*; this one has the *how* — concrete SQL, Go signatures, and
file-by-file diffs to implement the 9 phases tracked as GitHub issues
[#28](https://github.com/nerd-zero/listnun/issues/28)–[#35](https://github.com/nerd-zero/listnun/issues/35)
and [#38](https://github.com/nerd-zero/listnun/issues/38)
under [#27](https://github.com/nerd-zero/listnun/issues/27).

Status: **not started**. Nothing below has been applied to the codebase yet.

---

## Phase 1 — schema foundation

**Status: implemented** (`internal/migrations/v6.4.0.go`, `schema.sql`,
issue #28). The original draft below this line in earlier versions of this
doc had two bugs, caught while grounding it against the actual `schema.sql`
and `queries/*.sql` before running anything — recorded here so a future
session doesn't reintroduce them:

1. **It dropped constraints `ON CONFLICT` clauses depend on.** The draft
   re-scoped `subscribers_email_key`, `links_url_key`, `users_username_key`/
   `users_email_key`, `templates_is_default_idx`, `campaigns_archive_slug_key`,
   and `roles`' `idx_roles`/`idx_roles_name` to composite `(tenant_id, ...)`
   forms in the *same* migration as the `tenant_id` column addition. But
   `queries/subscribers.sql` has `ON CONFLICT (email)`, `queries/links.sql`
   has `ON CONFLICT (url)`, and `queries/roles.sql` has
   `ON CONFLICT (parent_id, list_id)` — dropping the constraint these
   target breaks those upserts immediately, before the query-layer changes
   land (those are phase 4's job, per the "migrate users/roles uniqueness
   constraints" step in `multi-tenancy.md`'s phased plan — the high-level
   plan already had this right; only this file's detailed draft jumped
   ahead).
2. **The nullable → backfill → `SET NOT NULL` dance was unnecessary.**
   Adding the column as `NOT NULL DEFAULT 1 REFERENCES tenants(id)` in one
   statement covers existing rows *and* any row the (not-yet-tenant-aware)
   app inserts, since no `INSERT` in `queries/*.sql` sets `tenant_id` until
   a later phase threads it through. Postgres 11+ treats a constant
   `DEFAULT` on `ADD COLUMN` as metadata-only (no table rewrite), so this
   is simpler *and* removes a whole class of migration-ordering bugs.

**What actually shipped** is purely additive: a new `tenants` table, plus
one `tenant_id INTEGER NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE
CASCADE ON UPDATE CASCADE` column + one `idx_<table>_tenant` index on every
scoped table (the 9 originally listed) and every join/log table
(`subscriber_lists`, `campaign_lists`, `campaign_views`, `campaign_media`,
`link_clicks`) and `settings` — 15 tables total, all via `DEFAULT 1`, no
parent-join backfill logic needed since every row today belongs to tenant
1 regardless of table. **No existing constraint, index, or query was
touched.** See `internal/migrations/v6.4.0.go` for the exact SQL and
`schema.sql` for the fresh-install equivalent (the `tenants` table + seed
row is added first, since every other table's FK needs it to exist).

Explicitly deferred to the phase that also updates the corresponding
`queries/*.sql` file: re-scoping `subscribers_email_key`, `links_url_key`,
`users_username_key`/`users_email_key`, `templates_is_default_idx`,
`campaigns_archive_slug_key`, `idx_roles`, `idx_roles_name`, and
`settings_key_key`/the settings PK to composite `(tenant_id, ...)` forms —
that's phase 4 (auth/query threading) and phase 5 (settings), not phase 1.

Verified against the live dev DB: migration runs clean, is a no-op on
re-run (version-tracked), all 15 tables backfilled to tenant 1, and the
running app (Campaigns/Subscribers pages) was unaffected before and after.

---

## Phase 2 — RLS policies and indexing

**Status: implemented** (`internal/migrations/v6.5.0.go`, issue #29).

The original draft here (`USING (tenant_id = current_setting('app.current_tenant', true)::INTEGER)`,
"unset context sees nothing") had the same class of ordering bug as the
original Phase 1 draft: since the app doesn't set `app.current_tenant`
until the auth/request-flow phase, a strict policy would make **every**
query return zero rows the moment this migration runs against any
correctly-permissioned deployment (non-superuser, non-owner app role) —
a total outage, well before this dev sandbox (where the connecting role
happens to be a superuser) would ever show a symptom.

**What shipped instead** is deliberately permissive while tenant context is
unset:

```sql
ALTER TABLE subscribers ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscribers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON subscribers
    USING (
        tenant_id = current_setting('app.current_tenant', true)::INTEGER
        OR current_setting('app.current_tenant', true) IS NULL
    );
-- repeat per table (all 15 from phase 1: the 9 scoped tables, 5 join/log
-- tables, and settings)
```

Two deliberate additions beyond the original draft:

- **The `OR ... IS NULL` fallback** keeps the app fully functional today
  (single tenant, no session plumbing yet) while still exercising the real
  RLS machinery — the policy correctly restricts once a session sets
  `app.current_tenant` (verified below), it just doesn't punish the app for
  not setting it yet. **Tighten this** (drop the `OR` branch) once Phase 4
  lands and the app reliably sets tenant context on every request — tracked
  as a follow-up, not a new phase.
- **`FORCE ROW LEVEL SECURITY`** in addition to `ENABLE`: table owners are
  exempt from RLS by default, and most self-hosted listnun installs
  (including this dev database) use a single Postgres role for both schema
  ownership and the app connection — so plain `ENABLE` alone would have
  silently been a no-op for that common deployment shape. `FORCE` closes
  that gap. Superusers remain exempt regardless of `FORCE` (no way around
  that in Postgres), which is why this is still inert in the current dev
  sandbox specifically — but it matters for any non-superuser owner role.

**Verify before merging this phase:**
```sql
-- confirm the app's role has no bypass
SELECT rolname, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user;
-- must show rolsuper=f, rolbypassrls=f
```
This dev database's role fails that check (superuser), which is why direct
verification required a throwaway non-owner, non-superuser role
(`rls_test_role`, `NOSUPERUSER NOBYPASSRLS`, granted table access, no RLS
bypass) — created, used to confirm all three cases (no context set → sees
all rows; `app.current_tenant='1'` → sees only tenant 1; `='2'` → sees only
tenant 2), then fully dropped afterward (role, grants, and the temporary
tenant-2 test data). Nothing from this test role persists in the repo or
the dev DB.

**Indexing:** the "tenant_id must be the leading column in composite
indexes" requirement from the RLS research notes above is satisfied by
Phase 1's per-table `idx_<table>_tenant` index (tenant_id alone, trivially
leading) — sufficient for the RLS predicate itself to use an index.
Rewriting *other* existing indexes (e.g. `idx_subs_status`,
`idx_camps_created_at`) to prepend `tenant_id` is a separate performance
optimization, not required for correctness, and is deferred until query-plan
profiling in a later phase shows an actual need — there's only one tenant
with data today, so there's nothing to profile against yet.

---

## Phase 3 — connection/session plumbing

**Status: implemented** (`internal/core/tenant.go`, `internal/core/tenant_test.go`,
issue #30). Two changes from the original draft:

1. **`WithTenant` is a `*Core` method, not a package-level function taking
   `*sqlx.DB`.** Phase 4 will call it from inside existing `Core` methods
   where `c` is already the receiver, so `c.WithTenant(ctx, tenantID, fn)`
   is more idiomatic than threading `c.db` through a free function — matches
   the existing style of other `Core` methods (`c.RefreshMatView`, etc.) in
   `internal/core/core.go`.
2. **The concurrency spike is a real, permanent test**
   (`TestWithTenant_ConcurrentIsolation`), not just a described plan — this
   was the repo's *first* Go test (`go test ./...` previously had zero test
   files anywhere outside `frontend/`), so it also had to establish how a
   Postgres-backed Go test connects to a database at all. It reads the same
   `LISTNUN_db__*` env vars `.github/workflows/tests.yml` already sets for
   CI, falling back to this repo's local dev `config.toml` values so it also
   runs against `make run`'s dev database with no extra setup, and skips
   (rather than fails) if no database is reachable.

Critically, the test does **not** run as the app's configured DB role.
Both this dev database's role and CI's default `postgres:16-alpine` image
role are superusers, and superusers (like table owners, see phase 2) bypass
RLS entirely regardless of policy — testing against either would make the
test pass for the wrong reason (or, if isolation were actually broken,
mask it). The test creates a throwaway, uniquely-named, least-privileged
role (`NOSUPERUSER NOBYPASSRLS`, granted only `SELECT`) for its own
duration, runs concurrent goroutines through `WithTenant` against **that**
role's connection pool, and tears the role + its temporary tenant/subscriber
test rows down in `t.Cleanup` afterward — mirroring the manual verification
process from phase 2.

```go
func (c *Core) WithTenant(ctx context.Context, tenantID int, fn func(tx *sqlx.Tx) error) error {
    tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{})
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1::TEXT, true)`, tenantID); err != nil {
        return err
    }
    if err := fn(tx); err != nil {
        return err
    }
    return tx.Commit()
}
```

Verified: `go test ./internal/core/... -run TestWithTenant_ConcurrentIsolation -race -count=1` passes reliably (run repeatedly to confirm — an early version had a `t.Cleanup`-ordering bug where an in-function `defer db.Close()` ran *before* the cleanup callback that still needed that connection, silently leaving orphaned test tenants/roles behind despite the test itself reporting `PASS`; fixed by closing the admin connection at the end of the same cleanup callback instead of via a separate `defer`). `go vet ./...` and the full `go test ./... -race` suite pass with the dev database up.

**Not in scope for this phase** (that's phase 4): no existing `Core` method
calls `WithTenant` yet. This is additive — a new file plus a new test, zero
behavior change to the running app, exactly like phases 1 and 2.

---

## Phase 4 — auth and request-flow tenant resolution

**Status: subdomain resolution implemented (issue #31); `internal/core`
tenantID-threading split out to a new follow-up issue.** Investigation
while grounding this phase found the `internal/core` part alone touches
**107** exported `Core` methods and **150** call sites in `cmd/*.go`, none
of which take a `context.Context` today and none run through a
transaction — they call directly on pool-level prepared statements
(`c.q.X.Select(...)`). Making phase 3's `WithTenant` actually apply to
real queries means rebinding each one via `tx.Stmtx(c.q.X)`, a mechanical
but invasive change across all 107 methods (plus adding `ctx` to all of
them as a prerequisite, since none have it). Too large to do safely in
one pass alongside the resolution work — deliberately deferred, tracked as
a separate follow-up rather than attempted piecemeal.

**What shipped** is the resolution/auth half only, gated behind a new
`app.multi_tenancy_enabled` config flag (default `false`) — purely
additive, like phases 1-3.

### `models/tenant.go` (new)

Holds `type Tenant struct { Base; Slug, Name, Status string }` *and* the
shared context-key constant `TenantCtxKey`. Both live in `models`, not in
`internal/tenant`, because of an import-cycle constraint discovered here:
`internal/core` already imports `internal/auth` (for permission types used
in `subscribers.go`/`users.go`/`roles.go`). A tenant-resolution middleware
needs `*core.Core`, so it can't live in — or be imported by —
`internal/auth`, or you get `auth → tenant → core → auth`. Putting the
struct and context key in `models` (which has zero internal dependencies)
lets both `internal/tenant` (imports `core` + `models`) and
`internal/auth` (imports only `models`) use them without cycling.

### `internal/tenant/resolve.go` (new)

```go
func Middleware(core *core.Core, rootDomain string, enabled bool) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if !enabled {
                c.Set(models.TenantCtxKey, defaultTenant) // stub, ID: 1, no DB hit
                return next(c)
            }

            host := c.Request().Host
            if h, _, err := net.SplitHostPort(host); err == nil {
                host = h
            }
            slug := strings.TrimSuffix(host, "."+rootDomain)
            if slug == host || slug == "" {
                return echo.NewHTTPError(http.StatusNotFound)
            }

            t, err := core.GetTenantBySlug(slug)
            if err != nil {
                return echo.NewHTTPError(http.StatusNotFound)
            }
            if t.Status != "active" {
                return echo.NewHTTPError(http.StatusServiceUnavailable, "workspace unavailable")
            }

            c.Set(models.TenantCtxKey, &t)
            return next(c)
        }
    }
}
```

No slug→tenant cache yet (the original draft's TTL cache) — deferred the
same way phase 2 deferred composite-index rework: there's only ever one
tenant to look up right now, revisit if profiling shows a need once
there's more than one.

Registered in `cmd/init.go:initHTTPServer`'s global `srv.Use`, immediately
after the existing `c.Set("app", app)` middleware and before
`initHTTPHandlers(srv, app)` sets up the app's 3 separate route groups
(authenticated pages, authenticated `/api`, public) — confirmed this is
the *only* middleware that spans all 3 today, which is exactly why it's
the right insertion point for something that must also cover public
routes (unsubscribe, archive, tracking pixel).

### `internal/auth/models.go`

```go
type User struct {
    Base
    TenantID int `db:"tenant_id" json:"tenant_id,omitempty"`
    // ... existing fields unchanged ...
}
```

No query changes needed: `queries/users.sql`'s `get-user`/`get-users` both
already `SELECT *`/`users.*`, so `tenant_id` (a real column since phase 1)
was already coming back in every result row.

### `internal/auth/auth.go`

`Auth.Middleware` never aborts on failure itself — on any failure it does
`c.Set(UserHTTPCtxKey, echo.NewHTTPError(...))` and calls `next(c)` anyway;
rejection is deferred to per-route-group wrapper middleware
(`redirectIfUnauth`/`jsonErrorIfUnauth` in `cmd/handlers.go`) that inspects
`c.Get(UserHTTPCtxKey)`. The tenant cross-check follows this exact
convention rather than returning an error directly:

```go
func tenantMismatch(c echo.Context, user User) bool {
    t, ok := c.Get(models.TenantCtxKey).(*models.Tenant)
    return ok && user.TenantID != t.ID
}
```
called after both places a user is successfully resolved (API-token path
and session path), setting the *same* `"invalid session"` error the
existing invalid-session case uses — so a cross-tenant replay is
indistinguishable from an expired session to the client, no extra
information disclosed.

Verified end-to-end against a temporary second backend instance
(`LISTNUN_app__multi_tenancy_enabled=true LISTNUN_app__root_domain=localhost`,
different port, same dev DB): logged in on tenant 1's own subdomain
(`default.localhost`), confirmed the resulting session cookie works there,
then confirmed reusing that exact cookie against a second throwaway
tenant's subdomain gets rejected with `403 invalid session` — the concrete
"session issued for tenant A replayed against tenant B" scenario this
check exists for.

**Session cookie:** already host-only (no `Domain` attribute set anywhere
in `internal/auth`'s `simplesessions.Options` or `cmd/init.go`'s
`initAuth()`) — this requirement was already satisfied before this phase,
nothing to change.

### OIDC callback, `internal/core/*` tenantID-threading, super-admin scope

Deferred — see status note above. OIDC callback tenant-awareness depends
on phase 5 (per-tenant settings) landing first; `internal/core` threading
is its own follow-up issue given the 107-method scope; `SuperAdminRoleID`
becoming tenant-scoped depends on that threading landing.

---

## Phase 5 — settings, fully per-tenant

**Decided:** no global/per-tenant split. Every key — including `smtp`,
`security.oidc`, and `upload.s3.*` — is per-tenant. `models.Settings` stays a
single flat struct; only the load/save path gains a `tenantID` parameter.

**Status: DB layer implemented (issue #32); subsystem redesign split into
its own follow-up.** Investigation before starting found this phase bundles
two very different sizes of work, the same pattern as phase 4/issue #40:
settings aren't read per-request — they're loaded **once at process boot**
into a global `koanf` config, and the SMTP messenger pools, the media/S3
store, the OIDC config, and the campaign manager are all built **once as
process-lifetime singletons** from that global state (`cmd/main.go`'s
`initSMTPMessengers`/`initMediaStore`/`initAuth`/`initCampaignManager`).
There's no live-reload mechanism today at all — even the *existing*
single-tenant "update settings" flow requires a full process re-exec
(`cmd/settings.go`'s `handleSettingsRestart` → `syscall.Exec`) to take
effect. Making those four subsystems genuinely per-tenant is a redesign,
not a parameter addition — split into its own follow-up issue, same as
`internal/core` threading was split into #40. **What shipped this session
is the DB/`Core` layer only**, with those subsystems deliberately left as
global singletons pinned to tenant 1 (documented in code, zero behavior
change from today).

### Migration `v6.6.0`: composite `(tenant_id, key)` primary key

Re-scopes `settings` from the phase-1-deferred `UNIQUE(key)` to a real
composite key — this was explicitly left for "the phase that also updates
the corresponding `queries/*.sql` file," and this is that phase:

```sql
ALTER TABLE settings DROP CONSTRAINT IF EXISTS settings_key_key;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'settings'::regclass AND contype = 'p') THEN
        ALTER TABLE settings ADD PRIMARY KEY (tenant_id, key);
    END IF;
END $$;
```

**Found and fixed while running this migration, not before:** `cmd/install.go`'s
`recordMigrationVersion` — the upgrade runner's *own* bookkeeping, used by
every migration including this one — does `INSERT INTO settings (key,
value) VALUES ('migrations', ...) ON CONFLICT (key) DO UPDATE ...`. Dropping
`settings_key_key` broke it immediately (confirmed by actually running the
migration, not just reading the code): `pq: there is no unique or exclusion
constraint matching the ON CONFLICT specification`. Fixed by adding
`tenant_id` to the insert and changing to `ON CONFLICT (tenant_id, key)`.
Also fixed `cmd/upgrade.go`'s `getLastMigrationVersion` (`SELECT value->>-1
FROM settings WHERE key='migrations'`) to add `AND tenant_id=1` for the same
reason — both pinned to tenant 1, matching `initSettings` below. This is
the exact class of bug phase 1 was originally trying to avoid by deferring
constraint changes; it still slipped through because this specific query
lives in the *migration framework itself*, not `queries/*.sql`, so it
wasn't caught by grepping for `ON CONFLICT` across the query files during
planning.

### `internal/core/settings.go`

```go
func (c *Core) GetSettings(ctx context.Context, tenantID int) (models.Settings, error)
func (c *Core) UpdateSettings(ctx context.Context, tenantID int, s models.Settings) error
func (c *Core) UpdateSettingsByKey(ctx context.Context, tenantID int, key string, value json.RawMessage) error
```
All three now run through `WithTenant`, using the `stmtx()` helper from
issue #40's slice 1 (same `.Unsafe()`-preservation requirement applies
here). `queries/misc.sql`'s `get-settings`/`update-settings`/
`update-settings-by-key` all gained a `tenant_id` parameter. For
`update-settings` specifically, the `tenant_id` filter is **required for
correctness, not just defense-in-depth**: it does `WHERE s.key = c.key` for
each key in an incoming JSON map, and since key names now repeat across
tenants by design, an unfiltered version would silently update every
tenant's row sharing a key name in that map.

### `cmd/init.go`'s `initSettings` — pinned to tenant 1

The boot-time global-config load now explicitly passes tenant 1 to
`get-settings`'s new `$1` parameter, with a code comment explaining this is
intentional and matches the "subsystems stay global singletons" scoping
decision above — not an oversight to fix later independently of the
subsystem-redesign follow-up.

### Knock-on effects — now explicitly deferred to the subsystem-redesign follow-up

- **`internal/media`'s S3 client**: per-tenant client cache (map keyed by
  `tenant_id`, lazily constructed).
- **OIDC callback**: needs tenant resolved before loading IdP config (phase 4
  already flagged this).
- **SMTP messenger pools**: per-tenant construction instead of one global
  pool built at boot.
- **Live settings reload**: replacing `syscall.Exec` full-process-restart
  with something that can refresh one tenant's config without affecting
  every other tenant's running state.

---

## Phase 6 — tenant-aware manager/dispatcher

**Status: implemented.** Was blocked on real per-tenant dispatch existing
to receive tenant-scoped scan results — unblocked once #41 slice 1 (SMTP
messenger resolution, below) shipped.

### What shipped

`Store` interface gained `NextCampaigns(tenantID int, currentIDs,
sentCounts []int64) ([]*models.Campaign, error)` (tenantID added as the
leading param) and a new `GetActiveTenantIDs() ([]int, error)` method,
backed by a plain `SELECT id FROM tenants WHERE status = 'active'` query
(no caching added — this only ever returns one row today with no way to
create more, so premature to optimize; revisit if profiling shows a need,
same deferral pattern used elsewhere in this plan).

`scanCampaigns` now: fetches active tenant IDs each tick, groups the
in-flight campaigns already being tracked (`m.pipes`) by tenant (via the
`TenantID` field `models.Campaign` gained in #41 slice 1), then calls
`NextCampaigns` once per active tenant with only that tenant's in-flight
IDs/counts. `queries/campaigns.sql`'s `next-campaigns` gained a `tenant_id
= $3` filter on its `camps` CTE.

**Why "group in-flight campaigns by tenant" rather than just adding the
tenant filter and calling it once per tenant with the same global
in-flight list each time:** `next-campaigns` reuses its `$1` (current IDs)
parameter for two purposes — excluding those campaigns from re-selection,
*and* (via `unnest($1::INT[], $2::INT[])` in an `updateCounts` CTE)
incrementing their `sent` counts in the DB. Passing the same global list to
every per-tenant call would apply that increment once per tenant scanned
each tick, double/triple/etc.-counting `sent`. Traced through this before
writing any code, not discovered by testing — the fix is in
`internal/manager`'s `getCurrentCampaigns` (now groups into
`map[int]currentCampaignIDs`), not in the SQL.

### A real bug caught by live-testing, not by reading the code

The zero-value of `currentCampaignIDs` (returned by a map lookup for a
tenant with zero in-flight campaigns — the common case) has `nil` `ids`/
`counts` slices. `pq.Int64Array(nil).Value()` serializes to SQL `NULL`,
not an empty array (confirmed via a throwaway Go program, not assumed) —
and `NOT(campaigns.id = ANY(NULL::INT[]))` evaluates to SQL `NULL` under
ordinary three-valued comparison semantics, which a `WHERE` clause treats
as "false", filtering out **every** row. A campaign set to `running` was
never picked up; no error anywhere, just silently zero results every
tick. Root-caused by comparing against the original single-tenant
`getCurrentCampaigns`, which explicitly built its slices with
`make([]int64, 0, len(m.pipes))` rather than a zero-value default — a
detail its own comment ("needs to return an empty slice in case there are
no campaigns") called out, that got lost when restructuring the return
type to a per-tenant map. Fixed by explicitly defaulting `e.ids`/
`e.counts` to `[]int64{}` before each `NextCampaigns` call, with a comment
explaining why so the next refactor of this code doesn't reintroduce it.

**Verified live**, not just built: started the existing draft campaign via
the API, confirmed via the backend log that `scanCampaigns` picked it up
on the very next tick, `getMessenger` resolved tenant 1's SMTP messenger
(from #41 slice 1), and the send failed at the expected point (a real
timeout against this dev environment's fake mail host) — then confirmed
the fix by reproducing the *bug* first (campaign stuck in `running`,
zero ticks logged as picking it up, across several tick cycles) before
applying and re-verifying the fix. Campaign state reset to `draft`
afterward.

Rate limiting (`app.message_rate`/`app.concurrency`) still shares one
global worker-pool size across all tenants' campaigns — deferred, matching
the original draft's recommendation to start simple and revisit only if a
tenant experiences starvation in practice.

---

## Issue #41 — per-tenant SMTP/media/OIDC/manager (4 subsystems, same
shape as #40)

### Slice 1 — SMTP messenger resolution: implemented

Investigation found `internal/manager` never imports `internal/core` (its
`Store` interface, implemented by `cmd/manager_store.go`, already bridges
DB access without one) and neither it nor `internal/messenger/email`
imports the other — confirmed via grep before designing, not assumed, so
wiring a new per-tenant resolver from `cmd/` carries zero cycle risk.

**What shipped:**
- `models.Campaign` and `models.Message` gained a `TenantID` field (the
  former free via existing `campaigns.*` queries, same pattern as
  `auth.User.TenantID`; the latter explicit, since the transactional send
  path — `cmd/tx.go`'s `SendTxMessage` — has no `Campaign` to derive it
  from).
- New `internal/manager.MessengerResolver` interface (`GetMessenger(ctx,
  tenantID, name) (Messenger, error)`, plus an `ErrMessengerNotFound`
  sentinel) and a `Manager.getMessenger` helper that tries the resolver
  first, falling back to the existing process-global `m.messengers` map
  (unchanged) on `ErrMessengerNotFound` — this is what keeps postback
  messengers (a different messenger type, out of scope for this slice)
  working without needing to explicitly classify messenger types.
- Concrete resolver: `cmd/tenant_messenger.go`'s `tenantMessengers`, holding
  a `*core.Core` reference and a `map[int]map[string]manager.Messenger`
  cache (mutex-protected, populated lazily, **never invalidated within a
  process's lifetime** — deliberately matching the existing
  process-global messenger set's staleness contract, since settings
  updates already require a full `syscall.Exec` restart to take effect
  today, confirmed live during phase 5. Live invalidation is out of scope,
  tracked as part of this issue's remaining work).
- `cmd/init.go`'s `initSMTPMessengers()` refactored to accept a
  `*koanf.Koanf` parameter (instead of closing over the global `ko`) and
  to **return an error instead of calling `lo.Fatalf`** — this was a
  necessary change beyond the original plan, found while implementing:
  the function is now called from two contexts (boot, which should still
  fail fast on bad config, vs. the lazy per-tenant path, where crashing
  the whole process over one tenant's malformed SMTP config would take
  down every tenant). The boot-time caller (`cmd/main.go`) still calls
  `lo.Fatalf` itself on the returned error, preserving today's behavior
  exactly.
- The per-tenant resolver reuses `initSMTPMessengers` rather than
  reimplementing SMTP-config parsing: it round-trips a tenant's
  `models.Settings` through `json.Marshal`/`json.Unmarshal` into a fresh
  per-tenant `*koanf.Koanf` (the JSON tags on `Settings` already match the
  dotted keys `initSettings` loads from the DB at boot), then calls the
  same function. This avoids a naive JSON-only conversion into
  `email.Server`, which embeds `smtppool.Opt` via a mapstructure-only
  `,squash` tag that plain `encoding/json` doesn't understand.

**Verified live, not just built/tested:** started the existing draft "Test
campaign" via the API and watched the backend log show
`initSMTPMessengers`'s init line fire a *second* time (the first is the
boot-time global construction; the second, at campaign-start, is the lazy
per-tenant resolver building tenant 1's messenger set for the first time),
followed by `newPipe`'s validation passing (no "unknown messenger" error)
and a real SMTP connection attempt that failed with a legitimate network
timeout against this dev environment's fake mail host — the expected,
correct failure mode, proving resolution succeeded and only delivery
(irrelevant to this slice) failed. Campaign state reset to `draft`
afterward to restore the dev DB baseline.

**Remaining in #41, not this slice:** media/S3 per-tenant client cache,
OIDC per-tenant config resolution, and Phase 6's scan-side tenant
awareness (now unblocked, see above).

### Slice 2 — media/S3 per-tenant store resolution: implemented

Same shape as slice 1, one subsystem later. `upload.*` settings
(provider, filesystem paths, S3 credentials/bucket/etc.) are already
per-tenant (phase 5), so this is another "settings are per-tenant, the
*consumer* isn't yet" gap.

**What shipped:**
- `cmd/init.go`'s `initMediaStore(ko *koanf.Koanf) media.Store` refactored
  to **return `(media.Store, error)` instead of calling `lo.Fatalf`** —
  identical reasoning to `initSMTPMessengers`'s slice-1 change: boot still
  fails fast (`cmd/main.go` calls `lo.Fatalf` on the returned error
  itself), but the lazy per-tenant path must not crash the whole process
  over one tenant's malformed upload config.
- New `cmd/tenant_media.go`: `tenantMedia`, a `*core.Core`-backed,
  mutex-protected, lazily-populated `map[int]tenantMediaStore` cache
  (never invalidated within a process lifetime — same staleness contract
  as `tenantMessengers` and the same underlying reason: settings updates
  require a full restart today). `Get(ctx, tenantID) (media.Store,
  models.Settings, error)` returns **both** the resolved store and the
  settings it was built from (not just the store) — `cmd/media.go`'s
  handlers need `UploadProvider` (the `media.provider` DB column) and
  `UploadExtensions` (upload validation) alongside the store itself, and
  returning the already-fetched `models.Settings` avoids a second
  `Core.GetSettings` round-trip for those two fields. Reuses the same
  `json.Marshal`/`Unmarshal`-into-fresh-`koanf` technique as
  `tenantMessengers`, then calls `initMediaStore(tenantKo)` — no
  S3/filesystem-specific parsing logic duplicated.
- Unlike SMTP's messenger resolver, **no fallback map is needed**: every
  media consumer (`cmd/media.go`'s HTTP handlers, `cmd/manager_store.go`'s
  campaign-attachment methods) already has a `tenantID` in scope, so
  there's no "postback"-shaped exemption to preserve. `App.media`'s type
  changed from `media.Store` to `*tenantMedia` outright; every call site
  in `cmd/media.go` now resolves via `a.media.Get(ctx, tenantID(c))`
  first, then uses the returned store/settings.
- `cmd/manager_store.go`'s `store.media` field changed from `media.Store`
  to `*tenantMedia`; `GetAttachment`/`GetInlineAttachmentByFilename`
  (already `ctx`/`tenantID`-scoped from the issue #40 sweep) now resolve
  the tenant's store via `s.media.Get(ctx, tenantID)` before using it —
  these are on the real campaign-send path (`internal/manager`'s
  `attachMedia`/`applyInlineImages`), not just the upload API.
- `initCampaignManager`'s `md media.Store` param became `md *tenantMedia`;
  `newManagerStore` likewise. `cmd/main.go`'s boot sequence now builds
  `mediaResolver := newTenantMedia(core)` (after `core`, since the
  resolver holds a reference to it) and threads it into both
  `initCampaignManager` and `App{media: mediaResolver}`.
- `cmd/admin.go`'s `GetServerConfig` (the frontend's config-bootstrap
  endpoint) had the same latent bug as `a.cfg.MediaUpload.Provider`
  everywhere else — it reported the *global* boot-time provider, not the
  requesting tenant's actual one. Fixed to resolve via `a.media.Get(ctx,
  tenantID(c))` with the global config as a fallback only if resolution
  errors.

**Verified live, not just built/tested**: uploaded a real file through
`POST /api/media` and confirmed `GET /api/config` reports the correct
per-tenant `media_provider`; then created a campaign with that media
attached and ran it to completion, confirming `attachMedia`'s
`GetAttachment` call resolved through `cmd/manager_store.go`'s new
`*tenantMedia`-based path with no "unknown"/resolution errors before the
expected fake-SMTP-host send failure. All test data cleaned up
afterward. `go build`, `go vet`, `go test ./... -race` all clean.

**Remaining in #41:** OIDC per-tenant config resolution. Phase 6's
scan-side tenant awareness already shipped separately (see Phase 6 below).

### Slice 3 — OIDC per-tenant config resolution: implemented, closes #41

Same subsystem-redesign shape as slices 1-2, but structurally different in
one way: `internal/auth` is a separate package that (correctly) never
imports `internal/core` — the existing `Callbacks` struct
(`GetCookie`/`SetCookie`/`GetUser`) already bridges DB access across that
boundary for session/user lookups, so this slice extends that same
pattern rather than reaching for a `cmd/`-side resolver type like
`tenantMessengers`/`tenantMedia`.

**What shipped:**
- `internal/auth.Callbacks` gained `GetOIDCConfig func(tenantID int)
  (OIDCConfig, error)`. `auth.Config`'s `OIDC OIDCConfig` field was
  removed entirely — every previous read of `o.cfg.OIDC.*` inside
  `internal/auth` only ever happened inside `initOIDC`, and that entire
  path is now driven by the callback instead, so keeping a stale
  boot-time `OIDC` field on `Config` would just be dead weight (unlike
  `MediaUpload.Extensions` in slice 2, which stayed as a legitimate UI
  fallback).
- `Auth`'s single `provider`/`verifier`/`oauthCfg` fields replaced with a
  `map[int]*tenantOIDC` cache (mutex-protected, reusing the struct's
  existing embedded `sync.RWMutex` — the same lock already guards
  `apiUsers`, no need for a second one). `initOIDC(tenantID)` now calls
  `o.cb.GetOIDCConfig(tenantID)`, and network-discovers+caches that
  tenant's `oidc.Provider`/`verifier`/`oauth2.Config` exactly once.
  **Unlike SMTP/media, there was no existing boot-time fail-fast to
  preserve** — `auth.New()` never called `initOIDC()` eagerly even in the
  original single-tenant code; it only initialized lazily on first
  `GetOIDCAuthURL`/`ExchangeOIDCToken` call. This simplified the slice:
  no `initX(ko) (X, error)` refactor was needed the way `initSMTPMessengers`/
  `initMediaStore` needed one.
- `GetOIDCAuthURL`/`ExchangeOIDCToken` both gained a leading `tenantID`
  param; their two callers in `cmd/auth.go` (`OIDCLogin`/`OIDCFinish`)
  pass `tenantID(c)`, same helper used everywhere else.
- `cmd/init.go`'s `initAuth` builds `GetOIDCConfig` as a closure over
  `co *core.Core`: fetches `co.GetSettings(ctx, tenantID)` and maps
  `settings.OIDC.*` (already a per-tenant sub-struct since phase 5, same
  dotted-key shape as SMTP/upload settings) into `auth.OIDCConfig`.
  **`RedirectURL` is computed here, not inside `internal/auth`**: it's
  `settings.AppRootURL + "/auth/oidc"`, using that *tenant's own* root
  URL rather than a single global one — `internal/auth` has no notion of
  tenant settings beyond what this callback returns, by design (matches
  how `GetUser`'s callback fully owns the DB lookup).
- **`cmd/handlers.go`'s route registration for `/auth/oidc` was a genuine
  design decision, not just a signature change**: the original code only
  registers the route at all if the boot-time global config has OIDC
  enabled. Under true per-tenant OIDC, one tenant could enable OIDC while
  the boot-time snapshot (tenant 1's settings) has it disabled, and vice
  versa - registering routes based on a single tenant's flag doesn't fit.
  Resolved by registering the routes whenever `app.multi_tenancy_enabled`
  is true (letting `OIDCLogin`/`OIDCFinish` check each request's actual
  resolved tenant at request time and fail cleanly if that tenant hasn't
  enabled it) **or** the existing global flag is true (byte-identical
  behavior for today's default single-tenant deployments — routes stay
  absent unless configured, exactly as before).
- `cmd/auth.go`'s `renderLoginPage` (which decides whether to show the
  "Login with OIDC" button/logo) and `createOIDCUser` (auto-provisioning,
  which needs `DefaultUserRoleID`/`DefaultListRoleID`) both switched from
  reading `a.cfg.Security.OIDC.*` (global) to fetching
  `a.core.GetSettings(ctx, tenantID(c))` and reading `settings.OIDC.*`
  instead - the same class of fix as slice 2's `GetServerConfig` bug
  (global config leaking into a per-tenant-facing response).

**Verified live end-to-end against a real IdP, not a stub**: enabled OIDC
via the Settings API pointed at `https://accounts.google.com` (a real,
publicly reachable OIDC provider) with a fake client ID/secret, confirmed
the settings-update restart picked up the change (`/auth/oidc` flipped
from 404 to registered), then hit `POST /auth/oidc` with a valid nonce
cookie and confirmed the `302 Location` header was a genuine Google OAuth
URL with the correct `client_id`, a `redirect_uri` correctly derived from
tenant 1's own `AppRootURL`, and the expected `state`/`nonce` - proving
`oidc.NewProvider`'s real network discovery against Google's endpoint
succeeded and every value came from the per-tenant resolver, not a
hardcoded fallback. Restored OIDC to disabled afterward and confirmed
`/auth/oidc` returned to 404, matching the pre-test baseline. `go build`,
`go vet`, `go test ./... -race` all clean.

Issue #41 (per-tenant SMTP/media/OIDC) is now fully complete. Manager/
scan-side tenant awareness already shipped as Phase 6.

---

## Phase 7 — frontend

No tenant switcher needed under the fully-isolated-orgs model (one user, one
tenant). Changes are audit-only:

- `frontend/src/store/index.js`: confirm no cached list/campaign/subscriber
  IDs persist across a login-as-different-tenant-user flow (e.g. logout
  should clear the Pinia store, not just redirect).
- `frontend/src/api/generated/`: no changes needed — tenant scoping happens
  server-side; the client never sends or needs a tenant ID explicitly.

### Audit performed (2026-07-08): both concerns confirmed already satisfied, no code changes needed

Both checklist items verified true by inspection rather than assumed:

- **Stale Pinia state across a login/logout boundary**: `frontend/src/App.vue`'s
  `doLogout()` calls the logout API then does `document.location.href =
  uris.root` — a **hard browser navigation**, not a client-side router
  transition. This unconditionally destroys the entire JS execution
  context (all in-memory Pinia state included), regardless of tenant.
  Confirmed no Pinia persistence plugin or direct `localStorage`/
  `sessionStorage` usage anywhere in `frontend/src/store/index.js` or
  `frontend/src/main.ts` (grepped, none found) that could survive the
  reload. Symmetrically, login itself is server-rendered
  (`cmd/auth.go`'s `LoginPage`, a Go template, not a Vue route) and
  completes via an HTTP redirect into the SPA - so the SPA's Pinia store
  only ever initializes fresh after a full page load following a
  successful login. There is no code path where one session's in-memory
  state could survive into a different session, tenant or not.
- **Tenant ID never needed client-side**: grepped
  `frontend/src/api/generated/`, `frontend/src/store/index.js`, and
  `frontend/src/main.ts` for "tenant" - zero matches anywhere. Confirms
  tenant scoping is entirely transparent to the frontend, exactly as the
  design intended (resolved server-side from the subdomain, never sent
  or read by client code).

Additionally, under the subdomain-per-tenant model, two different
tenants' admin UIs are two different browser origins by construction
(`tenant-a.example.com` vs `tenant-b.example.com`) - browser storage
(including any hypothetical future Pinia persistence) is
origin-partitioned, so cross-*tenant* state leakage isn't reachable even
in principle without an explicit, deliberate change to how the frontend
stores data. The scenario the original checklist item was written to
guard against (stale state surviving a user switch) turns out to already
be prevented by the existing hard-navigation logout, independent of
multi-tenancy.

No code changes required. Phase 7 is complete.

---

## Phase 8 — public-facing route audit

Routes to check in `cmd/handlers.go`'s public group: unsubscribe page,
subscription-preferences page, campaign archive view, opt-in confirmation,
link-click redirect, tracking pixel. Pattern for each: the handler currently
does something like

```go
sub, err := app.core.GetSubscriber(0, subUUID, "")
```

Post-migration, `GetSubscriber` (and equivalent lookups) must resolve
tenant *from* the UUID's own row rather than requiring it as an input — i.e.
`SELECT tenant_id FROM subscribers WHERE uuid = $1` is itself the tenant
resolution step for public routes, then subsequent joins (e.g. rendering
list name, campaign sender branding) must filter by that resolved
`tenant_id`, not trust a second UUID/ID param without checking it belongs to
the same tenant as the first. This is the concrete thing an audit test should
assert: pass a campaign UUID from tenant A alongside a list ID from tenant B
and confirm the response 404s rather than mixing data.

### Audit performed (2026-07-08): no leak found in multi-UUID public routes, but found and fixed 3 real bugs blocking the multi-tenant onboarding path entirely

Traced every multi-UUID public route (`ViewCampaignMessage`, `LinkRedirect`,
`RegisterCampaignView`, `SubscriptionPrefs`'s `UnsubscribeByCampaign`)
against the exact concern this section describes: does a second UUID/ID
param get trusted without re-checking it belongs to the same resolved
tenant as the first? Traced through each underlying SQL statement
(`register-link-click`, `register-campaign-view`, `unsubscribe-by-campaign`)
rather than assuming - all of them run inside the same `WithTenant`
transaction and RLS scopes every table reference in a single SQL
statement to the same `app.current_tenant`, so a cross-tenant UUID in a
compound query resolves to zero rows for that sub-lookup (e.g. a
cross-tenant `campUUID` inside `register-link-click`'s CTE), not another
tenant's row. Where the affected column is nullable (`link_clicks.campaign_id`/
`subscriber_id`), the result is a mis-attributed click (recorded, but not
linked to any campaign/subscriber) - a data-quality gap, not a leak, and
not new. Where it's `NOT NULL` (`campaign_views.campaign_id`), the insert
fails a constraint and is already caught and treated as a silent no-op
(pre-existing behavior). **No cross-tenant data exposure found in any
public route.**

**However, live-testing the onboarding path itself (not the audit's
original target, but adjacent) surfaced three real, severe bugs**,
found via a real two-tenant setup using `curl -H "Host: ..."` overrides
against `127.0.0.1:9000` (no `/etc/hosts` changes needed - confirmed
`internal/tenant.Middleware` reads `c.Request().Host` directly, which
curl's `-H` fully controls) plus a throwaway non-superuser DB role for
the RLS-dependent checks the dev DB's own superuser role can't
demonstrate (same technique as phases 2/3):

1. **`cmd/auth.go`'s `doFirstTimeSetup` hardcoded the new admin user's
   `user_role_id` to the `auth.SuperAdminRoleID` constant (1) instead of
   the ID of the role it had just created.** This only ever "worked" for
   the very first tenant on an installation, because `roles.id` is a
   single sequence shared across every tenant, and the very first row
   ever inserted into that table coincidentally gets id=1. For every
   subsequent tenant, `CreateRole` returns some other ID (the sequence
   has already advanced), but `CreateUser` was still called with the
   literal constant `1` - and since `create-user`'s role-ID lookup
   (`SELECT id FROM roles WHERE id = $7 AND type = 'user'`) is itself
   RLS-scoped, role id=1 belonging to a *different* tenant silently
   resolves to zero rows, setting `user_role_id = NULL`. The result: a
   login-successful account with **zero permissions**, completely broken.
   Fixed by capturing `CreateRole`'s actual returned `ID` and using that.
2. **`App.needsUserSetup` was a single process-wide boolean**, computed
   once at boot from whether *any* tenant had *any* user, and flipped
   `false` globally the first time *any* tenant completed setup. Once
   tenant 1 has a user (true immediately after any install), every other
   tenant's `/admin/login` renders the normal (unusable - no users exist
   yet) login form instead of the first-time-setup form, with no way to
   reach `doFirstTimeSetup` through the UI at all. Fixed by removing the
   cached field entirely and adding a lightweight `Core.HasUsers(ctx,
   tenantID)` (a `has-users` `EXISTS(...)` query, cheaper than `GetUsers`'
   full role/list-role joins) that `LoginPage` now calls per-request,
   per-tenant.
3. **Two pre-existing single-tenant `UNIQUE` indexes had no `tenant_id`
   dimension and actively blocked (not just weakened isolation for)
   legitimate multi-tenant usage**, found while root-causing bug #1
   above: `roles (type, name) WHERE name IS NOT NULL` - every tenant's
   setup creates a role literally named "Super Admin", so the *second*
   tenant ever to run setup hard-fails on a duplicate-key violation
   before bug #1 even gets a chance to matter; and `templates (is_default)
   WHERE is_default = true` - only the first tenant to ever mark a
   template default could have one, with every other tenant's attempt to
   set their own default template also hard-failing. Both are additive,
   single-tenant-era constraints from `v4.0.0`/early `schema.sql`,
   predating phase 1's `tenant_id` columns - phase 1 deliberately
   deferred touching pre-existing constraints for exactly this kind of
   follow-up. New migration `v6.8.0` widens both indexes to `(tenant_id,
   ...)`. Two *other* known global-uniqueness gaps (`subscribers.email`,
   `links.url`) were already found and documented earlier this session
   and deliberately left alone here - they cause soft cross-tenant
   collisions on human-chosen values, not hard onboarding failures, and
   widening them changes `ON CONFLICT` semantics several existing queries
   rely on, needing their own dedicated review.

All three were surfaced to the user via `AskUserQuestion` given their
severity (onboarding-blocking, not just isolation-weakening) before
fixing - user chose "fix now" each time. **Verified**: `go build`checked
the Go fix; the role-ID and onboarding-gate fixes were verified via a
direct SQL/RLS simulation under a real non-superuser role reproducing
`doFirstTimeSetup`'s exact operations (confirmed the *old* code's failure
mode reproduces exactly as diagnosed, then confirmed the *fixed* code's
operations succeed with a correctly non-null `user_role_id`) - a full
HTTP-level test of `doFirstTimeSetup` itself wasn't possible in this dev
environment since the app's own DB connection is a superuser (RLS-exempt,
same documented limitation from phases 2/3 onward), which is why
`HasUsers`'s per-tenant correctness had to be verified the same way. The
schema migration (`v6.8.0`) was verified directly: ran it against the dev
DB, confirmed via `psql` both indexes now include `tenant_id`, and
confirmed the existing single-tenant login/template/role flows still work
normally afterward. All test data (throwaway DB roles, temp tenant,
`config.toml`'s temporary `multi_tenancy_enabled`/`root_domain` overrides)
cleaned up and reverted to baseline afterward.

---

## Issue #40 — threading tenantID through internal/core

Split out from phase 4 (see that phase's status note) once investigation
found it touches 107 exported `Core` methods / 150 call sites, none of
which take a `context.Context` or run through a transaction today. Being
done file-by-file; each slice's findings are recorded here so the next
slice starts informed rather than re-discovering the same issues.

### Slice 1 — `subscribers.go` (issue #40, first slice): implemented

Confirmed the 3 execution shapes anticipated when scoping this work (see
this file's earlier "Tenant ID source at call sites" note under phase 4):
simple prepared-statement calls (rebind via a `stmtx` helper, below),
self-managed raw-SQL transactions (replace the method's own `BeginTxx`
with `WithTenant`), and one long-lived iterator (`ExportSubscribers`, left
with a `TODO(#40)` per the deferred-scoping decision).

**Critical finding, applies to every future slice:** `sqlx.Tx.Stmtx()`
does **not** propagate the `.Unsafe()` flag. `cmd/init.go` opens the pool
`*sqlx.DB` with `.Unsafe()` (several models don't map every returned
column — `internal/core/campaigns.go` already has a comment about this for
`models.Campaigns`, and now *every* table has an unmapped `tenant_id`
column since phase 1 unless its model struct happens to declare one).
`BeginTxx` correctly copies `.Unsafe()` onto the resulting `*sqlx.Tx`
(confirmed in sqlx v1.4.0 source: `&Tx{..., unsafe: db.unsafe, ...}`), but
`Tx.Stmtx()` does not carry it onto the `*sqlx.Stmt` it derives
(`&Stmt{Stmt: tx.Stmt(s), Mapper: tx.Mapper}` — no `unsafe` field set).
Symptom: `missing destination name tenant_id in *models.X` errors that
only appear on the tx-rebound path — `tx.Select(...)`/`tx.Get(...)` called
*directly* (shape 2) work fine since they inherit `.Unsafe()` correctly;
only statements rebound via `Stmtx` (shape 1) hit this.

**Fix, already in place for all future slices to reuse:** `internal/core/tenant.go`
now has a package-level `stmtx(tx *sqlx.Tx, stmt *sqlx.Stmt) *sqlx.Stmt`
helper that wraps `tx.Stmtx(stmt).Unsafe()`. **Use `stmtx(tx, c.q.X)`, never
`tx.Stmtx(c.q.X)` directly**, for every future slice's shape-1 methods.

**Other changes needed for shape 2 and the shared query-template helpers:**
- `Core.WithTenant`'s signature gained a `*sql.TxOptions` parameter (`nil`
  = `BeginTxx`'s own default) so shape-2 methods that need
  `&sql.TxOptions{ReadOnly: true}` as a security control (subscribers.go's
  arbitrary-query features — see the code comment there) don't lose that
  guarantee to the helper.
- `models/queries.go`'s `compileSubscriberQueryTpl`/`ExecSubQueryTpl`
  (shared by `subscribers.go` and `subscriptions.go`) had their `db
  *sqlx.DB` parameter widened to `db sqlx.Execer` (satisfied by both
  `*sqlx.DB` and `*sqlx.Tx`), and `compileSubscriberQueryTpl`'s own nested
  `BeginTxx`/`Rollback` was removed (confirmed safe: its "dry run" flag
  just changes a `LIMIT` clause on a pure `SELECT`, no mutation possible
  either way) — a transaction can't `BeginTxx` a second time inside itself.
  Migrated callers (`subscribers.go`) now pass the open `tx`; unmigrated
  callers (`subscriptions.go`, not yet in scope) keep passing `c.db`
  unchanged and are unaffected.
- **Tenant ID at call sites**: settled on always reading the
  subdomain-resolved tenant (`c.Get(models.TenantCtxKey)`, via a new
  `tenantID(c echo.Context) int` helper in `cmd/handlers.go`) rather than
  `auth.GetUser(c).TenantID` — simpler (one source, not two depending on
  route type) and already proven consistent by phase 4's auth cross-check.

**Verified:** full CRUD (create/view/edit/blocklist/delete, both
individual and bulk), the advanced arbitrary-query search path, and CSV
export all work identically under the default (multi-tenancy disabled)
config — tested live via the browser and `curl`. Real cross-tenant
isolation was verified in two layers: (1) confirmed **RLS itself** blocks
cross-tenant rows when queried via a non-superuser role with
`app.current_tenant` set (same throwaway-role technique as phase 2/3);
(2) confirmed the full HTTP path (two tenants, two logged-in users, real
subdomains) does **not** show isolation in this specific dev environment,
but only because — as already flagged in phase 2 — the dev DB role
(`listnun-dev`) is a superuser and superusers always bypass RLS
regardless of policy. This is a pre-existing, already-documented
dev-environment limitation, not a defect in this slice's code; the
per-layer test above is what actually confirms the code is correct.
**Any real deployment of this feature must use a non-superuser,
non-table-owner application role**, per the RLS "verify before merging"
checklist in phase 2.

### Slices 2-9 (2026-07-08): `campaigns.go`, `subscriptions.go`,
`bounces.go`, `lists.go`, `media.go`, `roles.go`, `templates.go`,
`users.go` — implemented, plus a cross-cutting RLS/INSERT gap fix

What started as slice 2 (`campaigns.go`) surfaced a schema-wide gap before
any code was written: Phase 2's RLS policy (`v6.5.0`) has no `FOR`/`WITH
CHECK` clause, so an `ALL`-command policy's `WITH CHECK` defaults to the
same expression as `USING` — every `INSERT` must supply a `tenant_id`
matching `current_setting('app.current_tenant')` under a real
(non-superuser) role, or the insert is rejected. Every `INSERT` in every
`queries/*.sql` file relied on the column's `DEFAULT 1` from phase 1 with
no exceptions. User chose the broadest option when asked: fix every
`INSERT` across the whole schema in one pass rather than scoping to
`campaigns.go` alone. Doing that requires `WithTenant` threading anyway
(the only way `app.current_tenant` gets set), so this ended up completing
slices 2-9 of issue #40 as a byproduct of the INSERT fix, not as
separately-scoped work.

**Shape notes specific to these slices** (beyond the 3 shapes documented
in slice 1):
- **`bounces.go`'s `RecordBounce`** is called from bounce
  webhooks/POP3-mailbox polling (`internal/bounce`), which have no
  request-scoped tenant at all — it does **not** go through `WithTenant`.
  `queries/bounces.sql`'s `record-bounce` instead derives the inserted
  row's `tenant_id` from the resolved subscriber's own `tenant_id`
  (`sub` CTE now selects it), which works under RLS's permissive-when-unset
  fallback regardless of session state.
- **`internal/subimporter` (bulk CSV import)** and
  **`cmd/manager_store.go`'s `CreateLink`/`GetAttachment`/
  `GetInlineAttachmentByFilename`** both call raw queries/`Core` methods
  from outside any HTTP request (`internal/manager`'s campaign-send
  pipeline, the importer's own goroutine), bypassing `WithTenant`
  entirely. Fixed by threading tenant identity through their own existing
  data instead of `WithTenant`: `subimporter.SessionOpt` gained a
  `TenantID` field (set by the HTTP handler that starts an import
  session, `json:"-"` so it can't be client-supplied), and
  `manager.Store`'s `CreateLink`/`GetAttachment`/
  `GetInlineAttachmentByFilename` gained `ctx`/`tenantID` params sourced
  from `models.Campaign.TenantID` (already present since #41 slice 1),
  called with `context.Background()` since these are background
  worker paths, matching the existing `getMessenger` convention.
- **`users.go`'s `LoginUser` had zero tenant filtering** — found while
  threading this file, not something being looked for. `username` is
  still a global `UNIQUE` constraint (deferred in phase 1), so without
  scoping, valid credentials for tenant A would also successfully log in
  on tenant B's subdomain if a same-named account existed there. Fixed by
  routing it through `WithTenant` like everything else. This is a
  different bug from what `internal/auth.tenantMismatch` guards (replay
  of an *existing* session/token against the wrong tenant) — that check
  never ran during the login call itself.
- **Two user lookups deliberately kept unscoped**, each with an explicit
  comment: `Core.GetUserUnscoped` (used only by `cmd/init.go`'s
  `auth.Callbacks.GetUser`, the session/API-token decode callback — it
  must find the user regardless of tenant so `tenantMismatch` can compare
  and reject with a *distinct* 403, rather than RLS making a cross-tenant
  replay silently indistinguishable from "user not found") and
  `Core.GetAllUsersUnscoped` (used only by `cmd/users.go`'s `cacheUsers`,
  which populates the in-memory API-token cache — needs every tenant's
  API users since an incoming token's tenant isn't known until *after*
  the token itself is matched).
- **`cmd/init.go`'s `initTxTemplates`** (boot-time tx-template cache
  warmup) was pinned to tenant 1 as a stopgap immediately after
  `manager.CacheTpl` gained a `tenantID` param (needed to compile), then
  properly fixed once `templates.go` itself was threaded: it now calls
  the same `Core.GetActiveTenantIDs()` phase-6's `scanCampaigns` uses and
  loops per tenant.
- **`install.go`'s many raw `q.X.Exec`/`q.X.Get` calls** (seed data —
  lists, subscribers, templates, campaign, role, user) all needed a
  trailing tenant literal `1` added by hand, since install always seeds
  the one tenant that exists before any provisioning flow (phase 9) is
  built. These are **not caught by `go build`** — prepared-statement
  `Exec`/`Get` take variadic `...any`, so a wrong argument count is a
  runtime `pq` error, not a compile error.
- **A real bug caught only by live-testing, again**: `CreateRole`/
  `CreateListRole` were written accepting the new `ctx`/`tenantID`
  params but the SQL call itself forgot to actually pass `tenantID` as
  the query's new trailing arg. `go build`/`go vet` were clean; a live
  `curl POST /api/roles/users` immediately surfaced `sql: expected 4
  arguments, got 3`. Fixed and reverified live.

**Verified live end-to-end** against the dev server (tenant 1, default
single-tenant config): created a list, a subscriber with a list
subscription, a template, a user role, a list role (with list
permissions), a user, a media upload, and a campaign (with a list and a
media attachment); ran the campaign draft → running → finished and
confirmed `campaign_media`, `campaign_lists`, and `links` (via a real
`@TrackLink`-tagged URL — the one exercising the previously
Core-bypassing `CreateLink` path) all wrote rows with `tenant_id=1`
before the expected fake-SMTP-host send failure. All test data deleted
afterward. `go build`, `go vet`, and `go test ./... -race` all clean. No
new migration — `tenant_id` columns already existed from phase 1; this
was purely a `queries/*.sql` + `internal/core`/`cmd` fix.

**Remaining in #40:** only `internal/core/dashboard.go`
(`GetDashboardCharts`/`GetDashboardCounts`, backed by global materialized
views) — tracked under this doc's "Matview refresh cost" open question,
not a plain threading job like the rest of #40 since it needs a
`tenant_id` dimension added to the matviews themselves first.

### `dashboard.go` (2026-07-08): implemented, closes issue #40

Turned out to be more than a threading job: `mat_dashboard_counts`,
`mat_dashboard_charts`, and `mat_list_subscriber_stats` each computed a
single **global** row with no `tenant_id` column at all. Worse than the
INSERT-rejection shape of the earlier fix, this was a **live cross-tenant
read leak** — `query-subscribers-count-all` falls back to
`mat_list_subscriber_stats`'s `list_id=0` "all subscribers" row whenever
a request has no list filter (the common Subscribers-page-load case), so
every tenant's unfiltered subscriber total was silently the sum across
*every* tenant. Same shape for the dashboard's totals and charts. Asked
the user how to scope this given the severity; chose to fix immediately.

**Migration `v6.7.0`** rewrites all three materialized views to compute
one row per tenant, driven by `SELECT ... FROM tenants t` with correlated
per-tenant subqueries (dashboard views) or `GROUP BY tenant_id`
(`mat_list_subscriber_stats`). Each view's unique index (required for
`REFRESH MATERIALIZED VIEW CONCURRENTLY`, which `Core.RefreshMatView`
already uses) is widened to lead with `tenant_id`. `queries/misc.sql`'s
`get-dashboard-charts`/`get-dashboard-counts` and
`queries/subscribers.sql`'s `query-subscribers-count-all` all gained an
explicit `tenant_id` filter param. `internal/core/dashboard.go`'s two
methods now take `ctx`/`tenantID` like everything else in `internal/core`
and route through `WithTenant`. The refresh mechanism is unchanged — one
`REFRESH` statement still refreshes every tenant's row in one shot,
matching the "keep the global cadence, filter at query time" default this
doc's Open Questions section already recommended.

**Found the same class of bug a second time in the same session**:
`GetDashboardCharts`/`GetDashboardCounts` correctly opened a `WithTenant`
transaction but the `.Get(&out, ...)` call forgot to actually pass
`tenantID` as the query's new `$1` arg (identical shape to the
`CreateRole`/`CreateListRole` bug from the INSERT fix above). `go build`/
`go vet` stayed clean; caught immediately via live `curl` (`sql: expected
1 arguments, got 0`) against `/api/dashboard/counts`. Fixed and
reverified live.

**Verified live end-to-end**: ran the migration against the dev DB,
confirmed via `psql` that each matview produces exactly one correctly-
scoped row per tenant, then hit `/api/dashboard/counts`,
`/api/dashboard/charts`, and `/api/subscribers` over HTTP and confirmed
tenant-1-only numbers. `go build`, `go vet`, `go test ./... -race` clean.

---

## Phase 9 — operator API

New route group, entirely separate from the tenant-facing app: no session/JWT
auth, no RLS-scoped DB role.

### `internal/operator/` (new package, mirrors `internal/auth/` structure)

```go
package operator

// Middleware checks a static bearer token from config against every request
// to /api/operator/*. No per-operator identity in v1 — see multi-tenancy.md's
// decisions log for why (audit trail is a known gap, acceptable for v1).
func Middleware(token string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            got := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
            if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
                return echo.NewHTTPError(http.StatusUnauthorized)
            }
            return next(c)
        }
    }
}
```

Config: `config.toml` gains an `[operator]` section (`token = "..."`), or
`LISTNUN_operator__token` env var, following the existing `LISTNUN_`
double-underscore convention.

### DB role

A new Postgres role (e.g. `listnun_operator`) with `BYPASSRLS`, distinct
from the tenant-app pool's role (which must never have it — see phase 2's
verification query). This is the second consumer of a bypass role, alongside
migrations/backup jobs.

### Routes (`cmd/handlers.go`, new group under `/api/operator`)

```go
g := srv.Group("/api/operator", operator.Middleware(app.constants.OperatorToken))
g.GET("/tenants", handleOperatorListTenants)
g.GET("/tenants/:id", handleOperatorGetTenant)
g.POST("/tenants", handleOperatorCreateTenant)
g.PUT("/tenants/:id/status", handleOperatorUpdateTenantStatus)
```

- `handleOperatorListTenants` / `GetTenant`: tenant row + basic counts
  (`SELECT COUNT(*) FROM users WHERE tenant_id = $1`, same for subscribers)
  for a support/billing dashboard.
- `handleOperatorCreateTenant`: inserts a `tenants` row + an initial admin
  `users` row in one transaction. **Open question (see multi-tenancy.md):**
  how the initial admin's password/invite is communicated — placeholder
  behavior for v1 is to return a one-time setup-link token in the response
  body, expiring after first use, rather than emailing anything (email
  requires that tenant's own SMTP config, which doesn't exist yet for a
  brand-new tenant).
- `handleOperatorUpdateTenantStatus`: validates the new status is one of
  `active`/`suspended`/`disabled`, updates `tenants.status`. Suspended/disabled
  tenants: the phase-4 resolution middleware renders the "workspace
  unavailable" page for any request on their subdomain, and phase 6's
  `scanCampaigns` skips them on the next tick (no immediate kill of an
  in-flight send — same "finish current batch, don't start new ones"
  semantics as pausing a single campaign today).

### Implemented (2026-07-08)

Built close to this section's sketch, with a few things resolved
concretely during implementation:

- **`internal/operator/operator.go`**: the middleware as sketched, plus a
  404 (not 401) when the token isn't configured at all, so an unconfigured
  Operator API is indistinguishable from a route that doesn't exist -
  matching how the group is only registered in `cmd/handlers.go` when
  `a.operator != nil`.
- **Config**: `[operator]` gained two more fields beyond the sketch's
  `token` - `db_user`/`db_password`, for the separate BYPASSRLS
  connection (see below). All three via `ko.Unmarshal("operator",
  &c.Operator)` in `initConstConfig`, `LISTNUN_operator__*` env vars
  work automatically (the env-var provider is fully generic, not
  per-section).
- **The BYPASSRLS DB role turned out to be a genuine design fork, not a
  given**: `tenants` itself has no RLS policy at all (confirmed earlier
  this session - it's the table that *identifies* tenants), so the
  normal app role can already read/write it freely. The only thing that
  actually needs cross-tenant visibility is `GetTenants`/`GetTenant`'s
  aggregate `user_count`/`subscriber_count` columns (one row per tenant,
  every tenant visible at once - impossible under RLS scoped to a single
  `current_setting('app.current_tenant')` at a time). Raised this to the
  user as an explicit choice (skip BYPASSRLS entirely and loop
  per-tenant via the existing `WithTenant` helper, vs. build the
  connection as originally sketched) - **user chose to build it as
  planned**, so `cmd/operator.go`'s `initOperatorDB` opens a second
  `*sqlx.DB` reusing `[db]`'s host/port/dbname/sslmode but a different
  role/password, and is nil (feature off) unless both `operator.token`
  and `operator.db_user` are set.
- **`queries/operator.sql`** (new file) holds the 4 queries, prepared
  against the *operator* connection via the same `goyesqlx.ScanToStruct`
  mechanism every other query set uses - just pointed at a different
  `*sqlx.DB`, reusing the same parsed `qMap` (promoted from a
  function-local to a package-level var in `cmd/main.go` so both the
  main and operator preparation calls can see it).
- **Creating the initial admin deliberately does *not* use the
  BYPASSRLS connection**: `operatorStore.CreateTenant` inserts the
  `tenants` row via the operator connection, but creates the "Super
  Admin" role and the admin user via the *normal* tenant-app `Core`
  (`Core.CreateRole`/`Core.CreateUser`, called with the newly-allocated
  tenant's own ID) - a same-tenant write using a real tenant ID needs no
  elevated privilege, and reusing already-hardened Core methods (the
  exact ones `cmd/auth.go`'s `doFirstTimeSetup` uses, including this
  session's role-ID-capture fix) keeps the two provisioning paths
  consistent rather than reimplementing user/role creation against a
  different connection.
- **Setup-link token flow**: `CreateTenant` generates a
  `generateRandomString(tmpAuthTokenLen)` token, stores
  `{TenantID, Email}` against it in `internal/tmptokens` (7-day TTL, the
  same in-memory/process-lifetime store already used for password-reset
  and 2FA tokens - same accepted limitation: a process restart between
  provisioning and redemption invalidates a pending link). New public
  route `GET/POST /admin/operator-setup` (registered unconditionally -
  harmless when the Operator API is off, since no token could ever
  exist to redeem) renders a set-password form
  (`static/public/templates/operator-setup.html`, structurally identical
  to the existing password-reset template) and, on submit, sets the
  user's password and flips `PasswordLogin: true` via the ordinary
  `Core.UpdateUser`. The link naturally resolves to the correct tenant
  because it points at that tenant's own subdomain
  (`https://<slug>.<root_domain>/admin/operator-setup?token=...`) and
  the existing subdomain-resolution middleware does the rest - no
  special-casing needed.
- **Found while wiring routes**: `internal/tenant.Middleware` is applied
  globally via `srv.Use`, so it was rejecting every `/api/operator/*`
  request outright (no subdomain can ever match an inherently
  cross-tenant path). Added an explicit `/api/operator/` path-prefix
  exemption inside the middleware, falling through to the same
  `defaultTenant` stub used when multi-tenancy is disabled entirely -
  correct since none of the operator handlers ever read the resolved
  tenant from context.
- **Found via live-testing**: a duplicate slug surfaced as a generic 500
  instead of a clean error. Fixed by checking for
  `pq.Error.Constraint == "tenants_slug_key"` and returning 409.
  Separately, the setup-link URL's scheme extraction was originally
  hand-rolled string slicing (`RootURL[:strings.Index(...)+3]`), which
  would panic on a malformed/empty `RootURL` - replaced with
  `net/url.Parse` before this shipped.

**Verified live end-to-end** with a real non-superuser `BYPASSRLS` role
(confirmed via `pg_roles.rolbypassrls`, not another superuser): token
auth (missing/wrong/correct), tenant listing with correct cross-tenant
counts, tenant creation (confirmed the created role has every
permission, scoped to the new tenant, and the new user's
`user_role_id` correctly points at it - not `NULL`, not tenant 1's role
1), the full setup-link flow (GET renders the form, POST sets the
password, the new admin then successfully logs in on their own
subdomain with real credentials), status transitions (`suspended` ->
503 workspace-unavailable on that subdomain, `active` -> restored), and
the duplicate-slug 409. Confirmed the Operator API is a true no-op when
unconfigured (route absent, existing single-tenant login/config
untouched). All test tenants, throwaway DB roles, and temporary
`config.toml` overrides cleaned up and reverted to baseline. `go build`,
`go vet`, `go test ./... -race` all clean.

**Known v1 limitations, matching the doc's original framing**: no
per-operator identity/audit trail (single shared token); setup tokens
don't survive a process restart (same as password-reset tokens);
`CreateTenant`'s two writes (tenant row, then role+user) aren't
transactionally atomic across the two DB connections - a failure
between them can leave an orphaned tenant with no admin, recoverable by
an operator noticing and retrying (a different slug, or manual
cleanup). None of these block v1 usage; they're accepted tradeoffs the
design doc already called out.

---

## Testing strategy across all phases

- Unit: every `internal/core` function gets a table-driven test with ≥2
  tenants seeded, asserting tenant B's data never appears in tenant A's
  query results.
- Integration: the phase 3 concurrency spike becomes a permanent test in
  `internal/core/tenant_test.go`, run in CI.
- Migration: `internal/migrations/v6.4.0_test.go` — apply migration to a
  seeded pre-migration single-tenant DB fixture, assert all rows land in
  tenant 1 and constraints are correctly re-scoped.
