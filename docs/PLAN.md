# PLAN.md — Gin Boilerplate

> A production-grade Gin + PostgreSQL + GORM boilerplate built for learning.
> Built on [Hexagonal Architecture](ARCHITECTURE.md) — every feature is split into an
> inner package (pure domain) plus outer `http/` and `adapter/` packages.

---

## 🎯 Goal

Build a **production-grade** Gin API foundation where:
- Every feature has a **clear learning purpose**.
- **Automated tests** exist from day one (unit + integration + E2E).
- The **hexagon stays intact** — the inner feature package has zero infrastructure imports,
  and that is enforced by a linter rather than by good intentions.
- Deploy is a human gate — CI is automated, but merge/deploy needs approval.

---

## 📐 Principles

1. **Loop, not a straight line** — each phase ends with automated tests + manual review.
2. **Test-first on critical layers** — auth, error handling, and repository must have tests before "done".
3. **Ports before adapters** — write the port interface and its fake first; the GORM/Redis/S3 implementation comes second.
4. **Domain stays pure** — the inner package `internal/<feature>` must never import `gin`,
   `gorm.io/gorm`, or `net/http`. This is enforced by a `depguard` lint rule, not by memory.
5. **Only declare what you use** — no port method without a caller; no endpoint without a need.
6. **Deploy = human gate** — CI automated, but merge/deploy requires approval.
7. **Don't bloat early** — "nice to have" features go last and can be skipped.

---

## 🧭 Decisions (locked)

These were decided up front so the phases below don't re-litigate them.

| Decision | Choice | Rationale |
|---|---|---|
| Architecture | Hexagonal (Ports & Adapters), feature-based packages | Business logic testable without infra; swappable adapters |
| Package layout | `internal/<feature>/` (inner) + `internal/<feature>/http/` + `internal/<feature>/adapter/` | Go linters enforce per **package** — the domain must be its own package to be lintable |
| Port shape | Split by capability (`UserReader` / `UserWriter`), only methods actually called | Keeps fakes small; avoids "god interface" |
| Errors | Inner sentinel → per-feature `http/errors.go` mapping → `pkg/errors.AppError` → `pkg/response` | Domain never speaks HTTP; no global registry to bottleneck on |
| Roles | Server-side only — the register DTO has **no** `Role` field | Prevents privilege escalation via self-registration |
| IDs | **UUID v7**, generated in the service via `pkg/id`; adapter holds `uuid.UUID` and the mapper parses | Time-ordered (dense indexes), no DB extension, known before INSERT |
| Logging | `*zap.Logger` used directly in the inner package — **no** `Logger` port | Zap is a pure utility; a per-feature logger interface buys nothing |
| Migrations | Raw SQL via `golang-migrate`; AutoMigrate only for local iteration | Versioned, reviewable, reversible |
| Tests | Fakes for ports, Testcontainers for adapters | Hexagon's seam is the port; that's where fakes go |
| Swagger | `swag` v1 (pinned) with `@Router` annotations; output to `swagger/`, not `docs/` | Keeps `docs/` markdown-only; avoids a Go package hiding in a docs folder |
| Wiring | `cmd/api/main.go` only | No DI framework, no magic |

---

## 🗺️ Phases

### Phase 0 — Project Foundation

- [ ] `go.mod` + module path; `cmd/api`, `cmd/seed` entry points
- [ ] Package skeleton: `internal/<feature>/{domain,dto,port,errors,service}`, `internal/<feature>/http/`, `internal/<feature>/adapter/`, `internal/router/`, `pkg/`
- [ ] Config Service (`internal/config` — env-driven, validated at startup, fails fast)
- [ ] Structured Logging (`pkg/logger` — Zap, JSON output, includes `request_id`)
- [ ] `pkg/id` — UUID v7 helper (`id.NewV7()`)
- [ ] `pkg/errors` — `AppError` type with a **real** `cause` field and `Unwrap()`; constructors take an optional cause
- [ ] `pkg/response` — uniform JSON success/error envelopes; `ErrorBody`/`ErrorEnvelope` types defined here
- [ ] Graceful Shutdown (`signal.NotifyContext` + `http.Server.Shutdown`)
- [ ] Health Check endpoint (`GET /health` — checks DB + Redis, returns 200/503)
- [ ] Middleware: Request ID, **AccessLog** (not `Logger` — the name collides with the service's logger field), Recovery, CORS whitelist (no wildcard)
- [ ] Makefile (`make run`, `make test`, `make test-integration`, `make lint`, `make seed`, `make swagger`)
- [ ] Docker Compose (app + PostgreSQL + Redis)
- [ ] `golangci-lint` config incl. the **depguard** import-boundary rule from `ARCHITECTURE.md`

**Done when:** `docker compose up` brings the app up, `/health` returns 200, `make test` passes, and the linter **rejects** a `gin` import in `internal/user/service.go`.

**Automated tests:** smoke test `/health` via `httptest`; config validation unit tests; error-envelope unit tests; a test asserting `pkg/errors` wraps and unwraps a cause correctly.

---

### Phase 1 — Database, Migrations & Seeding

- [ ] PostgreSQL + GORM connection (`pkg/database` — pool sizing, ping, close)
- [ ] Versioned migrations in `migrations/` (`golang-migrate`, up + down files)
- [ ] `internal/user` inner package: `domain.go`, `dto.go`, `port.go`, `errors.go`, `service.go`
- [ ] `internal/user/adapter/record.go` (GORM model, `uuid.UUID` PK)
- [ ] `internal/user/adapter/mapper.go` — `toDomain` / `toRecord`, both **returning an error** (UUID parsing)
- [ ] `internal/user/adapter/repository_gorm.go` — implements `user.UserReader` / `user.UserWriter`, returns `user.ErrNotFound`
- [ ] `internal/user/http/handler.go` + `internal/user/http/errors.go` — `toAppError` mapping
- [ ] `internal/router` — first route registered
- [ ] Seeding (`cmd/seed` — default admin with `RoleAdmin`; **idempotent**, safe to re-run)
- [ ] `testutil/` — Testcontainers PostgreSQL helper + migration runner

**Done when:** `make migrate-up` applies cleanly, `make seed` is idempotent, the repository round-trips a user, and `make lint` confirms `internal/user` imports no `gin`.

**Automated tests:** integration test with Testcontainers PostgreSQL — verify migrations applied, seeding is idempotent, and `FindByEmail` returns `ErrNotFound` for a missing user; table-driven test for `toAppError` covering every sentinel.

---

### Phase 2 — Auth & User ⭐ *learning priority*

- [ ] `POST /auth/register` — bcrypt hashing, email uniqueness via `user.ErrEmailTaken`; **role is server-side** (`RoleUser`)
- [ ] `POST /auth/login` — JWT access + refresh tokens
- [ ] `internal/auth` inner package: `domain`, `dto`, `port` (`UserFinder`, `TokenIssuer`, `TokenStore`, `Mailer`), `errors`, `service`
- [ ] `internal/auth/http/handler.go` + `internal/auth/http/errors.go`
- [ ] `internal/auth/adapter/token_jwt.go` — HS256 issuer/verifier, separates access vs refresh claims
- [ ] `internal/auth/adapter/token_redis.go` — refresh-token store with TTL + revocation
- [ ] `internal/auth/adapter/mailer_smtp.go` — `Mailer` port, SMTP implementation
- [ ] Auth middleware — validates JWT, rejects invalid/expired tokens with 401
- [ ] RBAC middleware — `RequireRole(user.RoleAdmin)`; regular user gets 403
- [ ] Admin-only user listing (`GET /users`) — the first caller of a `UserReader.FindByRole`-style port, added **only** because an endpoint needs it
- [ ] Request validation with user-friendly messages (custom validator)
- [ ] CORS whitelist middleware (no wildcard)

**Done when:** full auth flow works end-to-end and is covered by tests; the linter confirms `internal/auth` and `internal/user` import no `gin`.

**Automated tests:**
- Unit test `auth` service with a fake `UserFinder` + fake `TokenIssuer` + fake `Mailer` — no DB, no network.
- Integration test: register → login → access protected endpoint → access admin endpoint (must 403).
- Table-driven test that every `auth` sentinel maps to the intended status code.
- **Negative lint test:** a fixture service file importing `gin` must fail `make lint`.

---

### Phase 3 — File Upload

- [ ] `internal/file` inner package: `domain`, `dto`, `port` (`Storage`), `errors`, `service`
- [ ] `internal/file/http/handler.go` + `internal/file/http/errors.go`
- [ ] `internal/file/adapter/record.go` + `adapter/mapper.go` + `adapter/repository_gorm.go` — file metadata persistence
- [ ] `internal/file/adapter/storage_local.go` — local disk driver
- [ ] `internal/file/adapter/storage_s3.go` — S3 driver (MinIO in tests)
- [ ] `POST /files` — multipart upload + validation (content type, size limit)

**Done when:** uploads work for both drivers; invalid files rejected with a clear message; oversized uploads rejected before the body is fully buffered.

**Automated tests:** integration test upload to local + MinIO (Testcontainers); unit test the service with a fake `Storage`; table-driven test for `toAppError`.

---

### Phase 4 — Redis

- [ ] `pkg/cache` — Redis client wrapper (get/set with TTL, ping, close)
- [ ] Redis-backed rate limiting middleware (`RateLimitMiddleware`, excess → 429)
- [ ] Graceful degradation — Redis down → API still works (fall back to DB / skip cache)
- [ ] Cache-aside helper with an explicit contract: key format, TTL, invalidation trigger, and **cache failures never fail the request**

**Done when:** API survives a Redis outage without breaking core flows.

**Automated tests:** integration test with Testcontainers Redis — rate limit trips at the threshold; requests still succeed after Redis is stopped.

---

### Phase 5 — Documentation & Swagger

- [ ] Swagger annotations in handlers (`@Summary`, `@Tags`, `@Router`, `@Param`, `@Success`, `@Failure`)
- [ ] General API info block in `cmd/api/main.go`
- [ ] `swag init -o swagger` → `swagger/docs.go`, `swagger/swagger.json`, `swagger/swagger.yaml` (**not** `docs/`, which stays markdown-only)
- [ ] `/swagger/*any` endpoint serving Swagger UI
- [ ] Every endpoint documented; errors use the shared `ErrorEnvelope` schema
- [ ] `swag` version pinned (`go run github.com/swaggo/swag/cmd/swag@v1.16.6`) so the drift check is deterministic

**Done when:** `swagger.json` generates without errors and every route in `internal/router` appears in it.

**Automated tests:** CI step regenerates the spec with the **pinned** version and fails if the output differs from the committed one (drift detection).

---

### Phase 6 — Automation, Test & CI ⭐ *focus*

- [ ] Separate unit vs integration (`//go:build integration`)
- [ ] Testcontainers helpers for PostgreSQL, Redis, MinIO (`testutil/`)
- [ ] E2E test for the full flow (`e2e_test.go`: register → login → upload → access admin)
- [ ] Coverage report (`coverage.out`) — thresholds **start** at 40% overall / 60% for services and are raised in the same PR that raises them
- [ ] GitHub Actions: lint → unit test → integration test → build → coverage
- [ ] Build & push Docker image (multi-stage, non-root user, distroless base)
- [ ] Pre-commit hook (`gofmt`, `go vet`, `golangci-lint`)

**Done when:** PR fails if lint/tests fail; coverage visible on PR; image builds.

---

### Phase 7 — Nice to Have (optional)

- [ ] `make new-feature NAME=order` scaffolding target (templates for the inner package, `http/`, and `adapter/`)
- [ ] Circuit breaker for external dependencies
- [ ] Light load test (`k6` / `hey`) — performance baseline
- [ ] Tracing (OpenTelemetry) wired through `pkg/logger`
- [ ] `docs/API.md` — endpoint reference generated from the Swagger spec

---

## 🚦 Quality Gates (every phase)

1. **All tests green** before moving to the next phase.
2. **Import-boundary check passes** — the linter confirms `internal/<feature>` imports no
   `gin` / `gorm.io/gorm` / `net/http`, and `internal/<feature>/http` imports no adapter.
3. **Coverage** — starts at 40% overall / 60% for the service layer; raised gradually,
   never lowered to make CI green.
4. **Every new sentinel error has a `toAppError` mapping and a table-driven test.**
5. **Manual review** for DB schema & auth changes.
6. **Deploy = manual approval** — CI automated, but merge/deploy needs you.

---

## 📅 Estimate

**± 10–14 working days** (depending on learning pace).

To go faster:
- Skip Phase 7.
- Simplify Phase 4 (Redis as cache only, no rate limiting).

---

## 🧪 Testing Strategy

| Layer | Type | Tool | Infra |
|---|---|---|---|
| Domain / Service | Unit | `testing` + hand-written fakes | none |
| Mapper | Unit (pure functions) | none |
| Handler | Unit | `httptest` + fake service | none |
| Error mapping | Unit (table-driven) | `testing` | none |
| Repository / Adapter | Integration | Testcontainers PostgreSQL | Docker |
| Full flow | E2E | `httptest` + Testcontainers | Docker |
| CI | Automated | GitHub Actions | Docker |

**Rules:**

- Unit tests: fast, no external deps — they must pass in under a second per package.
- Integration tests: behind `//go:build integration`, run via `make test-integration` (`go test -tags=integration ./...`).
- E2E tests: cover at least one full user journey per major feature.
- Fakes are defined in `_test.go` files in the **inner** package, next to the port they satisfy,
  so production code cannot import them.
- Prefer fakes over generated mocks: a fake implements the port interface and is easier to read than a mock's call assertions.
- Every feature's `toAppError` gets a table-driven test asserting each sentinel → status code.
- Maps and slices in fakes are always initialized explicitly — never left `nil`.

---

## 🛠️ Tech Stack

| Concern | Choice |
|---|---|
| HTTP framework | Gin |
| Database | PostgreSQL + GORM |
| Migrations | `golang-migrate` (raw SQL) |
| Cache / rate limit | Redis (`go-redis`) |
| Config | env-driven (`internal/config`) with startup validation |
| Logging | Zap + `gin-contrib/zap` (`*zap.Logger` used directly in services) |
| IDs | `google/uuid` (UUID v7) |
| Auth | JWT (HS256) + bcrypt |
| Mail | `Mailer` port — SMTP implementation + fake in tests |
| Storage | Local + S3 (MinIO in tests) |
| Docs | Swagger (`swag`, pinned version; output in `swagger/`) |
| Testing | `testing`, `httptest`, Testcontainers |
| Lint | `golangci-lint` (incl. `depguard` import-boundary rules) |
| CI | GitHub Actions |
| Container | Docker + Docker Compose |

---

## 📌 Notes

- This file is a living document — update checkboxes as you go.
- Each phase should end with a **reviewable PR**.
- Keep PRs small: one feature per PR where possible.
- `docs/ARCHITECTURE.md` is the source of truth for structure; this file is the source of truth for order.

---

Previous: [ARCHITECTURE.md](ARCHITECTURE.md) · Next: [AGENT.md](../AGENT.md)
