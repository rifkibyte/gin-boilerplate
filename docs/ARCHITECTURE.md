# Architecture

> Gin + PostgreSQL + GORM boilerplate built on **Hexagonal Architecture (Ports & Adapters)**.
> This document is the source of truth for how code is organised. `PLAN.md` describes the
> order in which that code gets written.

---

## Table of Contents

- [Hexagonal Architecture](#hexagonal-architecture)
- [Motivation](#motivation)
- [Module Structure](#module-structure)
- [Layer Responsibilities](#layer-responsibilities)
  - [Domain](#domain)
  - [DTO](#dto)
  - [Ports](#ports)
  - [Service](#service)
  - [Adapters](#adapters)
  - [Handler](#handler)
  - [Router](#router)
- [Error Handling](#error-handling)
- [Identifiers](#identifiers)
- [Data Flow](#data-flow)
- [Recommendations](#recommendations)
  - [Repository](#repository)
  - [Mapper](#mapper)
  - [Dependency Wiring](#dependency-wiring)
  - [Import Rules](#import-rules)
- [Testing Strategy](#testing-strategy)
- [FAQ](#faq)
- [Links](#links)

---

## Hexagonal Architecture

This boilerplate is based on [Hexagonal Architecture](https://en.wikipedia.org/wiki/Hexagonal_architecture_(software)), also known as **Ports and Adapters**.

The core idea: **business logic sits at the center and knows nothing about the outside world**. Everything external — HTTP (Gin), PostgreSQL (GORM), Redis, S3, SMTP — is an *adapter* plugged into a *port* (a Go interface). The core depends only on interfaces, never on concrete implementations.

```
                    ┌─────────────────────────────┐
                    │        HTTP Handler         │  ← adapter (Gin)
                    └──────────────┬──────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │           Service           │  ← business logic
                    │  (depends on ports only)    │
                    └──────┬───────────────┬──────┘
                           │               │
              ┌────────────▼───┐   ┌───────▼────────┐
              │  UserRepository│   │  Mailer (port) │
              │      (port)    │   └───────┬────────┘
              └────────┬───────┘           │
                       │                   │
              ┌────────▼───────┐   ┌───────▼────────┐
              │ GORM adapter   │   │ SMTP adapter   │
              │  (PostgreSQL)  │   └────────────────┘
              └────────────────┘
```

In Go, this maps naturally to **interfaces**. Every port is a small interface defined *where it's consumed* (the service), not where it's implemented (the adapter).

---

## Motivation

The main reason for using Hexagonal Architecture is **to separate business logic from infrastructure**. This separation allows you to:

- Swap PostgreSQL for another database without touching business logic.
- Replace GORM with raw SQL or `sqlc` without rewriting handlers or services.
- Change file storage from local disk to S3 without rewriting business rules.
- Replace SMTP with a third-party mail API without touching the service layer.
- Unit-test business logic with mocked ports — no database, no network.

---

## Module Structure

Feature-based layout, with hexagonal layers inside each feature package:

```txt
.
├── cmd/
│   ├── api/main.go                 # Wires everything, starts the server
│   └── seed/main.go                # Seeding entry point
├── internal/
│   ├── config/                     # Env-driven configuration (single source of truth)
│   ├── user/
│   │   ├── domain.go               # Domain entity (plain struct, no GORM tags)
│   │   ├── dto.go                  # Request/Response structs (json + binding tags)
│   │   ├── port.go                 # Repository interface (consumed by service)
│   │   ├── errors.go               # Feature-scoped sentinel errors
│   │   ├── service.go              # Business logic (depends on port only)
│   │   ├── handler.go              # HTTP adapter (Gin)
│   │   └── adapter/
│   │       ├── record.go           # GORM model (gorm tags, table mapping)
│   │       ├── mapper.go           # record ↔ domain conversion
│   │       └── repository_gorm.go  # Implements user.Repository using GORM
│   ├── auth/
│   │   ├── domain.go
│   │   ├── dto.go
│   │   ├── port.go                 # UserFinder, TokenIssuer, TokenStore, Mailer
│   │   ├── errors.go
│   │   ├── service.go
│   │   ├── handler.go
│   │   └── adapter/
│   │       ├── mailer_smtp.go
│   │       ├── token_jwt.go
│   │       └── token_redis.go      # Refresh-token store (Redis)
│   ├── file/
│   │   ├── domain.go
│   │   ├── dto.go
│   │   ├── port.go                 # Storage interface
│   │   ├── errors.go
│   │   ├── service.go
│   │   ├── handler.go
│   │   └── adapter/
│   │       ├── storage_local.go
│   │       └── storage_s3.go
│   ├── middleware/                 # Gin middleware (request ID, CORS, auth, rate limit)
│   └── router/                     # Route registration (the only place paths are declared)
├── pkg/
│   ├── logger/                     # Zap wrapper (cross-cutting)
│   ├── database/                   # GORM + PostgreSQL connection setup
│   ├── cache/                      # Redis client wrapper
│   ├── errors/                     # Standardized AppError type
│   ├── response/                   # JSON success/error response helpers
│   └── id/                         # UUID v7 generation
├── migrations/                     # SQL migration files
├── docs/                           # Swagger documentation (generated) + these docs
├── deploy/                         # Dockerfile, docker-compose.yml
├── testutil/                       # Testcontainers helpers (integration build tag)
└── Makefile
```

**Key conventions:**

- **One feature = one package.** Everything related to `user` lives in `internal/user/`.
- **`port.go`** holds the small interfaces the service needs (defined *here*, not in the adapter).
- **`errors.go`** holds feature-scoped sentinel errors (`user.ErrEmailTaken`) — the domain never speaks HTTP.
- **`adapter/`** holds concrete implementations (GORM, SMTP, S3, Redis). The service never imports this subpackage directly.
- **`handler.go`** is the HTTP adapter — it translates HTTP into service calls. It knows about Gin; the service does not.
- **`internal/router/`** is the single place where URL paths are declared. Handlers never contain path strings.
- **`cmd/api/main.go`** is the *only* place that knows both ports and adapters, and wires them together.

---

## Layer Responsibilities

### Domain

Domain entities are plain Go structs with **zero dependencies** on GORM, HTTP, or any library. They represent the core concepts of the feature.

```go
// internal/user/domain.go
package user

import "time"

type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
)

type User struct {
    ID        string
    Email     string
    Password  string // already hashed
    Role      Role
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**Rule:** if the struct has a `gorm:"..."` tag or a `json:"..."` tag, it is *not* a domain entity — it's a persistence record or a DTO. Keep them separate; use a mapper to convert.

### DTO

DTOs are the request/response shapes for the HTTP layer. They carry `json` and `binding` (validation) tags.

```go
// internal/user/dto.go
package user

type CreateUserRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Role     Role   `json:"role" binding:"required,oneof=admin user"`
}

type UserResponse struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    Role      Role   `json:"role"`
    CreatedAt string `json:"created_at"`
}
```

**Rule:** DTOs never leak into the service layer as-is. The handler maps DTO → domain type before calling the service, and maps domain → DTO before responding.

### Ports

Ports are **small interfaces** defined in the package that *consumes* them. This is Go's idiom: *"accept interfaces, return structs."*

```go
// internal/user/port.go
package user

import "context"

type Repository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    FindByRole(ctx context.Context, role Role) ([]*User, error)
    Save(ctx context.Context, u *User) error
    Update(ctx context.Context, u *User) error
    Delete(ctx context.Context, id string) error
}
```

**Rule:** don't create a "god interface" with 20 methods. Split by responsibility — one port per capability the service needs. If `auth` needs user lookup, `auth` defines its own `UserFinder` interface with only the methods it uses (even if it's the same GORM adapter underneath).

### Service

The service contains **business logic only**. It knows nothing about HTTP or GORM — it works with domain types and port interfaces.

```go
// internal/user/service.go
package user

import (
    "context"
    "errors"
    "golang.org/x/crypto/bcrypt"
    "myapp/pkg/id"
)

type Service struct {
    repo   Repository
    logger Logger
}

func NewService(repo Repository, logger Logger) *Service {
    return &Service{repo: repo, logger: logger}
}

func (s *Service) Create(ctx context.Context, email, password string, role Role) (*User, error) {
    existing, err := s.repo.FindByEmail(ctx, email)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, err
    }
    if existing != nil {
        return nil, ErrEmailTaken
    }

    hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }

    u := &User{ID: id.NewV7(), Email: email, Password: string(hashed), Role: role}
    if err := s.repo.Save(ctx, u); err != nil {
        return nil, err
    }
    return u, nil
}
```

**Rule:** the service returns **sentinel errors** (`user.ErrEmailTaken`), never HTTP status codes. It also never *swallows* errors — the repository's `ErrNotFound` must be checked explicitly, not discarded with `_`.

### Adapters

Adapters implement ports. They live under `adapter/` and are the *only* place that touches a concrete technology — in this case, **PostgreSQL via GORM**.

**`adapter/record.go`** — the GORM model (persistence struct):

```go
// internal/user/adapter/record.go
package adapter

import "time"

type UserRecord struct {
    ID        string    `gorm:"primaryKey;type:uuid"`
    Email     string    `gorm:"uniqueIndex;not null"`
    Password  string    `gorm:"not null"`
    Role      string    `gorm:"not null;index"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (UserRecord) TableName() string { return "users" }
```

**`adapter/mapper.go`** — pure conversion functions between record and domain:

```go
// internal/user/adapter/mapper.go
package adapter

import "myapp/internal/user"

func toDomain(r UserRecord) *user.User {
    return &user.User{
        ID:        r.ID,
        Email:     r.Email,
        Password:  r.Password,
        Role:      user.Role(r.Role),
        CreatedAt: r.CreatedAt,
        UpdatedAt: r.UpdatedAt,
    }
}

func toRecord(u *user.User) UserRecord {
    return UserRecord{
        ID:        u.ID,
        Email:     u.Email,
        Password:  u.Password,
        Role:      string(u.Role),
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
    }
}
```

**`adapter/repository_gorm.go`** — the GORM implementation of `user.Repository`:

```go
// internal/user/adapter/repository_gorm.go
package adapter

import (
    "context"
    "errors"
    "myapp/internal/user"
    "gorm.io/gorm"
)

type UserRepositoryGORM struct {
    db *gorm.DB
}

func NewUserRepositoryGORM(db *gorm.DB) *UserRepositoryGORM {
    return &UserRepositoryGORM{db: db}
}

func (r *UserRepositoryGORM) FindByEmail(ctx context.Context, email string) (*user.User, error) {
    var rec UserRecord
    err := r.db.WithContext(ctx).Where("email = ?", email).First(&rec).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, user.ErrNotFound
    }
    if err != nil {
        return nil, err
    }
    return toDomain(rec), nil
}

func (r *UserRepositoryGORM) Save(ctx context.Context, u *user.User) error {
    rec := toRecord(u)
    if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
        return err
    }
    u.ID = rec.ID
    u.CreatedAt = rec.CreatedAt
    return nil
}
```

**Rule:** the adapter *imports* the domain and the port. The port never imports the adapter. This is the **Dependency Inversion Principle** in practice.

### Handler

Handlers are the **HTTP adapters**. They:

1. Parse and validate the request into a DTO.
2. Map DTO → domain type.
3. Call the service.
4. Map the result/error back to HTTP response.

```go
// internal/user/handler.go
package user

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "myapp/pkg/response"
)

type Handler struct {
    svc *Service
}

func NewHandler(svc *Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    u, err := h.svc.Create(c.Request.Context(), req.Email, req.Password, req.Role)
    if err != nil {
        response.FromError(c, err) // maps sentinel errors → status codes
        return
    }

    response.Created(c, toUserResponse(u))
}

func toUserResponse(u *User) UserResponse {
    return UserResponse{
        ID:        u.ID,
        Email:     u.Email,
        Role:      u.Role,
        CreatedAt: u.CreatedAt.Format(time.RFC3339),
    }
}
```

### Router

Routing lives in `internal/router/`, not in the feature packages. This keeps URL paths greppable in one place and avoids handlers importing each other.

```go
// internal/router/router.go
func New(u *user.Handler, a *auth.Handler) *gin.Engine {
    r := gin.New()
    r.Use(middleware.RequestID(), middleware.Logger(), middleware.Recovery())

    v1 := r.Group("/api/v1")
    {
        v1.POST("/users", u.Create)
        v1.POST("/auth/login", a.Login)
    }
    return r
}
```

---

## Error Handling

Errors travel in two stages:

1. **Domain stage** — the service returns a feature-scoped sentinel error:

   ```go
   // internal/user/errors.go
   package user

   import "errors"

   var (
       ErrNotFound   = errors.New("user not found")
       ErrEmailTaken = errors.New("email already registered")
   )
   ```

2. **Transport stage** — a central mapper converts known errors into HTTP responses:

   ```go
   // pkg/errors/errors.go
   package errors

   type AppError struct {
       Status  int    // HTTP status code
       Code    string // stable machine-readable code, e.g. "user.email_taken"
       Message string // human-readable message
       cause   error
   }

   func (e *AppError) Error() string { return e.Message }
   func (e *AppError) Unwrap() error { return e.cause }

   func NotFound(code, msg string) *AppError { ... }
   func Conflict(code, msg string) *AppError { ... }
   ```

   `pkg/response` maps a returned error to a status code:

   ```go
   // pkg/response/response.go
   func FromError(c *gin.Context, err error) {
       var appErr *apperrors.AppError
       if errors.As(err, &appErr) {
           c.AbortWithStatusJSON(appErr.Status, ErrorBody{Code: appErr.Code, Message: appErr.Message})
           return
       }
       // unknown error → 500, log it, never leak internals to the client
       c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorBody{Code: "internal_error"})
   }
   ```

**Rules:**

- The domain package **never** imports `net/http` and never constructs status codes.
- A central registry (e.g. `pkg/errors` or `internal/router`) holds the mapping `sentinel error → AppError`. Do not scatter `errors.Is` checks across every handler.
- Internal errors (DB failures, upstream timeouts) map to `500` and are logged with the full cause; the client only sees a generic message.
- Every error response has the same shape:

  ```json
  { "error": { "code": "user.email_taken", "message": "email already registered" } }
  ```

---

## Identifiers

All entity primary keys are **UUID v7** strings, generated in the **application** layer.

- Generated by `pkg/id.NewV7()` (wrapping `github.com/google/uuid` → `uuid.NewV7()`).
- Time-ordered, so indexes stay dense and range scans on `created_at`/`id` are efficient — unlike UUID v4.
- No database extension required (`gen_random_uuid()` / `pgcrypto` not needed), and the ID is known before the INSERT, so the service can log or enqueue it before the round-trip.
- The DB column type is `uuid`; the Go type is `string` at the domain level and `uuid.UUID` inside the adapter (mapped in `mapper.go`).

```go
// internal/user/domain.go
type User struct {
    ID string // UUID v7, canonical 36-char form
    // ...
}
```

**Rule:** IDs are generated in the service, never by the database. The adapter must not insert a record with an empty ID.

---

## Data Flow

A single request flows through the layers like this:

```
HTTP Request
   │
   ▼
[Handler]  ── parse DTO, validate, map to domain ──┐
   │                                               │
   ▼                                               │
[Service]  ── business rules, calls ports ────────┤
   │                                               │
   ▼                                               │
[Repository Port] (interface)                      │
   │                                               │
   ▼                                               │
[GORM Adapter] ── SQL against PostgreSQL ──────────┘
   │
   ▼
HTTP Response
```

**Mapping boundaries:**

| Boundary | Mapper | Location |
|---|---|---|
| HTTP ⇄ Domain | `toUserResponse`, `fromCreateRequest` | `handler.go` |
| Domain ⇄ GORM record | `toDomain`, `toRecord` | `adapter/mapper.go` |
| Domain error ⇄ HTTP | `AppError` registry | `pkg/errors` + `pkg/response` |

No mapper is allowed to have side effects. No layer is allowed to skip a boundary.

---

## Recommendations

### Repository

Don't try to build **universal methods** in the repository — they're hard to extend as the project grows. Instead, write **small, single-responsibility methods**.

```go
// ❌ Avoid: generic, hard to optimize and extend
type Repository interface {
    Find(ctx context.Context, cond map[string]any) ([]*User, error)
}

// ✅ Prefer: explicit, one responsibility per method
type Repository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    FindByRole(ctx context.Context, role Role) ([]*User, error)
    FindByIDs(ctx context.Context, ids []string) ([]*User, error)
}
```

Why:

- Each method can be optimized independently (index hints, joins, preloading).
- The interface is self-documenting — you see exactly what the service needs.
- Adding a new query doesn't risk breaking existing ones.

### Mapper

Keep mappers **pure and explicit**:

- One file per feature: `adapter/mapper.go`.
- One function per direction: `toDomain`, `toRecord`.
- Never embed GORM tags in domain structs.
- Never let a handler touch a `UserRecord` directly.

If a feature grows large, split mappers by entity: `adapter/mapper_user.go`, `adapter/mapper_profile.go` — still in the same package.

### Dependency Wiring

All wiring happens in **one place**: `cmd/api/main.go`. No DI framework, no magic — just constructor calls.

```go
// cmd/api/main.go
func main() {
    cfg := config.Load()
    log := logger.New(cfg.Log)

    db := database.NewPostgres(cfg.DB)        // *gorm.DB
    redisClient := cache.NewRedis(cfg.Redis)

    userRepo := useradapter.NewUserRepositoryGORM(db)
    userSvc := user.NewService(userRepo, log)
    userHandler := user.NewHandler(userSvc)

    authSvc := auth.NewService(userRepo, authadapter.NewJWTIssuer(cfg.JWT))
    authHandler := auth.NewHandler(authSvc)

    r := router.New(userHandler, authHandler)
    srv := &http.Server{Addr: cfg.HTTP.Addr, Handler: r}

    // graceful shutdown ...
}
```

**Rule:** if a feature needs a new dependency, add it in `main.go` — never reach across packages to instantiate adapters directly inside a service.

### Import Rules

These are the mechanical rules that keep the hexagon intact. `go list -deps` and `golangci-lint`'s `importas`/`depguard` linters can enforce them:

| Package | May import | Must NOT import |
|---|---|---|
| `internal/<feature>` (domain/dto/port/service) | stdlib, `pkg/errors`, `pkg/id`, `pkg/logger` | `gin`, `gorm.io/gorm`, `internal/<feature>/adapter`, `net/http` |
| `internal/<feature>/adapter` | `gorm.io/gorm`, `internal/<feature>` | `gin`, `net/http` |
| `internal/<feature>` (handler) | `gin`, `internal/<feature>`, `pkg/response` | `gorm.io/gorm` |
| `internal/router` | `gin`, `internal/<feature>` | `gorm.io/gorm` |
| `cmd/api` | everything | — |

**Rule:** a feature package must never import another feature's `adapter/` subpackage. Cross-feature collaboration happens through ports defined in the *consuming* feature.

---

## Testing Strategy

The hexagon pays off in tests: the port is the seam.

| Layer | Test kind | Needs infra? |
|---|---|---|
| Domain / service | Unit, with hand-written fake ports | No |
| Mapper | Unit (pure functions) | No |
| Handler | Unit, with a fake service + `httptest` | No |
| Adapter (GORM) | Integration, Testcontainers PostgreSQL | Yes |
| Full flow (E2E) | Integration, Testcontainers | Yes |

**Rules:**

- **Fake ports, not mocks.** A hand-written fake implementing the port interface (`internal/user/service_test.go`) is simpler than a generated mock and lives next to the port it satisfies.
- Integration tests carry the `//go:build integration` tag and run via `make test-integration`.
- Fakes must live in `internal/<feature>/` (e.g. `fake_repository_test.go`) so they can't be imported by production code.

```go
// internal/user/fake_repository_test.go
package user

type fakeRepository struct {
    users map[string]*User
}

func (f *fakeRepository) FindByEmail(_ context.Context, email string) (*User, error) {
    for _, u := range f.users {
        if u.Email == email {
            return u, nil
        }
    }
    return nil, ErrNotFound
}
```

---

## FAQ

### Where do GORM migrations live?

In `migrations/` as raw SQL files, applied via `golang-migrate` (recommended for production) or `gorm.io/gorm/migrator` (fine for development). AutoMigrate is convenient but **not** a substitute for versioned migrations — use it only for local iteration.

```txt
migrations/
├── 000001_create_users.up.sql
├── 000001_create_users.down.sql
├── 000002_create_files.up.sql
└── 000002_create_files.down.sql
```

### Where do cross-cutting concerns live?

In `pkg/`:

- `pkg/logger/` — structured logging (Zap).
- `pkg/errors/` — standardized `AppError` type used across handlers.
- `pkg/response/` — JSON success/error response helpers.
- `pkg/database/` — GORM + PostgreSQL connection setup (pool, logger, tracing).
- `pkg/cache/` — Redis client wrapper.
- `pkg/id/` — UUID v7 generation.
- `internal/middleware/` — Gin middleware (request ID, CORS, rate limit, auth).

Cross-cutting concerns should *not* be imported by domain/service code directly unless they're pure utilities (e.g. `logger`, `id`, `errors`).

### Where does validation live?

Two layers:

- **HTTP-level validation** (shape, required fields, format) — in the handler via `gin/binding` tags on DTOs.
- **Business-rule validation** (email uniqueness, password strength policy, role transitions) — in the service, returning a domain sentinel error.

### Why not put GORM tags directly on the domain struct?

Because it couples the domain to GORM. Once a `gorm:"..."` tag is on `User`, you can't:

- Swap GORM for `sqlc` or raw SQL without touching every domain file.
- Unit-test the domain without importing GORM.
- Reuse the same domain type across features with different persistence needs.

The `record.go` + `mapper.go` split costs ~30 lines per feature and buys you real decoupling.

### Why is the service not allowed to return `*apperrors.AppError`?

Because `AppError` carries an HTTP status code. If the service returns it, the domain layer now depends on transport semantics, and the same service can no longer be driven from a CLI, a gRPC handler, or a queue worker without translation. The service returns *what went wrong* (`user.ErrEmailTaken`); the transport layer decides *what that means over HTTP* (`409 Conflict`).

### Why does the repository return `nil, ErrNotFound` instead of `nil, nil`?

A `(nil, nil)` return is ambiguous: it looks like a success with an empty result. Returning a sentinel error makes "not found" explicit, lets the service use `errors.Is`, and prevents the classic bug where a missing user is treated as an existing one with an empty email.

### Is there a CLI to generate a new feature?

Not by default in Go (no equivalent of NestJS CLI). For now, copy an existing feature folder and rename:

```bash
cp -r internal/user internal/order
# then rename package user → order, update types, adjust main.go wiring
```

If the project grows, a small `go:generate` template or a `make new-feature NAME=order` target is a good addition.

---

## Links

- [Hexagonal Architecture (Alistair Cockburn)](https://alistair.cockburn.us/hexagonal-architecture/)
- [Dependency Inversion Principle](https://trilon.io/blog/dependency-inversion-principle)
- [GORM Documentation](https://gorm.io/docs/)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Go Code Review Comments — Interfaces](https://github.com/golang/go/wiki/CodeReviewComments#interfaces)
- [RFC 9562 — UUID Version 7](https://www.rfc-editor.org/rfc/rfc9562.html)

---

Previous: [PLAN.md](PLAN.md) · Next: [AGENT.md](../AGENT.md)
