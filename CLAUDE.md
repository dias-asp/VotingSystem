# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

The full design document lives at `anonymous_voting_audit_blockchain_project.md` (Russian). This file is the short, code-focused companion: architecture summary, layout, commands, and a status table tracking what's actually implemented vs. still on paper.

## Project Overview

Distributed online voting system in Go 1.25 (Dockerfile builders pinned to `golang:1.25-alpine`; `audit_service/go.mod` requires `1.25.0`, the others declare `1.24` and run fine on the same toolchain). Four backend services + a React SPA:

1. **auth_service** — user registration, login, refresh; issues RS256-signed JWTs (with `role` claim) and exposes JWKS for the rest.
2. **voting_service** — manages polls + candidates + per-poll votes; verifies access tokens via JWKS, gates `/admin/*` on `role=admin`.
3. **anonym_service** — replaces `userId` with an anonymous `mdmId`; keeps the mapping isolated from downstream.
4. **audit_service** — consumes anonymized events, batches them into blocks, builds Merkle trees, chains blocks by hash, serves Merkle proofs.
5. **frontend** — React + Vite + TypeScript SPA exercising auth, voting, and admin (poll/candidate management + chain status). Served by nginx in prod, Vite dev-server in dev. Talks to the three Go services through a reverse proxy (`/api/*`).

Event flow:

```text
client → voting_service → Kafka(vote.events)
                        → anonym_service → Kafka(vote.anonymized)
                        → audit_service → mempool → block + Merkle tree → proof API
```

Re-voting is modeled as an append-only event log (`VOTE_CAST`, `VOTE_CANCELLED`, `VOTE_CHANGED`); the latest event per `mdmId` wins when tallying. Old votes are never deleted — that's what keeps the audit trail intact.

## Repository Layout

Each service is its own Go module (`go.mod`). All three follow the same internal layout — clean separation between transport, business logic, and infrastructure:

```
<service>/
  cmd/<service>/main.go               # entry point: config + signal handling + wiring
  internal/
    config/                           # env-based config loader
    domain/                           # domain types (no external deps)
    service/                          # business logic + ports (interfaces it depends on)
    repository/postgres/              # postgres implementations of service ports
    transport/
      httpapi/                        # net/http server, router, handlers
      kafka/                          # Kafka consumer/producer (interfaces only — no client lib chosen yet)
  migrations/                         # 000N_<name>.up.sql / .down.sql
  Dockerfile                          # multi-stage: golang:1.24-alpine → distroless/static
  .env.example
```

**Dependency direction**: `transport/*` and `repository/*` import `service` (to satisfy interfaces declared there). `service` imports `domain`. Nothing imports `transport`. Standard ports-and-adapters layout.

Root files:
- `Makefile` — orchestrates per-service `build / run / test / tidy / vet` and `compose-up/down/logs` (+ `dev-*` for the Air dev stack).
- `infrastrucrure/docker/docker-compose.yml` — Postgres + Kafka (KRaft) + all four services. **Preserve the typo in `infrastrucrure/`** — paths depend on it.
- `infrastrucrure/docker/docker-compose.dev.yml` — same stack with Air hot-reload.
- `httprequests/<service>/<endpoint>.http` — JetBrains HTTP Client requests, one file per endpoint. Env file at `httprequests/http-client.env.json` (`dev` → `localhost`, `docker` → in-compose hostnames). `login.http` / `refresh.http` capture `access_token` + `refresh_token` into `client.global` so other requests reuse them. Add new endpoints as new `.http` files; new services as new folders.
- `scripts/migrate.sh`, `scripts/deploy.sh` — empty placeholders.
- `.gitignore` — bin/, tmp/, .env, *.pem, OS junk.

## Conventions Locked In

These were chosen when scaffolding; change deliberately, not casually.

- **stdlib-only** in the skeleton, with exceptions: every service except `auth_service` uses `github.com/jackc/pgx/v5/stdlib` (Postgres driver) — `auth_service` does too. `auth_service` adds `golang.org/x/crypto/bcrypt`; `anonym_service`, `voting_service`, and `audit_service` add `github.com/segmentio/kafka-go`. `net/http` (with Go 1.22+ method routing in `ServeMux`), `log/slog`, `database/sql`, manual env parsing.
- **Kafka client**: `segmentio/kafka-go` everywhere it's needed. Wire format is JSON with snake_case keys (`event_id`, `user_id`/`mdm_id`, `candidate_id`, `type`, `timestamp` as RFC3339); message key is the user/mdm id for ordering.
- **Postgres driver**: all four services use `pgx/v5/stdlib` (registered as `pgx` driver via `database/sql`).
- **Migrations**: every service runs `internal/database.Migrate` on startup — applies any `*.up.sql` from `MIGRATIONS_DIR` not yet recorded in a `schema_migrations` table. Plain stdlib, no external migration tool.
- **Compile-time interface checks**: every concrete adapter has `var _ service.X = (*Y)(nil)` so the compiler catches port/adapter drift.
- **Stub bodies** return `service.ErrNotImplemented` (or empty values) so the skeleton compiles and runs cleanly — HTTP servers respond, Kafka loops idle. No panics, no nil derefs from the stubs themselves. `nil` repos are passed in `main.go` until DB wiring lands; methods don't dereference them.
- **Distroless** runtime image. Static binary, non-root user.
- HTTP ports: voting=8080, anonym=8081, audit=8082, auth=8083, frontend prod=8084 (host) / 80 (container), frontend dev=3000.
- **Auth model**: own JWT issuer (RS256). `auth_service` mints access (15m default) + refresh (30d default) tokens; other services fetch the public key from `GET /.well-known/jwks.json` and verify locally — no per-request hop to auth.
- **Roles**: JWT carries a `role` claim (`user` / `admin`). `voting_service/internal/auth.RequireRole` is wrapped around `/admin/*` endpoints. Admin is granted on registration only when the email matches `ADMIN_EMAIL` env var on `auth_service`. There is no manual promote/demote endpoint.
- **Password hashing**: bcrypt cost 12 via `golang.org/x/crypto/bcrypt` (real implementation in `auth_service/internal/passwords`).
- **JWT**: hand-rolled RS256 in stdlib (`crypto/rsa` + `crypto/sha256` + `encoding/base64`), no external JWT lib. Issuer signs and exposes JWKS; Verifier validates signature, issuer claim, and expiry. Private key auto-generated to `JWT_PRIVATE_KEY_PATH` on first start if missing — see `tokens.LoadOrGeneratePrivateKey`.

## Commands

```bash
make build               # build all three services
make build-voting_service
make run-voting_service  # go run ./cmd/voting_service
make test                # go test ./... per service
make tidy                # go mod tidy per service
make vet
make compose-up          # docker compose up -d --build (prod-ish images)
make compose-down        # down -v (drops volumes)
make compose-logs
make dev-up              # docker compose up -d --build with Air hot-reload (dev)
make dev-down            # dev down -v
make dev-logs
make frontend-install    # npm install inside frontend/
make frontend-dev        # vite dev server on :3000 (uses VITE_*_PROXY env vars)
make frontend-build      # production bundle in frontend/dist
```

## Implementation Status

Update this table as work lands. When a feature lands, flip the box and (briefly) note the entrypoint file/package so future work has a starting point.

### Infrastructure
- [x] Per-service skeleton (cmd, internal/{config,domain,service,repository,transport}, migrations, Dockerfile)
- [x] Multi-stage Dockerfile (golang:1.25-alpine → distroless/static, non-root)
- [x] `docker-compose.yml` — Postgres 16, Apache Kafka 3.7.2 (`apache/kafka` official image, KRaft, no ZooKeeper), all four Go services + `frontend` (nginx serving Vite-built static, `/api/*` proxied to backends). Frontend exposed on host `:8084`.
- [x] `docker-compose.dev.yml` — same stack with Air hot-reload for Go services + `frontend` running `vite --host` (port `:3000`, source mounted, named volume for `node_modules`).
- [x] `ADMIN_EMAIL` env var threaded into `auth_service` from compose (defaults to `admin@example.com`)
- [x] Makefile — build / run / test / tidy / vet / compose / dev-* targets + `frontend-install`, `frontend-dev`, `frontend-build`
- [x] `.env.example` per service (incl. `frontend/.env.example`)
- [x] `.gitignore`
- [ ] `scripts/migrate.sh` — pick a migration tool (golang-migrate / goose / atlas) and wire it. All four services already self-migrate on startup via `internal/database.Migrate`; this is only needed for ops scenarios where startup migration isn't viable.
- [ ] `scripts/deploy.sh`
- [ ] Shared module / package for cross-service event types — currently each service redeclares them in its own `domain/`. Refactor when a real Kafka client lands.

### auth_service
- [x] HTTP server scaffold (graceful shutdown)
- [x] Routes: `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /.well-known/jwks.json`, `GET /healthz`
- [x] `service.Auth` with ports (`UserRepository`, `RefreshTokenRepository`, `TokenIssuer`, `TokenVerifier`, `PasswordHasher`)
- [x] Postgres repositories (`user.go`, `refresh.go`) — real queries via `database/sql` + pgx
- [x] `tokens.Issuer` / `tokens.Verifier` — real RS256 (hand-rolled) + JWKS marshaling, includes `role` in JWT payload
- [x] `passwords.BcryptHasher` — real bcrypt (cost 12)
- [x] Migration `0001_init` — `users`, `refresh_tokens` (citext for emails); applied by self-migrating runner on startup
- [x] Migration `0002_add_role` — `users.role TEXT NOT NULL DEFAULT 'user'`
- [x] Role assignment on register: when the email matches `ADMIN_EMAIL` env var (lowercased trim), role = `admin`; otherwise `user`. No promote/demote API.
- [x] Real handler bodies (JSON in/out, error → status code mapping); register response includes `role`
- [x] Real `Auth.Register/Login/Refresh/Logout` (refresh-token rotation, revoke-on-refresh)
- [x] Key generation helper — `tokens.LoadOrGeneratePrivateKey` auto-creates an RSA-2048 key at `JWT_PRIVATE_KEY_PATH` if missing
- [x] Tests — `passwords` (round-trip), `tokens` (issue/verify/JWKS with role, expiry, wrong-issuer)

### voting_service
- [x] HTTP server scaffold (stdlib `net/http`, graceful shutdown); opens DB and self-migrates on startup
- [x] JWT middleware wired (`internal/auth.Middleware`) — pulls JWKS from auth_service, verifies access tokens, puts `Subject` + `Role` in request context. `RequireRole(domain.RoleAdmin)` gates `/admin/*` routes.
- [x] User-facing routes: `GET /polls`, `GET /polls/{id}`, `GET /polls/{id}/results`, `GET /polls/{id}/me`, `POST /vote`, `POST /revote`, `POST /cancel-vote`. `user_id` is taken from JWT context, never from the request body. Body shape: `{ "poll_id": "...", "candidate_id": "..." }` (cancel takes only `poll_id`).
- [x] Admin routes (require `role=admin`): `POST /admin/polls`, `POST /admin/polls/{id}/candidates`, `PATCH /admin/polls/{id}/status`.
- [x] `service.Voting` + `service.Polls` with ports (`VoteRepository`, `PollRepository`); voting state machine `(none) → CAST → CANCELLED → CHANGED → CANCELLED → ...` enforced via `ErrAlreadyVoted` / `ErrNoActiveVote` / `ErrCannotChange`. Voting also requires the poll to be `active` (`ErrPollNotActive`).
- [x] Postgres repositories (`vote.go`, `poll.go`) — append-only `INSERT` into `votes`; `LatestByUser` and `Tally` are scoped by `poll_id`; candidates have a composite `(poll_id, id)` PK with FK from votes.
- [x] Real Kafka producer (`segmentio/kafka-go` `Writer`, `Hash` balancer, `RequireAll` acks, snake_case JSON with `poll_id`; key = `user_id`)
- [x] Candidate validation via `VoteRepository.CandidateExists(poll_id, candidate_id)` before any state-changing op
- [x] Self-migration on startup via shared `internal/database.Migrate`
- [x] Migration `0001_init` — `candidates`, `votes`
- [x] Migration `0002_seed_candidates` — sample candidates `candA` / `candB` / `candC`
- [x] Migration `0003_outbox` — transactional outbox for Kafka publish
- [x] Migration `0004_polls` — adds `polls` table, `poll_id` columns on `candidates` + `votes`, composite PK + FK, default poll `00000000-0000-0000-0000-000000000001` (active) for seeded data
- [x] Tests — voting service (cast happy path, unknown candidate, double-cast, cancel-without-active, full cast→cancel→change flow with tally, change-without-cancel, publish failure surfaces) — all run with poll-scoped fakes

### anonym_service
- [x] Service entry with HTTP healthz + Kafka consumer goroutine + graceful shutdown; opens DB and self-migrates on startup
- [x] `service.Anonymizer` with ports (`MappingRepository`, `EventPublisher`); validates incoming events
- [x] Postgres repository (`repository/postgres/mapping.go`) — `GetOrCreate` is one atomic upsert (`INSERT ... ON CONFLICT (user_id) DO UPDATE ... RETURNING mdm_id`)
- [x] Real Kafka consumer (`segmentio/kafka-go` Reader, consumer group, `FetchMessage` + commit-on-success; commit-and-skip on decode error so a poison message can't stall the group)
- [x] Real Kafka producer (`kafka.Writer` with `Hash` balancer, `RequireAll` acks, snake_case JSON; key = `mdm_id`)
- [x] Migration `0001_init` — `user_mdm_mapping`
- [x] `mdmId` strategy: random UUID v4 generated in the repo, written under `ON CONFLICT DO UPDATE` so concurrent inserts converge on a single value
- [x] Self-migration on startup via shared `internal/database.Migrate` (mirrors `auth_service`)
- [x] Tests — anonymizer (publish path, mapping reuse, invalid-event guard, repo-error propagation)

### audit_service
- [x] Service entry with HTTP API + Kafka consumer + block-builder ticker + graceful shutdown; opens DB and self-migrates on startup
- [x] Domain: `AuditRecord`, `Block`, `MerkleProof` / `MerkleProofStep`
- [x] `service.Auditor`, `BlockBuilder`, `Prover`, `ChainVerifier` with ports
- [x] Postgres repositories (`blocks.go`, `records.go`) — real queries via `database/sql` + pgx; `AssignBlock` runs in a transaction with a per-id `UPDATE ... WHERE block_id IS NULL` guard
- [x] Real Kafka consumer (`segmentio/kafka-go` Reader, consumer group, commit-on-success; commit-and-skip on decode error)
- [x] HTTP routes: `GET /healthz`, `GET /proof/{eventID}`, `POST /verify`, `GET /chain/verify` — real bodies (JSON in/out with hex-encoded hashes)
- [x] Merkle helpers — real implementations of `MerkleRoot` / `BuildProof` / `VerifyProof` (duplicate-last-leaf for odd levels; `HashLeaf` 0x00-prefixed, `HashNode` 0x01-prefixed for second-preimage resistance)
- [x] Real `Auditor.Handle` — validates event, computes leaf hash from `CanonicalRecordBytes` (deterministic `event_id|mdm_id|candidate_id|type|timestamp` layout), appends to mempool
- [x] Real `BlockBuilder.buildOnce` — pulls pending records, builds Merkle tree, chains prev-hash, persists block, assigns records to it. Block hash = SHA-256(0x02 || prev_hash || merkle_root || created_at_unix_nanos); `created_at` truncated to microsecond before insert so the round-tripped TIMESTAMPTZ recomputes the same hash
- [x] Real `Prover.Proof` — fetches block + sibling records, rebuilds proof from the leaf hashes
- [x] Chain integrity endpoint — `service.ChainVerifier.Verify` walks blocks in id order, recomputes each hash, asserts `prev_hash` chain
- [x] Self-migration on startup via shared `internal/database.Migrate`
- [x] Migration `0001_init` — `blocks`, `records`, `merkle_proofs`
- [x] Tests — merkle (root/proof round-trip across odd/even sizes, tampered-leaf and tampered-sibling rejection), auditor (hash computed, validation, repo error propagation), block builder (no-pending no-op, persists + assigns, chains prev_hash across blocks), prover (round-trip + pending-record not-found)

### frontend
React 18 + Vite 5 + TypeScript + TanStack Query + React Router 6 + Tailwind. Layout:

```
frontend/
  src/
    main.tsx             # entry, BrowserRouter + QueryClientProvider
    App.tsx              # route table; auth/admin guards
    api/
      client.ts          # fetch wrapper, ApiError, JWT decode helpers, token storage in localStorage
      endpoints.ts       # auth / polls / admin / audit endpoint helpers (all paths under `/api/*`)
      types.ts           # DTOs mirroring backend JSON shapes
    components/
      Layout.tsx         # header + nav (Polls / Admin / logout)
      Guards.tsx         # RequireAuth, RequireAdmin
    pages/
      LoginPage.tsx
      RegisterPage.tsx
      PollsListPage.tsx       # lists polls, status badges
      PollDetailsPage.tsx     # vote / cancel / switch (cancel → revote sequence), live results polled every 5 s
      AdminPage.tsx           # create poll, add candidates, change status, audit chain status (polled every 10 s)
  index.html, index.css (Tailwind), tsconfig.json, vite.config.ts, tailwind.config.js, postcss.config.js
  Dockerfile             # multi-stage: node:20-alpine build → nginx:1.27-alpine, copies dist + nginx.conf
  Dockerfile.dev         # node:20-alpine, runs `npm run dev`
  nginx.conf             # SPA fallback + reverse proxy: /api/auth → auth_service, /api/audit → audit_service, /api/* → voting_service
  .env.example           # VITE_*_PROXY for local `npm run dev` outside docker
```

- [x] Auth flow: login/register, tokens cached in `localStorage` (access + refresh + role + email). JWT `role` claim is read client-side to switch UI.
- [x] User flow: list polls → open poll → vote / cancel / switch (switch = cancel + revote chained via mutation). Live tally via `useQuery` + `refetchInterval`.
- [x] Admin flow: create poll, add candidates, flip status (draft / active / closed), live audit chain status from `audit_service /chain/verify`.
- [x] Reverse proxy via nginx in prod and Vite proxy in dev — frontend never talks to backends by hardcoded host.
- [ ] Refresh-token rotation on 401 — currently a 401 just clears storage and bounces to /login. Wire the `/auth/refresh` flow into the fetch wrapper if sessions need to outlive the access TTL without re-login.
- [ ] Frontend tests — none yet (component/integration). Vitest is not wired.

### Cross-cutting
- [x] JWT auth middleware in `voting_service` — fetches JWKS from `auth_service`, verifies access tokens locally; gates `/admin/*` on `role=admin`
- [ ] JWT refresh on the frontend — see frontend section
- [ ] Swagger / OpenAPI specs per service
- [ ] Real health checks (DB ping, Kafka connectivity) — currently `/healthz` returns 200 unconditionally
- [ ] Structured request logging middleware

## Working Style Notes

- The design doc is the source of truth for *intent*; this file is the source of truth for *current code state*. If they conflict about what exists, trust the code.
- Preserve the `infrastrucrure/` typo. Renaming is a separate, deliberate change with `docker-compose` path updates.
- Folder is `anonym_service`, not `anonymization_service` as the design doc names it.
