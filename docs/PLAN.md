# PLAN.md — Gin Boilerplate

> A production-grade Gin + PostgreSQL + GORM boilerplate built for learning.
> Built on [Hexagonal Architecture](ARCHITECTURE.md) — every feature is a package with
> domain / dto / port / service / handler / adapter layers.

---

## 🎯 Goal

Build a **production-grade** Gin API foundation where:
- Every feature has a **clear learning purpose**.
- **Automated tests** exist from day one (unit + integration + E2E).
- The **hexagon stays intact** — the domain layer has zero infrastructure imports.
- Deploy is a human gate — CI is automated, but merge/deploy needs approval.

---

## 📐 Principles

1. **Loop, not a straight line** — each phase ends with automated tests + manual review.
2. **Test-first on critical layers** — auth, error handling, and repository must have tests before "done".
3. **Ports before adapters** — write the port interface and its fake first; the GORM/Redis/S3 implementation comes second.
4. **Domain stays pure** — `internal/<feature>` (domain/dto/port/service) must never import `gin`, `gorm.io/gorm`, or `net/http`. This is enforced by a lint check, not by memory.
5. **Deploy = human gate** — CI automated, but merge/deploy requires approval.
6. **Don't bloat early** — "nice to have" features go last and can be skipped.

---

## 🧭 Decisions (locked)

These were decided up front so the phases below don't re-litigate them.

| Decision | Choice | Rationale |
|---|---|---|
| Architecture | Hexagonal (Ports & Adapters), feature-based packages | Business logic testable without infra; swappable adapters |
| Layout | `internal/<feature>/{domain,dto,port,errors,service,handler}` + `internal/<feature>/adapter/` | One feature = one package; adapters quarantined |
| Routing | `internal/router/` | All URL paths in one greppable place |
| IDs | **UUID v7**, generated in the service via `pkg/id` | Time-ordered (dense indexes), no DB extension, known before INSERT |
| Errors | Feature sentinel errors → `pkg/errors.AppError` → `pkg/response` | Domain never speaks HTTP; one central status mapping |
| Migrations | Raw SQL via `golang-migrate`; AutoMigrate only for local iteration | Versioned, reviewable, reversible |
| Tests | Fakes for ports, Testcontainers for adapters | Hexagon's seam is the port; that's where fakes go |
| Swagger | `swag` v1 with `@Router` annotations on handlers | Annotations live next to the code they describe |

---

## 🗺️ Phases

### Phase 0 — Project Foundation

- [ ] `go.mod` + module path; `cmd/api`, `cmd/seed` entry points
- [ ] Feature-based skeleton: `internal/<feature>/` layout, `internal/router/`, `pkg/`
- [ ] Config Service (`internal/config` — env-driven, validated at startup, fails fast)
- [ ] Structured Logging (`pkg/logger` — Zap, JSON output, includes `request_id`)
- [ ] `pkg/id` — UUID v7 helper
- [ ] `pkg/errors` — `AppError` type + constructor helpers
- [ ] `pkg/response` — uniform JSON success/error envelopes
- [ ] Graceful Shutdown (`signal.NotifyContext` + `http.Server.Shutdown`)
- [ ] Health Check endpoint (`GET /health` — checks DB + Redis, returns 200/503)
- [ ] Middleware: Request ID, structured access log, Recovery, CORS whitelist (no wildcard)
- [ ] Makefile (`make run`, `make test`, `make test-integration`, `make lint`, `make seed`, `make swagger`)
- [ ] Docker Compose (app + PostgreSQL + Redis)
- [ ] `golangci-lint` config incl. an **import-boundary** check for the domain purity rule

**Done when:** `docker compose up` brings the app up, `/health` returns 200, `make test` passes, and the linter rejects a `gin` import in a service file.

**Automated tests:** smoke test `/health` via `httptest`; config validation unit tests; error-envelope unit tests.

---

### Phase 1 — Database, Migrations & Seeding

- [ ] PostgreSQL + GORM connection (`pkg/database` — pool sizing, ping, close)
- [ ] Versioned migrations in `migrations/` (`golang-migrate`, up + down files)
- [ ] `internal/user` feature skeleton: domain, dto, port, errors, service, handler, `adapter/`
- [ ] `adapter/record.go` (GORM model, `uuid` PK), `adapter/mapper.go` (`toDomain` / `toRecord`)
- [ ] `adapter/repository_gorm.go` — implements `user.Repository`, returns `user.ErrNotFound`
- [ ] Seeding (`cmd/seed` — default admin; **idempotent**, safe to re-run)
- [ ] `testutil/` — Testcontainers PostgreSQL helper + migration runner

**Done when:** `make migrate-up` applies cleanly, `make seed` is idempotent, and the repository round-trips a user.

**Automated tests:** integration test with Testcontainers PostgreSQL — verify migrations applied, seeding is idempotent, and `FindByEmail` returns `ErrNotFound` for a missing user.

---

### Phase 2 — Auth & User ⭐ *learning priority*

- [ ] `POST /auth/register` — bcrypt hashing, email uniqueness via `user.ErrEmailTaken`
- [ ] `POST /auth/login` — JWT access + refresh tokens
- [ ] `internal/auth` feature: `domain`, `dto`, `port` (`UserFinder`, `TokenIssuer`, `TokenStore`), `errors`, `service`, `handler`
- [ ] `adapter/token_jwt.go` — HS256 issuer/verifier, separates access vs refresh claims
- [ ] `adapter/token_redis.go` — refresh-token store with TTL + revocation
- [ ] `adapter/mailer_smtp.go` — `Mailer` port, SMTP implementation
- [ ] Auth middleware — validates JWT, rejects invalid/expired tokens with 401
- [ ] RBAC middleware — `RequireRole(user.RoleAdmin)`; regular user gets 403
- [ ] Central sentinel-error → `AppError` → HTTP status registry
- [ ] Request validation with user-friendly messages (custom validator)
- [ ] CORS whitelist middleware (no wildcard)

**Done when:** full auth flow works end-to-end and is covered by tests; domain layer imports no `gin`.

**Automated tests:**
- Unit test `auth` service with a fake `UserFinder` + fake `TokenIssuer` + fake `Mailer` — no DB, no network.
- Integration test: register → login → access protected endpoint → access admin endpoint (must 403).
- Test that the linter/import check fails if a service imports `gin`.

---

### Phase 3 — File Upload

- [ ] `internal/file` feature with a `Storage` port (abstraction for local & S3)
- [ ] `adapter/storage_local.go` — local disk driver
- [ ] `adapter/storage_s3.go` — S3 driver (MinIO in tests)
- [ ] `POST /files` — multipart upload + validation (content type, size limit)
- [ ] File metadata persisted via the `file` GORM adapter

**Done when:** uploads work for both drivers; invalid files rejected with a clear message; oversized uploads rejected before the body is fully buffered.

**Automated tests:** integration test upload to local + MinIO (Testcontainers); unit test the service with a fake `Storage`.

---

### Phase 4 — Redis

- [ ] `pkg/cache` — Redis client wrapper (get/set with TTL, ping, close)
- [ ] Redis-backed rate limiting middleware (`RateLimitMiddleware`, excess → 429)
- [ ] Graceful degradation — Redis down → API still works (fall back to DB / skip cache)
- [ ] Cache-aside helper for read-heavy domain queries

**Done when:** API survives a Redis outage without breaking core flows.

**Automated tests:** integration test with Testcontainers Redis — rate limit trips at the threshold; requests still succeed after Redis is stopped.

---

### Phase 5 — Documentation & Swagger

- [ ] Swagger annotations in handlers (`@Summary`, `@Tags`, `@Router`, `@Param`, `@Success`, `@Failure`)
- [ ] General API info block in `cmd/api/main.go`
- [ ] `swag init` → `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`
- [ ] `/swagger/*any` endpoint serving Swagger UI
- [ ] Every endpoint documented; errors use the shared `ErrorResponse` schema

**Done when:** `swagger.json` generates without errors and every route in `internal/router` appears in it.

**Automated tests:** CI step regenerates the spec and fails if the output differs from the committed one (drift detection).

---

### Phase 6 — Automation, Test & CI ⭐ *focus*

- [ ] Separate unit vs integration (`//go:build integration`)
- [ ] Testcontainers helpers for PostgreSQL, Redis, MinIO (`testutil/`)
- [ ] E2E test for the full flow (`e2e_test.go`: register → login → upload → access admin)
- [ ] Coverage report (`coverage.out`, thresholds enforced in CI)
- [ ] GitHub Actions: lint → unit test → integration test → build → coverage
- [ ] Build & push Docker image (multi-stage, non-root user, distroless base)
- [ ] Pre-commit hook (`gofmt`, `go vet`, `golangci-lint`)

**Done when:** PR fails if lint/tests fail; coverage visible on PR; image builds.

---

### Phase 7 — Nice to Have (optional)

- [ ] `make new-feature NAME=order` scaffolding target (templates for domain/dto/port/service/handler/adapter)
- [ ] Circuit breaker for external dependencies
- [ ] Light load test (`k6` / `hey`) — performance baseline
- [ ] Tracing (OpenTelemetry) wired through `pkg/logger`
- [ ] `docs/API.md` — endpoint reference generated from the Swagger spec

---

## 🚦 Quality Gates (every phase)

1. **All tests green** before moving to the next phase.
2. **Minimum coverage** — 60% for service layer, 40% overall; raise gradually.
3. **Import-boundary check passes** — no `gin`/`gorm`/`net/http` in domain, dto, port, service files.
4. **Manual review** for DB schema & auth changes.
5. **Deploy = manual approval** — CI automated, but merge/deploy needs you.

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
| Mapper | Unit | `testing` | none |
| Handler | Unit | `httptest` + fake service | none |
| Repository / Adapter | Integration | Testcontainers PostgreSQL | Docker |
| Full flow | E2E | `httptest` + Testcontainers | Docker |
| CI | Automated | GitHub Actions | Docker |

**Rules:**

- Unit tests: fast, no external deps — they must pass in under a second per package.
- Integration tests: behind `//go:build integration`, run via `make test-integration` (`go test -tags=integration ./...`).
- E2E tests: cover at least one full user journey per major feature.
- Fakes are defined in `_test.go` files next to the port they satisfy, so production code cannot import them.
- Prefer fakes over generated mocks: a fake implements the port interface and is easier to read than a mock's call assertions.

---

## 🛠️ Tech Stack

| Concern | Choice |
|---|---|
| HTTP framework | Gin |
| Database | PostgreSQL + GORM |
| Migrations | `golang-migrate` (raw SQL) |
| Cache / rate limit | Redis (`go-redis`) |
| Config | env-driven (`internal/config`) with startup validation |
| Logging | Zap + `gin-contrib/zap` |
| IDs | `google/uuid` (UUID v7) |
| Auth | JWT (HS256) + bcrypt |
| Mail | `Mailer` port — SMTP implementation + fake in tests |
| Storage | Local + S3 (MinIO in tests) |
| Docs | Swagger (`swag`) |
| Testing | `testing`, `httptest`, Testcontainers |
| Lint | `golangci-lint` (incl. import-boundary rules) |
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
