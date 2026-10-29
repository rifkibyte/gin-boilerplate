# Architecture

> Gin + PostgreSQL + GORM boilerplate built on **Hexagonal Architecture (Ports & Adapters)**.
> This document is the source of truth for how code is organised. `PLAN.md` describes the
> order in which that code gets written.

---

## Table of Contents

- [Hexagonal Architecture](#hexagonal-architecture)
- [Motivation](#motivation)
- [Package Layout](#package-layout)
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
- Unit-test business logic with fake ports — no database, no network.

---

## Package Layout

Feature-based layout, with each feature split into an **inner package** (pure domain) and
**outer packages** (adapters). The split is what makes the boundary *enforceable* — Go
linters enforce rules per **package**, so the domain must live in a package of its own.

```txt
.
├── cmd/
│   ├── api/main.go                 # Wires everything, starts the server
│   └── seed/main.go                # Seeding entry point
├── internal/
│   ├── config/                     # Env-driven configuration (single source of truth)
│   ├── user/                       # ── INNER package: pure domain ──
│   │   ├── domain.go               # Domain entity (plain struct, no GORM tags)
│   │   ├── dto.go                  # Request/Response structs (json + binding tags)
│   │   ├── port.go                 # Repository interfaces (consumed by service)
│   │   ├── errors.go               # Feature-scoped sentinel errors
│   │   └── service.go              # Business logic (depends on ports only)
│   │   ├── http/                   # ── OUTER: HTTP adapter ──
│   │   │   ├── handler.go          # Gin handler (imports gin)
│   │   │   └── errors.go           # sentinel → AppError mapping for this feature
│   │   └── adapter/                # ── OUTER: persistence adapter ──
│   │       ├── record.go           # GORM model (gorm tags, table mapping)
│   │       ├── mapper.go           # record ↔ domain conversion
│   │       └── repository_gorm.go  # Implements user.UserRepository using GORM
│   ├── auth/                       # Same three-package shape
│   │   ├── domain.go
│   │   ├── dto.go
│   │   ├── port.go                 # UserFinder, TokenIssuer, TokenStore, Mailer
│   │   ├── errors.go
│   │   ├── service.go
│   │   ├── http/
│   │   │   ├── handler.go
│   │   │   └── errors.go
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
│   │   ├── http/
│   │   │   ├── handler.go
│   │   │   └── errors.go
│   │   └── adapter/
│   │       ├── record.go           # GORM model for file metadata
│   │       ├── mapper.go
│   │       ├── repository_gorm.go
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
├── swagger/                        # GENERATED Swagger package (swag init -o swagger)
├── docs/                           # Markdown docs only (this file, PLAN.md)
├── deploy/                         # Dockerfile, docker-compose.yml
├── testutil/                       # Testcontainers helpers (integration build tag)
└── Makefile
```

### Why the feature is split into three packages

| Package | Contains | Imports `gin`? | Imports `gorm`? |
|---|---|---|---|
| `internal/user` | domain, dto, port, errors, service | **No** | **No** |
| `internal/user/http` | Gin handler, feature error mapping | Yes | No |
| `internal/user/adapter` | GORM model, mapper, repository | No | Yes |

**This is not decoration — it is what makes the rule enforceable.** A linter analyses
packages, not files. If `handler.go` lived beside `service.go`, then `internal/user`
would import `gin`, and "the domain layer never imports gin" would be unenforceable
by any tool. With the split, `internal/user` is a package that simply *does not*
import gin, and `depguard` can reject it in one line:

```yaml
# .golangci.yml
linters-settings:
  depguard:
    rules:
      domain-purity:
        files:
          - "**/internal/*/*.go"   # inner feature packages, not */http/ or */adapter/
        deny:
          - pkg: github.com/gin-gonic/gin
            desc: "the inner feature package must stay HTTP-free"
          - pkg: gorm.io/gorm
            desc: "the inner feature package must stay persistence-free"
```

**Key conventions:**

- **One feature = three packages.** `internal/user/` is the domain, `internal/user/http/`
  is the HTTP adapter, `internal/user/adapter/` holds the infrastructure adapters.
- **`port.go`** holds the small interfaces the service needs (defined *here*, not in the adapter).
- **`errors.go`** (inner) holds feature-scoped sentinel errors (`user.ErrEmailTaken`) —
  the domain never speaks HTTP.
- **`http/errors.go`** holds the *feature-local* mapping from sentinel → `AppError`.
- **The service never imports `http/` or `adapter/`.** The direction is one-way.
- **`internal/router/`** is the single place where URL paths are declared. Handlers never
  contain path strings.
- **`cmd/api/main.go`** is the *only* place that knows both ports and adapters, and wires
  them together.

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
}

type UserResponse struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    Role      Role   `json:"role"`
    CreatedAt string `json:"created_at"`
}
```

**Rule — `Role` is never accepted from the client.** The public register endpoint always
creates a `RoleUser`. Roles are assigned server-side only (by the seeder, or by an
admin-only endpoint). A `Role` field on a request DTO is a privilege-escalation hole.

**Rule:** DTOs never leak into the service layer as-is. The handler maps DTO → domain type before calling the service, and maps domain → DTO before responding.

### Ports

Ports are **small interfaces** defined in the package that *consumes* them. This is Go's idiom: *"accept interfaces, return structs."*

Read and write capabilities are **separate interfaces**, so a consumer that only reads
never has to implement `Delete`:

```go
// internal/user/port.go
package user

import "context"

// UserReader — read-only capability.
type UserReader interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
}

// UserWriter — write-only capability.
type UserWriter interface {
    Save(ctx context.Context, u *User) error
}
```

The service depends on exactly what it uses:

```go
type Service struct {
    reader UserReader
    writer UserWriter
    logger *zap.Logger
}
```

**Rule:** don't create a "god interface" with 20 methods. Split by capability — one port per responsibility. If `auth` needs user lookup, `auth` defines its own `UserFinder` interface with only the methods it uses (even if it's the same GORM adapter underneath).

**Rule:** only declare methods you actually call. An unused method is a promise to every
implementor (and every fake) that they will never need to keep.

### Service

The service contains **business logic only**. It knows nothing about HTTP or GORM — it works with domain types and port interfaces.

```go
// internal/user/service.go
package user

import (
    "context"
    "errors"

    "go.uber.org/zap"
    "golang.org/x/crypto/bcrypt"

    "myapp/pkg/id"
)

// CreateUserCommand — the input to Create. A command struct keeps the signature
// stable as the use case grows, and avoids a 4-argument function.
type CreateUserCommand struct {
    Email    string
    Password string
}

type Service struct {
    reader UserReader
    writer UserWriter
    logger *zap.Logger
}

func NewService(reader UserReader, writer UserWriter, logger *zap.Logger) *Service {
    return &Service{reader: reader, writer: writer, logger: logger}
}

func (s *Service) Create(ctx context.Context, cmd CreateUserCommand) (*User, error) {
    existing, err := s.reader.FindByEmail(ctx, cmd.Email)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, err
    }
    if existing != nil {
        return nil, ErrEmailTaken
    }

    hashed, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }

    u := &User{
        ID:       id.NewV7(),
        Email:    cmd.Email,
        Password: string(hashed),
        Role:     RoleUser, // assigned server-side, never from the request
    }
    if err := s.writer.Save(ctx, u); err != nil {
        return nil, err
    }

    return u, nil
}
```

**Rule:** the service returns **sentinel errors** (`user.ErrEmailTaken`), never HTTP status codes. It also never *swallows* errors — the repository's `ErrNotFound` must be checked explicitly, not discarded with `_`.

**Rule — the service uses `*zap.Logger` directly.** Zap is a pure structured-logging
library with no I/O policy of its own, so depending on it does not compromise the
hexagon. Defining a `Logger` *port* for every feature would mean duplicating an
interface N times to gain nothing. (If you need to swap the logger, wrap it in
`pkg/logger` — one place, not one per feature.)

### Adapters

Adapters implement ports. They live under `adapter/` and are the *only* place that touches a concrete technology — in this case, **PostgreSQL via GORM**.

**`adapter/record.go`** — the GORM model (persistence struct):

```go
// internal/user/adapter/record.go
package adapter

import (
    "time"

    "github.com/google/uuid"
)

type UserRecord struct {
    ID        uuid.UUID `gorm:"primaryKey;type:uuid"`
    Email     string    `gorm:"uniqueIndex;not null"`
    Password  string    `gorm:"not null"`
    Role      string    `gorm:"not null;index"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
    UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (UserRecord) TableName() string { return "users" }
```

**`adapter/mapper.go`** — pure conversion functions between record and domain. Because the
record holds a `uuid.UUID` and the domain holds a `string`, the mapper **parses**, and
therefore **returns an error**:

```go
// internal/user/adapter/mapper.go
package adapter

import (
    "fmt"

    "github.com/google/uuid"

    "myapp/internal/user"
)

func toDomain(r UserRecord) (*user.User, error) {
    id, err := uuid.Parse(r.ID.String())
    if err != nil {
        return nil, fmt.Errorf("parsing user id: %w", err)
    }

    return &user.User{
        ID:        id.String(),
        Email:     r.Email,
        Password:  r.Password,
        Role:      user.Role(r.Role),
        CreatedAt: r.CreatedAt,
        UpdatedAt: r.UpdatedAt,
    }, nil
}

func toRecord(u *user.User) (UserRecord, error) {
    id, err := uuid.Parse(u.ID)
    if err != nil {
        return UserRecord{}, fmt.Errorf("parsing user id: %w", err)
    }

    return UserRecord{
        ID:        id,
        Email:     u.Email,
        Password:  u.Password,
        Role:      string(u.Role),
        CreatedAt: u.CreatedAt,
        UpdatedAt: u.UpdatedAt,
    }, nil
}
```

**Rule:** a mapper that can fail returns an error. A mapper that cannot fail returns
nothing extra. Never swallow a parse error.

**`adapter/repository_gorm.go`** — the GORM implementation of `user.UserReader` / `user.UserWriter`:

```go
// internal/user/adapter/repository_gorm.go
package adapter

import (
    "context"
    "errors"
    "fmt"

    "gorm.io/gorm"

    "myapp/internal/user"
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
        return nil, fmt.Errorf("finding user by email: %w", err)
    }

    return toDomain(rec)
}

func (r *UserRepositoryGORM) Save(ctx context.Context, u *user.User) error {
    rec, err := toRecord(u)
    if err != nil {
        return err
    }

    if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
        return fmt.Errorf("creating user: %w", err)
    }

    u.ID = rec.ID.String()
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
// internal/user/http/handler.go
package http

import (
    "time"

    "github.com/gin-gonic/gin"

    "myapp/internal/user"
    "myapp/pkg/response"
)

type Handler struct {
    svc *user.Service
}

func NewHandler(svc *user.Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
    var req user.CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, err) // uses the custom validator's user-friendly message
        return
    }

    cmd := user.CreateUserCommand{
        Email:    req.Email,
        Password: req.Password,
    }

    u, err := h.svc.Create(c.Request.Context(), cmd)
    if err != nil {
        response.FromError(c, toAppError(err))
        return
    }

    response.Created(c, toUserResponse(u))
}

func toUserResponse(u *user.User) user.UserResponse {
    return user.UserResponse{
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
package router

import (
    "github.com/gin-gonic/gin"

    "myapp/internal/auth/http"
    "myapp/internal/middleware"
    "myapp/internal/user/http"
)

func New(u *userhttp.Handler, a *authhttp.Handler) *gin.Engine {
    // gin.New(), not gin.Default() — we install our own middleware chain below,
    // so the default Logger and Recovery are deliberately not used.
    r := gin.New()
    r.Use(middleware.RequestID(), middleware.AccessLog(), middleware.Recovery())

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

Errors travel in three stages:

### 1. Domain stage — the service returns a feature-scoped sentinel error

```go
// internal/user/errors.go
package user

import "errors"

var (
    ErrNotFound   = errors.New("user not found")
    ErrEmailTaken = errors.New("email already registered")
)
```

### 2. Feature mapping stage — each feature maps its own sentinels to `AppError`

The mapping lives in the feature's **`http/` package**, right next to the handler that
needs it. It is *not* global, because a global registry would have to import every
feature's domain package — which inverts the dependency direction and creates a
bottleneck every time a new feature is added.

```go
// internal/user/http/errors.go
package http

import (
    "errors"

    apperrors "myapp/pkg/errors"
    "myapp/internal/user"
)

// toAppError maps this feature's sentinel errors to transport-level errors.
// Unknown errors pass through untouched and become a 500.
func toAppError(err error) error {
    switch {
    case errors.Is(err, user.ErrNotFound):
        return apperrors.NotFound("user.not_found", "user not found")
    case errors.Is(err, user.ErrEmailTaken):
        return apperrors.Conflict("user.email_taken", "email already registered")
    default:
        return err
    }
}
```

**Rule:** one `toAppError` per feature — *one function*, not one check per handler method.
The rule protects against scattering `errors.Is` across every handler; a single
feature-local switch does not scatter anything.

### 3. Transport stage — `pkg/response` renders the error

```go
// pkg/errors/errors.go
package errors

import "fmt"

// AppError carries an HTTP status code and a stable, machine-readable code.
type AppError struct {
    Status  int
    Code    string // e.g. "user.email_taken"
    Message string // human-readable, safe to show a client
    cause   error  // wrapped, never shown to the client
}

func (e *AppError) Error() string { return e.Message }

func (e *AppError) Unwrap() error { return e.cause }

func New(
    status int,
    code string,
    message string,
    cause error,
) *AppError {
    return &AppError{
        Status:  status,
        Code:    code,
        Message: message,
        cause:   cause,
    }
}

func NotFound(code, message string) *AppError {
    return New(
        http.StatusNotFound,
        code,
        message,
        nil,
    )
}

func Conflict(code, message string) *AppError {
    return New(
        http.StatusConflict,
        code,
        message,
        nil,
    )
}
```

> **Note:** `pkg/errors` is imported as `apperrors` everywhere, because the package name
> `errors` collides with the stdlib `errors` package. This alias is **mandatory** — see
> the [FAQ](#why-is-pkgerrors-aliased-as-apperrors).

```go
// pkg/response/response.go
package response

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    apperrors "myapp/pkg/errors"
)

// ErrorBody is the single error shape returned by every endpoint.
type ErrorBody struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ErrorEnvelope struct {
    Error ErrorBody `json:"error"`
}

// FromError renders err as the standard error envelope.
// Unknown errors are logged with their cause and reported as a generic 500 —
// internals never leak to the client.
func FromError(c *gin.Context, err error) {
    var appErr *apperrors.AppError
    if errors.As(err, &appErr) {
        c.AbortWithStatusJSON(appErr.Status, ErrorEnvelope{
            Error: ErrorBody{Code: appErr.Code, Message: appErr.Message},
        })
        return
    }

    logger := logger.FromContext(c)
    logger.Error("unhandled error", zap.Error(err))

    c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorEnvelope{
        Error: ErrorBody{Code: "internal_error", Message: "internal server error"},
    })
}
```

**Rules:**

- The inner feature package **never** imports `net/http` and never constructs status codes.
- Each feature owns its sentinel → `AppError` mapping, in `internal/<feature>/http/errors.go`.
- Internal errors (DB failures, upstream timeouts) map to `500`, are logged with the full
  cause, and the client only sees a generic message.
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
- The DB column type is `uuid`; the Go type is `string` at the domain level and
  `uuid.UUID` inside the adapter.

```go
// internal/user/domain.go
type User struct {
    ID string // UUID v7, canonical 36-char form
    // ...
}
```

**Rule:** IDs are generated in the service, never by the database. The adapter must not insert a record with an empty ID.

**Rule — the adapter converts, and the conversion can fail.** The domain speaks `string`;
the record speaks `uuid.UUID`. The mapper therefore parses, and returns an error when
the stored value is not a valid UUID. A malformed row is a real data problem and must
not be silently swallowed.

---

## Data Flow

A single request flows through the layers like this:

```
HTTP Request
   │
   ▼
[internal/user/http.Handler]  ── parse DTO, validate, map to command ──┐
   │                                                                  │
   ▼                                                                  │
[internal/user.Service]  ── business rules, calls ports ─────────────┤
   │                                                                  │
   ▼                                                                  │
[UserReader / UserWriter] (interfaces)                              │
   │                                                                  │
   ▼                                                                  │
[internal/user/adapter.RepositoryGORM] ── SQL against PostgreSQL ────┘
   │
   ▼
HTTP Response
```

**Mapping boundaries:**

| Boundary | Mapper | Location |
|---|---|---|
| HTTP ⇄ Domain | `toUserResponse`, `CreateUserCommand` | `internal/user/http/handler.go` |
| Domain ⇄ GORM record | `toDomain`, `toRecord` | `internal/user/adapter/mapper.go` |
| Domain error ⇄ HTTP | `toAppError` | `internal/user/http/errors.go` |

No mapper is allowed to have side effects. No layer is allowed to skip a boundary.

---

## Recommendations

### Repository

Don't try to build **universal methods** in the repository — they're hard to extend as the project grows. Instead, write **small, single-responsibility methods**, split by capability.

```go
// ❌ Avoid: generic, hard to optimize and extend
type Repository interface {
    Find(ctx context.Context, cond map[string]any) ([]*User, error)
}

// ❌ Also avoid: one wide interface mixing reads and writes
type Repository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    Save(ctx context.Context, u *User) error
    Update(ctx context.Context, u *User) error
    Delete(ctx context.Context, id string) error
}

// ✅ Prefer: split by capability, one responsibility per method
type UserReader interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
}

type UserWriter interface {
    Save(ctx context.Context, u *User) error
}
```

Why:

- Each method can be optimized independently (index hints, joins, preloading).
- The interface is self-documenting — you see exactly what the service needs.
- Adding a new query doesn't risk breaking existing ones.
- Splitting read from write keeps fakes small — a fake reader never implements `Save`.

### Mapper

Keep mappers **pure and explicit**:

- One file per feature: `adapter/mapper.go`.
- One function per direction: `toDomain`, `toRecord`.
- Never embed GORM tags in domain structs.
- Never let a handler touch a `UserRecord` directly.
- If the conversion can fail (UUID parsing), the mapper returns an error.

If a feature grows large, split mappers by entity: `adapter/mapper_user.go`, `adapter/mapper_profile.go` — still in the same package.

### Dependency Wiring

All wiring happens in **one place**: `cmd/api/main.go`. No DI framework, no magic — just constructor calls.

```go
// cmd/api/main.go
package main

import (
    userhttp "myapp/internal/user/http"
    useradapter "myapp/internal/user/adapter"
    "myapp/internal/user"
    "myapp/pkg/cache"
    "myapp/pkg/database"
    "myapp/pkg/logger"
)

func main() {
    cfg := config.Load()
    log := logger.New(cfg.Log)

    db := database.NewPostgres(cfg.DB) // *gorm.DB
    redisClient := cache.NewRedis(cfg.Redis)

    userRepo := useradapter.NewUserRepositoryGORM(db)
    userSvc := user.NewService(userRepo, userRepo, log)
    userHandler := userhttp.NewHandler(userSvc)

    authSvc := auth.NewService(userRepo, authadapter.NewJWTIssuer(cfg.JWT), log)
    authHandler := authhttp.NewHandler(authSvc)

    r := router.New(userHandler, authHandler)
    srv := &http.Server{Addr: cfg.HTTP.Addr, Handler: r}

    // graceful shutdown ...
}
```

**Rule:** if a feature needs a new dependency, add it in `main.go` — never reach across packages to instantiate adapters directly inside a service.

**Rule — adapter sub-packages are always aliased on import.** `internal/user/adapter`,
`internal/auth/adapter`, and `internal/file/adapter` all have the package name `adapter`;
without an alias the import collides. The convention is `<feature>adapter` and
`<feature>http`.

### Import Rules

These are the mechanical rules that keep the hexagon intact, and they are enforceable
with `depguard` because each rule targets a distinct **package**.

| Package | May import | Must NOT import |
|---|---|---|
| `internal/<feature>` (inner) | stdlib, `pkg/id`, `pkg/logger`, `pkg/errors` | `gin`, `gorm.io/gorm`, `net/http`, `internal/<feature>/http`, `internal/<feature>/adapter` |
| `internal/<feature>/http` | `gin`, `internal/<feature>`, `pkg/response`, `pkg/errors` | `gorm.io/gorm`, `internal/<feature>/adapter` |
| `internal/<feature>/adapter` | `gorm.io/gorm`, `internal/<feature>`, `pkg/id` | `gin`, `net/http` |
| `internal/middleware` | `gin`, `pkg/*` | `gorm.io/gorm`, `internal/<feature>` |
| `internal/router` | `gin`, `internal/<feature>/http` | `gorm.io/gorm`, `internal/<feature>/adapter` |
| `internal/config` | stdlib, third-party config libs | everything else |
| `cmd/api` | everything | — |

**Rules:**

- A feature package must never import another feature's `adapter/` sub-package.
- Cross-feature collaboration happens through **ports defined in the consuming feature**,
  and through **domain types**, which may cross packages:
  `auth` may import `user.User`, but never `user/adapter`.
- `internal/<feature>/http` must not import `internal/<feature>/adapter` — the handler
  talks to the service, not to the database.

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

- **Fake ports, not mocks.** A hand-written fake implementing the port interface is
  simpler than a generated mock and lives next to the port it satisfies.
- Integration tests carry the `//go:build integration` tag and run via `make test-integration`.
- Fakes must live in the **inner** package (`internal/user/fake_repository_test.go`) so
  they can't be imported by production code.
- Every feature gets a **table-driven test for its `toAppError` mapping** — each sentinel
  must map to the intended status code. This is the contract clients depend on.

```go
// internal/user/fake_repository_test.go
package user

import "context"

type fakeUserReader struct {
    users map[string]*User
}

func newFakeUserReader() *fakeUserReader {
    // Maps are always initialized explicitly — a nil map panics on write.
    return &fakeUserReader{users: map[string]*User{}}
}

func (f *fakeUserReader) FindByEmail(_ context.Context, email string) (*User, error) {
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

Cross-cutting concerns should *not* be imported by inner feature code directly unless
they're pure utilities (e.g. `logger`, `id`, `errors`).

### Why is `pkg/errors` aliased as `apperrors`?

Because the package name `errors` collides with the stdlib `errors` package, and a single
file often needs both — `errors.Is`/`errors.As` alongside `apperrors.NotFound`. The
alias is mandatory wherever both appear:

```go
import (
    "errors"

    apperrors "myapp/pkg/errors"
)
```

**Rule:** always alias `myapp/pkg/errors` as `apperrors`. Never import it unaliased.

### Where does validation live?

Two layers:

- **HTTP-level validation** (shape, required fields, format) — in the handler via `gin/binding` tags on DTOs, with a custom validator that produces user-friendly messages.
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

### Why is the feature split into three packages instead of one?

Because Go linters enforce rules per **package**, not per file. If `handler.go` sat
beside `service.go`, the package would import `gin`, and "the domain layer never imports
gin" would be unenforceable by any tool — it would be a convention, not a guarantee.
The split turns that guarantee into a single `depguard` rule.

### Why does the mapper return an error?

Because the domain holds a UUID as `string` and the record holds `uuid.UUID`. Converting
between them requires parsing, and parsing can fail on a malformed row. A mapper that
cannot fail returns nothing extra; a mapper that can fail returns the error — never
swallows it.

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
- [depguard (golangci-lint)](https://github.com/OpenPeeDeeP/depguard)

---

Previous: [PLAN.md](PLAN.md) · Next: [AGENT.md](../AGENT.md)
