# AGENT.md

> Working agreement for AI agents (and humans) contributing to this repository.
> Read this before writing any code.

---

## 🔁 The Loop

```
plan → write ↔ test
```

This is the **only** workflow. Do not skip steps.

1. **Plan** — pick the next unchecked item in `docs/PLAN.md`. Write the plan down
   (a short list of files to create/change) *before* editing anything.
2. **Write** — implement exactly what the plan says. No scope creep.
3. **Test** — every line of code you write ships with a test. Then re-read the
   code and fix what the test reveals.
4. **Loop** — repeat until the phase's **Done when** criteria are met.

### Per-task checklist

- [ ] Plan written down (files to add/modify) before the first edit.
- [ ] Ports and fakes written **before** adapters.
- [ ] Tests written alongside the code, not after.
- [ ] New sentinel errors have a `toAppError` mapping **and** a table-driven test.
- [ ] `make lint` passes — including the `depguard` import-boundary check.
- [ ] `make test` passes (unit — must be fast, no infra).
- [ ] `make test-integration` passes (integration — needs Docker).
- [ ] `docs/PLAN.md` checkboxes updated to reflect reality.
- [ ] No new violations of the [Import Rules](#import-rules).

### If you get stuck

- **Blocked on a decision** → ask. Do not guess at architecture.
- **Blocked on a bug** → write the failing test first, then fix.
- **Unsure where code goes** → check the [Module Structure](#module-structure).
  If the structure doesn't cover your case, **stop and ask** before inventing
  a new layout.

---

## 📐 Architecture

**Source of truth: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).**

This project uses **Hexagonal Architecture (Ports & Adapters)**, organised
feature-first. Each feature is split into three packages — an inner domain package
plus outer `http/` and `adapter/` packages — because Go linters enforce rules per
**package**. If the architecture doc and this file disagree, the architecture doc wins.

```
HTTP Request
   │
   ▼
[internal/<feature>/http.Handler]  ── parse DTO, validate, map ──┐
   │                                                              │
   ▼                                                              │
[internal/<feature>.Service]  ── business rules, calls ports ─────┤
   │                                                              │
   ▼                                                              │
[Port] (interface)                                                │
   │                                                              │
   ▼                                                              │
[internal/<feature>/adapter] ── SQL against PostgreSQL ──────────┘
   │
   ▼
HTTP Response
```

---

## 🔒 Locked Decisions

**Do not re-litigate these. Do not change them without being asked.**

| Decision | Choice | Rationale |
|---|---|---|
| Architecture | Hexagonal (Ports & Adapters), feature-based packages | Business logic testable without infra; swappable adapters |
| Package layout | `internal/<feature>/` (inner) + `internal/<feature>/http/` + `internal/<feature>/adapter/` | Go linters enforce per **package** — the domain must be its own package to be lintable |
| Port shape | Split by capability (`UserReader` / `UserWriter`), only methods actually called | Keeps fakes small; avoids “god interface” |
| Routing | `internal/router/` | All URL paths in one greppable place |
| IDs | **UUID v7**, generated in the service via `pkg/id`; adapter holds `uuid.UUID`, mapper parses | Time-ordered (dense indexes), no DB extension, known before INSERT |
| Errors | Inner sentinel → per-feature `http/errors.go` → `pkg/errors.AppError` → `pkg/response` | Domain never speaks HTTP; no global registry to bottleneck on |
| Roles | Server-side only — the register DTO has **no** `Role` field | Prevents privilege escalation via self-registration |
| Logging | `*zap.Logger` used directly — **no** `Logger` port | Zap is a pure utility; a per-feature logger interface buys nothing |
| Migrations | Raw SQL via `golang-migrate`; AutoMigrate only for local iteration | Versioned, reviewable, reversible |
| Tests | Fakes for ports, Testcontainers for adapters | The hexagon's seam is the port; that's where fakes go |
| Swagger | `swag` v1 (pinned) with `@Router` annotations; output to `swagger/`, not `docs/` | Keeps `docs/` markdown-only; avoids a Go package hiding in a docs folder |
| Wiring | `cmd/api/main.go` only | No DI framework, no magic |

---

## 📁 Module Structure

```txt
.
├── cmd/
│   ├── api/main.go                 # Wires everything, starts the server
│   └── seed/main.go                # Seeding entry point (idempotent)
├── internal/
│   ├── config/                     # Env-driven config, validated at startup
│   ├── <feature>/                  # INNER package — pure domain, no gin, no gorm
│   │   ├── domain.go               # Plain structs — no gorm/json tags
│   │   ├── dto.go                  # json + binding tags
│   │   ├── port.go                 # Interfaces the service consumes
│   │   ├── errors.go               # Feature-scoped sentinel errors
│   │   └── service.go              # Business logic — ports only
│   │   ├── http/                   # OUTER — Gin handler + toAppError mapping
│   │   │   ├── handler.go
│   │   │   └── errors.go
│   │   └── adapter/                # OUTER — GORM / SMTP / S3 / Redis
│   │       ├── record.go           # GORM model
│   │       ├── mapper.go           # record ↔ domain (returns an error)
│   │       └── repository_gorm.go
│   ├── middleware/                 # Request ID, AccessLog, CORS, auth, rate limit
│   └── router/                     # ALL URL paths live here
├── pkg/
│   ├── logger/  database/  cache/  errors/  response/  id/
├── migrations/                     # Raw SQL, up + down
├── swagger/                        # GENERATED Swagger package (swag init -o swagger)
├── docs/                           # Markdown only — ARCHITECTURE.md, PLAN.md
├── testutil/                       # Testcontainers helpers
└── Makefile
```

The feature is split into **three packages** because Go linters enforce rules per
**package**, not per file. With the handler in its own package, "the domain imports
no gin" becomes a single enforceable `depguard` rule.

---

## 🚫 Import Rules

**The mechanical rules that keep the hexagon intact.**

| Package | May import | Must NOT import |
|---|---|---|
| `internal/<feature>` (inner) | stdlib, `pkg/id`, `pkg/logger`, `pkg/errors` | `gin`, `gorm.io/gorm`, `net/http`, `internal/<feature>/http`, `internal/<feature>/adapter` |
| `internal/<feature>/http` | `gin`, `internal/<feature>`, `pkg/response`, `pkg/errors` | `gorm.io/gorm`, `internal/<feature>/adapter` |
| `internal/<feature>/adapter` | `gorm.io/gorm`, `internal/<feature>`, `pkg/id` | `gin`, `net/http` |
| `internal/middleware` | `gin`, `pkg/*` | `gorm.io/gorm`, `internal/<feature>` |
| `internal/router` | `gin`, `internal/<feature>/http` | `gorm.io/gorm`, `internal/<feature>/adapter` |
| `cmd/api` | everything | — |

Additional rules:

- A feature package **must never** import another feature's `adapter/`.
- Cross-feature collaboration goes through ports defined in the **consuming** feature,
  and through **domain types** — `auth` may import `user.User`, never `user/adapter`.
- The inner package **never** imports `net/http` and never constructs status codes.
- `internal/<feature>/http` must not import `internal/<feature>/adapter` — the handler
  talks to the service, not to the database.
- Handlers contain **no URL path strings** — those live in `internal/router/`.
- Handlers **never** touch a `*Record` (GORM model) directly.
- `myapp/pkg/errors` is **always** imported as `apperrors` — it collides with stdlib `errors`.
- Adapter sub-packages are **always** aliased: `<feature>adapter`, `<feature>http`.

If a lint rule catches you, the fix is almost always to add a mapper or a port —
not to relax the rule.

---

## ✍️ Code Conventions

### Domain

```go
// ✅ Plain struct, zero dependencies
type User struct {
    ID        string
    Email     string
    Password  string // already hashed
    Role      Role
    CreatedAt time.Time
}
```

- If a struct has a `gorm:"..."` tag or a `json:"..."` tag, it is **not** a domain entity.
- IDs are generated in the **service**, never by the database.
- Service returns **sentinel errors** (`user.ErrEmailTaken`), never `*apperrors.AppError`.

### Ports

```go
// ✅ Split by capability — a reader-only consumer never implements Save
type UserReader interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
}

type UserWriter interface {
    Save(ctx context.Context, u *User) error
}

// ❌ One wide interface mixing reads and writes
type Repository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    Save(ctx context.Context, u *User) error
    Update(ctx context.Context, u *User) error
    Delete(ctx context.Context, id string) error
}

// ❌ Generic "god interface" — hard to optimize, hard to extend
type Repository interface {
    Find(ctx context.Context, cond map[string]any) ([]*User, error)
}
```

- Define the port where it is **consumed**, not where it is implemented.
- A missing row returns `nil, ErrNotFound` — never `nil, nil`.
- Never discard errors with `_`. Check `ErrNotFound` explicitly.
- **Only declare methods you actually call.** An unused method is a promise every
  implementor and every fake has to keep.

### Errors

```go
// internal/user/errors.go — the domain knows nothing about HTTP
var (
    ErrNotFound   = errors.New("user not found")
    ErrEmailTaken = errors.New("email already registered")
}
```

```go
// internal/user/http/errors.go — ONE mapping function per feature
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

- Domain → sentinel error. Feature `http/` → `pkg/errors.AppError` → `pkg/response`.
- The sentinel → `AppError` mapping lives in **one function per feature**
  (`internal/<feature>/http/errors.go`), not in a global registry and not scattered
  across every handler method.
- `pkg/errors` is **always** imported as `apperrors` — it collides with stdlib `errors`.
- Internal failures map to `500`, are logged with the full cause, and never leak
  internals to the client.

### Mappers

- Pure functions only. No side effects, no I/O.
- One file per feature: `adapter/mapper.go`. One function per direction:
  `toDomain`, `toRecord`.
- Never embed GORM tags in domain structs.
- **A mapper that can fail returns an error.** The domain holds a UUID as `string`
  and the record holds `uuid.UUID`, so the mapper parses — and must not swallow a
  parse failure.

### Handlers

```go
func (h *Handler) Create(c *gin.Context) {
    var req user.CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, err) // uses the custom validator's user-friendly message
        return
    }

    cmd := user.CreateUserCommand{Email: req.Email, Password: req.Password}

    u, err := h.svc.Create(c.Request.Context(), cmd)
    if err != nil {
        response.FromError(c, toAppError(err))
        return
    }

    response.Created(c, toUserResponse(u))
}
```

- Parse → validate → map → call service → map → respond. No business logic.
- Handlers are thin. If a handler grows past ~20 lines, logic belongs in the service.
- Never put a `Role` (or any other privilege field) on a request DTO. Roles are
  assigned server-side.

### Wiring

- **All** dependency wiring happens in `cmd/api/main.go`. Just constructor calls.
- If a feature needs a new dependency, add it there. Never instantiate an adapter
  inside a service.

---

## 🧪 Testing Rules

| Layer | Test kind | Infra |
|---|---|---|
| Domain / Service | Unit, hand-written fakes | none |
| Mapper | Unit (pure functions) | none |
| Handler | Unit, fake service + `httptest` | none |
| Adapter | Integration, Testcontainers | Docker |
| Full flow | E2E, Testcontainers | Docker |

- **Every** code change ships with a test.
- Unit tests are **fast** — no network, no DB, no sleeps. They must pass in
  under a second per package.
- Integration tests carry `//go:build integration` and run via `make test-integration`.
- Prefer **fakes** over generated mocks — a fake implements the port and is
  easier to read than a mock's call assertions.
- Fakes live in `_test.go` files in the **inner** package, next to the port they
  satisfy, so production code cannot import them.
- Test the *contract*, not the implementation: assert on returned domain values
  and sentinel errors, not on internal call sequences.
- Every feature gets a **table-driven test for `toAppError`** — each sentinel must
  map to the intended status code.
- Maps and slices in fakes are always initialized explicitly — never left `nil`.

```go
// ✅ Fake implements the port, lives beside it
// internal/user/fake_user_reader_test.go
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

## 🛠️ Commands

| Command | What it does |
|---|---|
| `make run` | Start the API locally |
| `make test` | Unit tests only (fast, no infra) |
| `make test-integration` | Integration tests (needs Docker) |
| `make lint` | `gofmt` + `go vet` + `golangci-lint` (incl. import-boundary rules) |
| `make seed` | Run the idempotent seeder |
| `make migrate-up` / `make migrate-down` | Apply / roll back migrations |
| `make swagger` | Regenerate `swagger/docs.go` + `swagger/swagger.json` (pinned `swag` version) |
| `make cover` | Coverage report → `coverage.out` |

**Before declaring a task done**, run:

```bash
make lint && make test && make test-integration
```

---

## 🚦 Quality Gates

1. All tests green.
2. `make lint` clean — including the `depguard` import-boundary check. The inner
   package `internal/<feature>` must import no `gin`, no `gorm.io/gorm`, no `net/http`.
3. Coverage: starts at 40% overall / 60% for the service layer, and is raised
   gradually — never lowered to make CI green.
4. Every new sentinel error has a `toAppError` mapping **and** a table-driven test.
5. Manual review required for DB schema & auth changes.
6. Deploy is a **human gate** — CI is automated, merge/deploy is not.

---

## 📌 Things That Are Easy to Get Wrong

| Trap | Correct approach |
|---|---|
| `handler.go` sitting next to `service.go` in one package | Split it out — `internal/<feature>/http/`. Linters enforce per package, so a shared package makes domain purity unenforceable |
| Service returns `*apperrors.AppError` with a status code | Service returns a sentinel error; the feature's `http/errors.go` maps it |
| A "central registry" mapping every sentinel to a status | One `toAppError` per feature — a global registry would import every feature's domain and invert the dependency direction |
| `Role` on a request DTO | Roles are assigned server-side; the register DTO has no `Role` field |
| Repository returns `nil, nil` for a missing row | Return `nil, ErrNotFound` |
| `existing, _ := repo.FindByEmail(...)` | Check `ErrNotFound` explicitly, don't swallow |
| `gorm:"..."` tags on a domain struct | Separate `record.go` + `mapper.go` |
| Mapper converting `string` ⇄ `uuid.UUID` without an error return | Parsing can fail — the mapper returns the error, never swallows it |
| A port method with no caller | Delete it; every unused method is a promise every fake must keep |
| One wide `Repository` interface mixing reads and writes | Split into `UserReader` / `UserWriter` |
| A `Logger` port defined per feature | Use `*zap.Logger` directly — Zap is a pure utility |
| `pkg/errors` imported unaliased next to stdlib `errors` | Always alias it as `apperrors` |
| `adapter` packages imported without an alias | Alias as `<feature>adapter` / `<feature>http` — all three collide |
| Handler builds the URL path | Paths live in `internal/router/` |
| `AutoMigrate` used for schema changes | Versioned SQL in `migrations/` |
| Adapter imported inside a service | Wire it in `cmd/api/main.go` |
| Mock framework for service tests | Hand-written fake implementing the port |
| A nil map in a fake | Initialize it explicitly — a nil map panics on write |
| ID generated by the DB (`gen_random_uuid()`) | Service calls `pkg/id.NewV7()` |
| Generic `Find(ctx, cond map[string]any)` | Explicit `FindByX` methods |

---

## 📚 Reference

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — **source of truth** for structure
- [`docs/PLAN.md`](docs/PLAN.md) — source of truth for order; check off items as you go
- `swagger/swagger.json` — generated endpoint reference (run `make swagger` to refresh)

Both docs are **living documents**. If your work changes the structure or the
plan, update them in the same commit.
