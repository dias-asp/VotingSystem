# NGVS — Next Generation Voting System

A distributed online voting system with vote anonymization and cryptographic auditing built on a blockchain-like chain of blocks.

---

## What it does

- **Registration and authentication** with a self-hosted JWT issuer (access + refresh tokens, refresh-token rotation).
- **Voting** on polls with multiple candidates. A vote can be **cancelled and changed** — the re-voting history is kept as an append-only event log (`VOTE_CAST`, `VOTE_CANCELLED`, `VOTE_CHANGED`), and the latest event wins when tallying.
- **Anonymization**: before a vote reaches the audit trail, `userId` is irreversibly replaced with an anonymous `mdmId`. The mapping table is isolated inside a dedicated service and never leaves it.
- **Audit and verification**: anonymized events are batched into blocks, a Merkle tree is built for each block, and blocks are chained by the hash of the previous one (like a blockchain). Anyone can request a **Merkle proof** for their vote and verify the integrity of the entire chain.
- **Roles**: regular user and admin. The admin creates polls, adds candidates, changes poll status, and sees the integrity status of the audit chain.
- **Web interface**: an SPA for login, voting, and administration.

---

## Architecture

Four independent backend services (each its own module, clean ports-and-adapters architecture) + a frontend. Services communicate with each other asynchronously over Kafka.

| Service | Responsibility |
|---|---|
| **auth_service** | Register / login / refresh, issuing RS256-signed JWTs, publishing JWKS |
| **voting_service** | Polls, candidates, accepting votes, the re-voting state machine, tallying |
| **anonym_service** | Replacing `userId` → `mdmId`, isolating the mapping |
| **audit_service** | Block assembly, Merkle trees, the block chain, serving proofs |
| **frontend** | React SPA (auth / voting / admin) |

Authentication happens without a round-trip to auth on every request: services fetch the public key via JWKS and **verify tokens locally**.

---

## Data flow

```text
client
  │  POST /vote  (JWT, poll_id, candidate_id)
  ▼
voting_service ──► Kafka: vote.events        (userId, candidateId, ...)
                        │
                        ▼
                  anonym_service   userId → mdmId (irreversible)
                        │
                        ▼
                   Kafka: vote.anonymized     (mdmId, candidateId, ...)
                        │
                        ▼
                   audit_service
                        │  mempool accumulates events
                        ▼
                  block builder (on a timer)
                        │  builds a Merkle tree from the events,
                        │  chains the block by the previous block's hash
                        ▼
                  block chain  ──►  GET /proof/{eventID}   (Merkle proof)
                                    POST /verify            (proof verification)
                                    GET  /chain/verify       (chain integrity)
```

Key properties:

- **Anonymity.** `voting_service` knows *who* voted but never stores that in the audit trail. `audit_service` knows *what* was counted but only ever deals with `mdmId`. The "person ↔ vote" link exists solely in the isolated mapping inside `anonym_service`.
- **Verifiability.** A leaf hash is computed from a deterministic canonical serialization of the event. `HashLeaf` is prefixed with `0x00` and `HashNode` with `0x01` (second-preimage resistance). A block's hash includes the previous block's hash — tampering with any historical vote breaks the whole chain.
- **History integrity.** Votes are never deleted. Re-voting is a new event on top of the old one, not an overwrite. That is exactly what keeps the audit trail honest.

---

## Tech stack

**Backend**
- Go — four services, mostly on the standard library (`net/http` with method routing, `log/slog`, `database/sql`)
- PostgreSQL — storage (`pgx` driver)
- Apache Kafka (KRaft, no ZooKeeper) — asynchronous event bus between services
- Hand-rolled RS256 JWT + JWKS (no external JWT libraries)
- bcrypt — password hashing
- Hand-rolled Merkle trees and block chain
- Self-migrating runner — migrations are applied on each service's startup

**Frontend**
- React + TypeScript
- Vite
- TanStack Query
- React Router
- Tailwind CSS
- nginx as a reverse proxy (`/api/*`) in production

**Infrastructure**
- Docker + Docker Compose (production images and a dev stack with hot-reload via Air / Vite)
- Multi-stage Go build: `golang:alpine` → distroless/static, non-root
- Makefile for build / test / run / compose

---

## Running

```bash
make compose-up      # bring up the whole stack (Postgres + Kafka + 4 services + frontend)
make compose-logs
make compose-down    # stop (with volume cleanup)
```

Dev mode with hot-reload:

```bash
make dev-up
make dev-down
```

The frontend is available at `http://localhost:8084` (prod) or `http://localhost:3000` (dev).

Service ports: voting `8080`, anonym `8081`, audit `8082`, auth `8083`.

---

## Implementation status

All four services and the frontend work end-to-end: registration → voting → anonymization → block assembly → issuing and verifying Merkle proofs. Each service is covered by unit tests (the voting state machine, Merkle trees, block assembly, anonymization, JWT/JWKS).

> The project is a work in progress. The up-to-date implementation status for each item is in `CLAUDE.md`. The full design document (RU) is in `anonymous_voting_audit_blockchain_project.md`.