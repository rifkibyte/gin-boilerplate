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
- [ ] `make lint` passes.
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
feature-first. If the architecture doc and this file disagree, the architecture
doc wins.

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

---

## 🔒 Locked Decisions

**Do not re-litigate these. Do not change them without being asked.**

| Decision | Choice | Rationale |
|---|---|---|
| Architecture | Hexagonal (Ports & Adapters), feature-based packages | Business logic testable without infra; swappable adapters |
| Layout | `internal/<feature>/{domain,dto,port,errors,service,handler}` + `internal/<feature>/adapter/` | One feature = one package; adapters quarantined |
| Routing | `internal/router/` | All URL paths in one greppable place |
| IDs | **UUID v7**, generated in the service via `pkg/id` | Time-ordered (dense indexes), no DB extension, known before INSERT |
| Errors | Feature sentinel errors → `pkg/errors.AppError` → `pkg/response` | Domain never speaks HTTP; one central status mapping |
| Migrations | Raw SQL via `golang-migrate`; AutoMigrate only for local iteration | Versioned, reviewable, reversible |
| Tests | Fakes for ports, Testcontainers for adapters | The hexagon's seam is the port; that's where fakes go |
| Swagger | `swag` v1 with `@Router` annotations on handlers | Annotations live next to the code they describe |
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
│   ├── <feature>/                  # One package per feature
│   │   ├── domain.go               # Plain structs — no gorm/json tags
│   │   ├── dto.go                  # json + binding tags
│   │   ├── port.go                 # Interfaces the service consumes
│   │   ├── errors.go               # Feature-scoped sentinel errors
│   │   ├── service.go              # Business logic — ports only
│   │   ├── handler.go              # Gin adapter
│   │   └── adapter/                # GORM / SMTP / S3 / Redis implementations
│   │       ├── record.go           # GORM model
│   │       ├── mapper.go           # record ↔ domain
│   │       └── repository_gorm.go
│   ├── middleware/                 # Request ID, CORS, auth, rate limit
│   └── router/                     # ALL URL paths live here
├── pkg/
│   ├── logger/  database/  cache/  errors/  response/  id/
├── migrations/                     # Raw SQL, up + down
├── docs/                           # ARCHITECTURE.md, PLAN.md, generated Swagger
├── testutil/                       # Testcontainers helpers
└── Makefile
```

---

## 🚫 Import Rules

**The mechanical rules that keep the hexagon intact.**

| Package | May import | Must NOT import |
|---|---|---|
| `internal/<feature>` (domain/dto/port/service) | stdlib, `pkg/errors`, `pkg/id`, `pkg/logger` | `gin`, `gorm.io/gorm`, `net/http`, `internal/<feature>/adapter` |
| `internal/<feature>/adapter` | `gorm.io/gorm`, `internal/<feature>` | `gin`, `net/http` |
| `internal/<feature>` (handler) | `gin`, `internal/<feature>`, `pkg/response` | `gorm.io/gorm` |
| `internal/router` | `gin`, `internal/<feature>` | `gorm.io/gorm` |
| `cmd/api` | everything | — |

Additional rules:

- A feature package **must never** import another feature's `adapter/`.
- Cross-feature collaboration goes through ports defined in the **consuming** feature.
- The domain layer **never** imports `net/http` and never constructs status codes.
- Handlers contain **no URL path strings** — those live in `internal/router/`.
- Handlers **never** touch a `*Record` (GORM model) directly.

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
// ✅ Small, explicit, one responsibility per method
type Repository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    Save(ctx context.Context, u *User) error
}

// ❌ Generic "god interface" — hard to optimize, hard to extend
type Repository interface {
    Find(ctx context.Context, cond map[string]any) ([]*User, error)
}
```

- Define the port where it is **consumed**, not where it is implemented.
- A missing row returns `nil, ErrNotFound` — never `nil, nil`.
- Never discard errors with `_`. Check `ErrNotFound` explicitly.

### Errors

```go
// internal/user/errors.go
var (
    ErrNotFound   = errors.New("user not found")
    ErrEmailTaken = errors.New("email already registered")
)
```

- Domain → sentinel error. Transport → `pkg/errors.AppError` → `pkg/response`.
- The sentinel → `AppError` mapping lives in **one** registry. Do not scatter
  `errors.Is` checks across every handler.
- Internal failures map to `500`, are logged with the full cause, and never leak
  internals to the client.

### Mappers

- Pure functions only. No side effects, no I/O.
- One file per feature: `adapter/mapper.go`. One function per direction:
  `toDomain`, `toRecord`.
- Never embed GORM tags in domain structs.

### Handlers

```go
func (h *Handler) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, err.Error())
        return
    }

    u, err := h.svc.Create(c.Request.Context(), req.Email, req.Password, req.Role)
    if err != nil {
        response.FromError(c, err)
        return
    }

    response.Created(c, toUserResponse(u))
}
```

- Parse → validate → map → call service → map → respond. No business logic.
- Handlers are thin. If a handler grows past ~20 lines, logic belongs in the service.

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
- Fakes live in `_test.go` files next to the port they satisfy, so production
  code cannot import them.
- Test the *contract*, not the implementation: assert on returned domain values
  and sentinel errors, not on internal call sequences.

```go
// ✅ Fake implements the port, lives beside it
// internal/user/fake_repository_test.go
type fakeRepository struct{ users map[string]*User }

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

## 🛠️ Commands

| Command | What it does |
|---|---|
| `make run` | Start the API locally |
| `make test` | Unit tests only (fast, no infra) |
| `make test-integration` | Integration tests (needs Docker) |
| `make lint` | `gofmt` + `go vet` + `golangci-lint` (incl. import-boundary rules) |
| `make seed` | Run the idempotent seeder |
| `make migrate-up` / `make migrate-down` | Apply / roll back migrations |
| `make swagger` | Regenerate `docs/docs.go` + `swagger.json` |
| `make cover` | Coverage report → `coverage.out` |

**Before declaring a task done**, run:

```bash
make lint && make test && make test-integration
```

---

## 🚦 Quality Gates

1. All tests green.
2. `make lint` clean — including the import-boundary check.
3. Coverage: 60% for service layer, 40% overall (raise gradually).
4. Manual review required for DB schema & auth changes.
5. Deploy is a **human gate** — CI is automated, merge/deploy is not.

---

## 📌 Things That Are Easy to Get Wrong

| Trap | Correct approach |
|---|---|
| Service returns `*apperrors.AppError` with a status code | Service returns a sentinel error; the transport layer maps it |
| Repository returns `nil, nil` for a missing row | Return `nil, ErrNotFound` |
| `existing, _ := repo.FindByEmail(...)` | Check `ErrNotFound` explicitly, don't swallow |
| `gorm:"..."` tags on a domain struct | Separate `record.go` + `mapper.go` |
| Handler builds the URL path | Paths live in `internal/router/` |
| `AutoMigrate` used for schema changes | Versioned SQL in `migrations/` |
| Adapter imported inside a service | Wire it in `cmd/api/main.go` |
| Mock framework for service tests | Hand-written fake implementing the port |
| ID generated by the DB (`gen_random_uuid()`) | Service calls `pkg/id.NewV7()` |
| Generic `Find(ctx, cond map[string]any)` | Explicit `FindByX` methods |

---

## 📚 Reference

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — **source of truth** for structure
- [`docs/PLAN.md`](docs/PLAN.md) — source of truth for order; check off items as you go
- `docs/swagger.json` — generated endpoint reference (run `make swagger` to refresh)

Both docs are **living documents**. If your work changes the structure or the
plan, update them in the same commit.
