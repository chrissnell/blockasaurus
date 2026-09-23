# Upstream sync plan

Blockasaurus is a fork of [0xERR0R/blocky](https://github.com/0xERR0R/blocky). This document
records the measured state of the divergence, the plan for the next sync, and the set of files
we have intentionally forked so future syncs are cheaper.

Last measured: 2026-09-23, against upstream `main` @ `2bb9b70` (2026-09-21).

## 1. Measured divergence

| Metric | Value |
| --- | --- |
| Merge base | `d459311` (2026-02-27) |
| Commits we are ahead | 211 |
| Commits we are behind | 220 (113 dependency bumps, ~107 substantive) |
| Our diff vs merge base | 214 files, +29,731 / −2,382 |
| Upstream diff vs merge base | 305 files, +39,496 / −5,131 |
| Files both sides touched | 57 |

Reproduce with:

```bash
git remote add upstream https://github.com/0xERR0R/blocky.git
git fetch upstream main
MB=$(git merge-base origin/main upstream/main)
git rev-list --left-right --count $MB...origin/main
git rev-list --left-right --count $MB...upstream/main
```

## 2. Real conflict surface

A trial `git merge upstream/main` was run on 2026-09-23. It produced **41 conflicted paths**:
31 content conflicts and 10 delete/modify conflicts. The remaining 264 upstream-changed files
applied cleanly.

### Content conflicts, by hunk count

| Hunks | File | Nature |
| --- | --- | --- |
| 9 | `go.mod` | Go version + dependency sets. Regenerate, do not hand-merge. |
| 8 | `server/server.go` | **Hard.** Upstream restructured resolver construction and added HTTP/3; we added auth, config store, log broadcaster, stats collector, admin router. |
| 5 | `api/mocks_test.go` | Generated-ish test mocks; regenerate after the API interface settles. |
| 4 | `util/edns0_test.go` | Upstream EDNS0 cookie + OPT-record fixes vs our EDNS0 changes. |
| 4 | `resolver/blocking_resolver.go` | **Hard.** Upstream rewrote client-group matching; we added client-group attribution and group-disable filtering into the same function. |
| 4 | `go.sum` | Regenerate. |
| 4 | `.goreleaser.yml` | Our packaging vs upstream release config. |
| 3 | `config/config.go` | **Hard.** Upstream added per-field doc comments, JSON-schema generation and 4 new config sections; we made `upstreams` YAML-rejecting and added admin ports. |
| 3 | `cmd/root.go` | Upstream added `cache`/`stats` subcommands and API/DNS host+port globals; we removed several subcommands and added `user`. |
| 3 | `README.md` | Branding. Trivial. |
| 3 | `.github/workflows/release.yml` | Our release pipeline. Keep ours. |
| 2 | `server/server_endpoints.go` | **Hard.** Route ownership, including the `/api/stats` collision (§3.1). |
| 2 | `querylog/database_writer_test.go` | Upstream SQLite/dnstap targets. |
| 2 | `e2e/containers.go` | Upstream moved `docker/docker` → `moby/moby/api`. |
| 2 | `docs/configuration.md` | Take upstream, re-apply our sections. |
| 2 | `cmd/serve.go` | Startup wiring. |
| 2 | `api/api_interface_impl.go` | Upstream added the `/stats` operation to the generated API. |
| 2 | `Makefile` | Our web/ui build steps vs upstream tooling bumps. |
| 1 each | `web/index.html`, `server/server_test.go`, `server/http.go`, `resolver/query_logging_resolver.go`, `resolver/metrics_resolver.go`, `metrics/metrics_test.go`, `docs/installation.md`, `docs/index.md`, `docs/config.yml`, `docs/api/openapi.yaml`, `config/upstreams.go`, `cmd/root_test.go`, `api/api_interface_impl_test.go` | Mostly branding or single-site integration. |

### Delete/modify conflicts (10)

We deleted these; upstream still edits them. Resolution for all ten is **re-delete** (`git rm`),
unless a decision in §4 says otherwise.

`.github/workflows/build-bin.yml`, `close_stale.yml`, `codeql-analysis.yml`,
`dependabot-auto-merge.yml`, `development-docker.yml`, `docs.yml`, `goreleaser-test.yml`,
`makefile.yml`, `mirror-repo.yml`, `cmd/lists.go`

Our other intentional deletions (`cmd/blocking.go`, `cmd/cache.go`, `cmd/query.go` and their
tests, `.github/workflows/fork-sync.yml`, `pr-title.yml`) were untouched by upstream and stay
deleted automatically.

### The dangerous set: files that merged *cleanly* but changed semantically

These 16 files were modified by both sides and produced **no conflict markers**. A clean textual
merge here is not evidence of a correct merge — each needs a read of the upstream change against
our change:

`model/models.go`, `querylog/database_writer.go`, `querylog/writer.go`,
`resolver/caching_resolver.go`, `resolver/dnssec/validator.go`, `util/edns0.go`,
`resolver/query_logging_resolver_test.go`, `config/config_test.go`, `e2e/integration_test.go`,
`e2e/metrics_test.go`, `Dockerfile`, `.gitignore`, `docs/additional_information.md`,
`docs/interfaces.md`, `docs/network_configuration.md`, `docs/prometheus_grafana.md`

`resolver/dnssec/validator.go` is the highest-stakes entry: upstream landed a DNSSEC validation
bypass fix there (GHSA-x845-2f78-7v36) plus three other DNSSEC correctness fixes.

## 3. Why this is more than a textual merge

The original estimate treated the fork as "mostly additive." The trial merge says otherwise in
five specific places. These are the items that actually drive the schedule.

### 3.1 Duplicate statistics subsystems — guaranteed collision

Both sides independently built a stats subsystem in the drift window.

| | Ours | Upstream |
| --- | --- | --- |
| Collector | `pkg/statscollector` | `stats/` |
| Persistence | `configstore/stats.go` (SQLite) | in-memory, 24h window |
| Wiring | `server/server_stats.go` | `resolver/stats_resolver.go` |
| HTTP | `r.Get("/api/stats", …)` plus 7 `/api/stats/*` series routes | generated OpenAPI `GET /stats`, mounted under `/api` |
| CLI | — | `cmd/stats.go` (`blocky stats`) |
| Config | config store | `statistics:` config section |

After the merge both definitions exist. They are different Go packages so the tree still
compiles, but both sides register `GET /api/stats`, and chi panics on a duplicate route pattern
registered on the same router. **This needs an explicit owner decision before resolution starts**
(§4, D1), not an improvised fix at conflict-resolution time.

### 3.2 Resolver construction was refactored underneath us

Upstream's redis write-through cache refactor (#2025) changed `createQueryResolver` to take a
`resolver.CacheDecorator`, and dropped the `redisClient` parameter from `NewBlockingResolver`.
We pass `redisClient` into `NewBlockingResolver` and a `logstream.Broadcaster` into
`NewQueryLoggingResolver`. Both of our injection points have to be re-established on top of the
new signatures — this is a rewrite of our wiring, not a hunk pick.

### 3.3 Client-group matching was rewritten in the function we patched

Upstream's #2103 and #2146 replaced the `clientGroupsBlock` map scan with pre-classified
structures (`byID` exact map, parsed CIDR list, FQDN list) and a `scheduledGroup` type from
schedule-based blocking (#2037). We modified that same function to record `request.ClientGroup`
and to filter disabled groups. Our behavior must be re-implemented against upstream's new data
model; taking either side wholesale loses something.

### 3.4 Config ownership is a design conflict

We made `Config.Upstreams` `yaml:"-"` with a sentinel that rejects a legacy `upstreams:` block,
moving upstream configuration into the SQLite config store and the web UI. In the same window
upstream added schema-driven config validation and JSON-schema generation
(`tools/schemagen`, `docs/config.schema.json`, #2066), which reflects over every `Config` field,
plus four new sections (`statistics`, `http3`, `rateLimit`, `rebindingProtection`). Our sentinel
field will flow into the generated schema unless it is handled deliberately.

### 3.5 Toolchain and generated-artifact churn

- Go 1.26.1 → 1.26.2; golangci-lint → v2.12.2, with additional linters enabled upstream (#2073).
- `oapi-codegen/runtime` 1.1.2 → 1.7.0 — `api/*.gen.go` must be regenerated, not merged.
- `docker/docker` → `moby/moby/api` in the e2e harness.
- New upstream deps: `quic-go` (DoQ/DoH3), `invopop/jsonschema`, `jedib0t/go-pretty`,
  `dnstap/golang-dnstap`, `pires/go-proxyproto`.
- Generated files to rebuild rather than merge: `api/api_{client,server,types}.gen.go`,
  `docs/config.schema.json`, all `go-enum` outputs (`config/`, `log/`, `lists/`, `model/`,
  `resolver/dnssec/`).

## 3a. Guardrails

Two checks exist so that "did we lose anything?" is a test result rather than a
judgement call. **Run both before the merge and again after every resolution
phase** — a guard that is only consulted at the end tells you something broke
without telling you where.

```bash
go test ./server -run 'TestAPIContract|TestAPISpecContract'
make check-fork-additions
```

### `TestAPIContract` — routing and auth

`server/api_contract_test.go` builds the real production router
(`createHTTPRouter`, plus `registerDoHEndpoints` for combined-port mode), walks
it, and records for every route the method, path pattern and **middleware
chain**, against `server/testdata/api_contract.golden`.

The middleware chain is the part that matters most. Moving a route out of the
authenticated group strips `RequireAuth` without changing its method or path —
the likeliest and most damaging thing a badly resolved `Group` block can do —
and the golden catches it as:

```text
- GET     /api/version    [RequireAuth RequireCSRFHeader RequireAdminForMutations]
+ GET     /api/version    [public]
```

### `TestAPISpecContract` — payloads

`server/api_spec_contract_test.go` locks every operation and every component
schema (property names, types, required fields) in `docs/api/openapi.yaml` and
`docs/api/openapi-config.yaml` against
`server/testdata/api_spec_contract.golden`. `openapi.yaml` is an upstream file
carrying our edits, so it is a prime candidate for being reverted wholesale.

It deliberately ignores prose. Descriptions and branding will legitimately
change during this merge, and a guard that fires on a reworded sentence gets
regenerated without being read.

### `make check-fork-additions` — file survival

Verifies every path in `.fork-additions` (the files present here and absent
upstream) still exists. This is what catches a delete/modify conflict resolved
toward upstream. The manifest lists itself and both contract tests, so deleting
the guard is itself a failure. `make check-fork-additions-sync` reports when the
manifest has gone stale and prints the diff to apply.

### Reading a golden diff

A diff is never automatically a bug — but it is always a change to the contract
`web/ui` and any API consumer depend on. Three legitimate reasons to regenerate:

1. We deliberately added an endpoint.
2. We deliberately removed one, and the UI no longer calls it.
3. An upstream fix changed a schema we decided to adopt.

Anything else is the merge eating our work. Regenerate only with:

```bash
go test ./server -run 'TestAPIContract|TestAPISpecContract' -update-api-contract
```

and call the change out explicitly in the pull request.

### What the guardrails do not cover

Say this out loud so nobody trusts them further than they reach:

- **Fork edits to upstream files.** `.fork-additions` only catches deletions of
  fork-*only* files. The likelier casualty in a 220-commit merge is one of the
  ~18 upstream files carrying our patches (§7) being reverted by a sloppy hunk
  resolution. Nothing fails automatically there — that is what the §7 register
  and a `git diff` against the pre-merge tag are for.
- **Schemas behind the operations.** `TestAPISpecContract` locks the declared
  component schemas, not inline request/response bodies or handler behavior.
- **Middleware behavior.** Only the identity and order of the chain is recorded.
- **The Svelte UI.** Only that its files still exist and that it builds.

## 4. Decisions required before merging

These are owner calls. Resolving them at conflict time produces arbitrary outcomes.

| # | Decision | Recommendation |
| --- | --- | --- |
| D1 | Stats: keep ours, adopt upstream's, or both? | Keep ours (it backs the dashboard's overtime/top-clients series, which upstream's does not provide). Do not mount upstream's `/stats` operation; keep `stats/` and `resolver/stats_resolver.go` out of the chain, or drop them. Revisit later as a single subsystem. |
| D2 | `upstreams:` YAML sentinel — keep rejecting, or accept as read-only fallback? | Keep rejecting; exclude the sentinel field from schema generation. |
| D3 | Re-delete `cmd/lists.go` and keep `cache`/`stats` subcommands out of `cmd/root.go`? | Yes — these are UI-driven in Blockasaurus. |
| D4 | Re-delete the 9 upstream GitHub workflows? | Yes. |
| D5 | Which new upstream features ship enabled in this sync? DoQ, DoH3, per-client rate limiting, DNS rebinding protection, PROXY protocol, SQLite/dnstap query log, schedule-based blocking, on-disk list cache. | Merge the code, leave each at upstream defaults, ship no UI/config-store plumbing in this sync. Each gets its own follow-up issue. |
| D6 | Does `docs/` stay branded Blockasaurus, or track upstream? | Take upstream content, re-apply branding as a final pass. |

## 5. Plan

Each phase ends at a gate. Do not start a phase before its gate passes.

| Phase | Work | Gate | Est. |
| --- | --- | --- | --- |
| 0. Baseline | Tag `pre-upstream-sync-v0.34.38`. Record current behavior: `go test ./...`, e2e suite, and a captured set of DNS answers (blocked/allowed/custom/conditional/DNSSEC/EDNS0) plus dashboard screenshots. This is what "did we regress?" is measured against later. | Baseline artifacts committed to `scratch/` and green. | 0.5d |
| 1. Decisions | Close D1–D6 in §4. | Written answers on the sync issue. | — |
| 2. Merge + mechanical | Branch `sync/upstream-2026-09`. `git merge upstream/main`. Resolve in order: `go.mod`/`go.sum` (regenerate), workflows (re-delete), `cmd/lists.go` (re-delete), `.goreleaser.yml`/`Makefile`/`README`/`web/index.html` (keep ours + branding), `docs/*` (take upstream). | Only the Go integration conflicts remain unresolved. | 0.5d |
| 3. Config + CLI | `config/config.go`, `config/upstreams.go`, `cmd/root.go`, `cmd/serve.go`. Regenerate enums and `docs/config.schema.json`. | `go build ./config/... ./cmd/...`, config tests green. | 1d |
| 4. Resolver chain | `resolver/blocking_resolver.go`, `metrics_resolver.go`, `query_logging_resolver.go`, plus semantic review of the cleanly-merged `caching_resolver.go`, `dnssec/validator.go`, `querylog/*`, `util/edns0.go`, `model/models.go`. Re-establish our redis and broadcaster injection against upstream's new signatures (§3.2) and our client-group attribution against upstream's new matcher (§3.3). | `go test ./resolver/... ./querylog/... ./util/...` green. | 2–3d |
| 5. Server + API | `server/server.go`, `http.go`, `server_endpoints.go`. Reconcile admin ports and the UI router with upstream's HTTP/3 and PROXY-protocol listeners. Apply D1. Regenerate `api/*.gen.go` and mocks. | `go build ./...`, `go test ./server/... ./api/...` green. Server starts without a route-registration panic. | 1–1.5d |
| 6. Full verification | `go test ./...`, e2e suite, lint at upstream's v2.12.2 ruleset, `web/ui` build. | All green. | 0.5–1d |
| 7. Behavioral smoke | Replay the Phase 0 DNS capture and diff. Manually exercise: login/session, dashboard, client groups, domain entries, blocklists, upstream groups, users, query log stream. | No unexplained delta vs Phase 0. | 0.5d |
| 8. Port checklist | Walk §6 and confirm each upstream fix is actually present and effective in the merged tree. | Checklist complete. | 0.5d |
| 9. Land | PR, review, merge. Update this document's "Last measured" line and §7. | Merged. | 0.5d |

**Estimate: 7–9 focused days.** The earlier 3–4 day estimate assumed the fork was additive; the
trial merge shows three of our integration points sit inside code upstream refactored (§3.2–3.4),
which is where the extra time goes. Phases 4 and 5 carry essentially all of the schedule risk.

## 6. Upstream port checklist

Verify each of these is present *and effective* in the merged tree — several land in files where
our version won the conflict.

**Security / correctness (must-have)**

- [ ] `2496d12` DNSSEC validation bypass & cache-scope pollution — GHSA-x845-2f78-7v36
- [ ] `a191ad2` DNSSEC: propagate Indeterminate, not Bogus, for unreachable chain of trust
- [ ] `d3a1fe5` DNSSEC: don't cache transient Indeterminate results
- [ ] `fc353a0` DNSSEC: only validate public-upstream answers
- [ ] `a42d656` DNSSEC: don't return records to clients with the DO bit clear
- [ ] `2ffe18a` RFC 4034 canonical name ordering for NSEC coverage
- [ ] `e0ea9b3` eliminate recursive RLock deadlock in blocking group resolution

**Protocol fixes**

- [ ] `78d5367` don't pass EDNS0 DNS cookies through
- [ ] `ff2aae4` always answer an EDNS0 query with an OPT record
- [ ] `2e5d478` NOTFQDN → well-formed NXDOMAIN
- [ ] `802869a` SOA record on custom-DNS NOERROR
- [ ] `c46ed64` retry DoH queries failing on a stale pooled connection
- [ ] `db8d889` compress responses larger than 512 bytes
- [ ] `190d512` count down cached authority/additional TTLs
- [ ] `1d450af` case-insensitive custom-DNS PTR matching
- [ ] `91a8f44` bootstrap: fall back to other resolved addresses on dial failure
- [ ] `e2b40db` / `dcdd952` rewritten-query handling: original name to next resolver, fallbackUpstream

**Blocking / resolver behavior**

- [ ] `344de86` scope allowlist-only mode to the whole client
- [ ] `769d908` `refused` block type
- [ ] `4b524e8` ECS `useAsClient` applied above cache and client-name lookup
- [ ] `c851293` log the matched rule in the block reason
- [ ] `99ae703` filter `ipv6hint` in HTTPS/SVCB when AAAA is filtered

**Metrics / performance**

- [ ] `7dd039c` `blocky_client_response_total` metric
- [ ] `77b0fe7` avoid per-query label map allocations in the metrics resolver
- [ ] `06555e0` bound reason label cardinality for blocked responses
- [ ] `b73422e` allocation-free resolver selection in `ParallelBestResolver`
- [ ] `316b073` pre-classify client groups
- [ ] `6da7ce1` lock-free grouped cache, cheapest-first lookup
- [ ] `87be127` sharded result cache
- [ ] `302ca65` cut per-request logger allocations
- [ ] `223df0c` skip per-request LogEntry build when query log is off

**Features (merge; enablement is D5)**

- [ ] `c32863d` DoQ upstream, `842dda9` DoH3, `1b8e08a` DoT pooling, `bee2d8b` UDP-first plain DNS
- [ ] `e6b41db` per-client rate limiting, `0b70e5c` rebinding protection, `7abca44` PROXY protocol
- [ ] `e43b5e5` SQLite query log, `e9deb53` dnstap query log, `b82199b` query-log domain ignore
- [ ] `22b0bdd` schedule-based blocking, `e7958e0` on-disk list download cache
- [ ] `c44017a` sensitive config values from files, `7c6da15` config-folder structural merge
- [ ] `fdcf351` client names from hosts file and custom DNS, `f457ec9` `resolvFile` bootstrap
- [ ] `0de3fac` `resolver.arpa` / DDR per RFC 9462
- [ ] `4cf62ce` healthcheck follows `ports.dns`

## 7. What we have intentionally forked

Keep this current — it is what makes the *next* sync cheap.

**Additive, no upstream contact (safe):** `web/ui/` (Svelte SPA), `auth/`, `configstore/`,
`logstream/`, `pkg/statscollector/`, `packaging/`, `assets/`, `VERSION`,
`Dockerfile.goreleaser`, `api/configapi/`, `cmd/user.go`, `server/server_stats.go`.

**Upstream files we carry patches in (the recurring cost):** `config/config.go`,
`config/upstreams.go`, `cmd/root.go`, `cmd/serve.go`, `server/server.go`, `server/http.go`,
`server/server_endpoints.go`, `api/api_interface_impl.go`, `resolver/blocking_resolver.go`,
`resolver/query_logging_resolver.go`, `resolver/metrics_resolver.go`, `querylog/writer.go`,
`model/models.go`, `util/edns0.go`, `web/index.html`, `Makefile`, `.goreleaser.yml`,
`.github/workflows/release.yml`.

**Deliberately deleted:** `cmd/blocking.go`, `cmd/cache.go`, `cmd/lists.go`, `cmd/query.go`
(+ tests) — replaced by the web UI. Upstream CI workflows other than `release.yml`.

**Design divergences:** upstream configuration lives in the SQLite config store, not YAML
(`upstreams:` is rejected); admin UI runs on its own listeners (`adminPort`, `adminPortTLS`);
statistics are persisted rather than in-memory.

## 8. Cadence

Seven months of drift is what turned this into a week of work. Sync every 4–6 weeks:

```bash
git fetch upstream main
git merge upstream/main
```

At that cadence each merge should be a handful of conflicts in the §7 patched-file list. After
every sync, update §7 and the "Last measured" line at the top.
