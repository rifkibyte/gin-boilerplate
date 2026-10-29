# Gin Boilerplate

A production-grade Gin + PostgreSQL + GORM boilerplate built on **Hexagonal Architecture
(Ports & Adapters)**. Designed for learning, with automated tests from day one.

## Tech Stack

| Concern            | Choice                                               |
| ------------------ | ---------------------------------------------------- |
| HTTP framework     | Gin                                                  |
| Database           | PostgreSQL + GORM                                    |
| Migrations         | golang-migrate (raw SQL)                             |
| Cache / rate limit | Redis                                                |
| Logging            | Zap (JSON)                                           |
| IDs                | UUID v7                                              |
| Auth               | JWT (HS256) + bcrypt                                 |
| Docs               | Swagger (swag)                                       |
| Testing            | testing, httptest, Testcontainers                    |
| Lint               | golangci-lint (incl. depguard import-boundary rules) |
| Container          | Docker + Docker Compose                              |

## Quick Start

### Prerequisites

- Go 1.26+
- Docker + Docker Compose
- `golangci-lint` (optional, for `make lint`)
- `golang-migrate` (optional, for `make migrate-up`)

### 1. Clone and configure

```bash
git clone https://github.com/rifkibyte/gin-boilerplate.git
cd gin-boilerplate
cp .env.example .env   # if present; otherwise set env vars manually
```

### 2. Start infrastructure

```bash
docker compose -f deploy/docker-compose.yml up -d postgres redis
```

Wait for both services to become healthy:

```bash
docker compose -f deploy/docker-compose.yml ps
```

### 3. Run the application

```bash
# With env vars inline
JWT_SECRET="change-me-in-production-at-least-32-chars-long" make run

# Or export them first
export JWT_SECRET="change-me-in-production-at-least-32-chars-long"
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=gin_boilerplate
export REDIS_ADDR=localhost:6379
make run
```

The server starts on `http://localhost:8080`.

### 4. Verify

```bash
curl http://localhost:8080/api/v1/health
# {"status":"ok"}
```

### 5. Full stack with Docker Compose

```bash
docker compose -f deploy/docker-compose.yml up --build
```

This builds the app image and starts everything (app + PostgreSQL + Redis).

## Environment Variables

| Variable                | Default               | Description                             |
| ----------------------- | --------------------- | --------------------------------------- |
| `HTTP_ADDR`             | `:8080`               | Listen address                          |
| `HTTP_READ_TIMEOUT`     | `5s`                  | Read timeout                            |
| `HTTP_WRITE_TIMEOUT`    | `10s`                 | Write timeout                           |
| `HTTP_IDLE_TIMEOUT`     | `60s`                 | Idle timeout                            |
| `DB_HOST`               | `localhost`           | PostgreSQL host                         |
| `DB_PORT`               | `5432`                | PostgreSQL port                         |
| `DB_USER`               | `postgres`            | PostgreSQL user                         |
| `DB_PASSWORD`           | `postgres`            | PostgreSQL password                     |
| `DB_NAME`               | `gin_boilerplate`     | Database name                           |
| `DB_SSLMODE`            | `disable`             | SSL mode                                |
| `DB_MAX_OPEN_CONNS`     | `25`                  | Max open connections                    |
| `DB_MAX_IDLE_CONNS`     | `10`                  | Max idle connections                    |
| `DB_CONN_MAX_LIFETIME`  | `5m`                  | Connection max lifetime                 |
| `DB_CONN_MAX_IDLE_TIME` | `1m`                  | Connection max idle time                |
| `REDIS_ADDR`            | `localhost:6379`      | Redis address                           |
| `REDIS_PASSWORD`        | (empty)               | Redis password                          |
| `REDIS_DB`              | `0`                   | Redis DB index                          |
| `LOG_LEVEL`             | `info`                | Log level (debug/info/warn/error/fatal) |
| `LOG_FORMAT`            | `json`                | Log format (json/console)               |
| `LOG_SERVICE`           | `gin-boilerplate`     | Service name in logs                    |
| `JWT_SECRET`            | **required**          | JWT signing secret (min 32 chars)       |
| `JWT_ACCESS_TTL`        | `15m`                 | Access token TTL                        |
| `JWT_REFRESH_TTL`       | `168h`                | Refresh token TTL                       |
| `MAIL_HOST`             | `localhost`           | SMTP host                               |
| `MAIL_PORT`             | `1025`                | SMTP port                               |
| `MAIL_USERNAME`         | (empty)               | SMTP username                           |
| `MAIL_PASSWORD`         | (empty)               | SMTP password                           |
| `MAIL_FROM`             | `noreply@example.com` | From address                            |
| `STORAGE_DRIVER`        | `local`               | Storage driver (local/s3)               |
| `STORAGE_LOCAL_PATH`    | `./uploads`           | Local storage path                      |
| `S3_BUCKET`             | (empty)               | S3 bucket name                          |
| `S3_REGION`             | `us-east-1`           | S3 region                               |
| `S3_ENDPOINT`           | (empty)               | S3 endpoint (for MinIO)                 |
| `S3_ACCESS_KEY`         | (empty)               | S3 access key                           |
| `S3_SECRET_KEY`         | (empty)               | S3 secret key                           |

## Makefile Targets

| Target                  | Description                          |
| ----------------------- | ------------------------------------ |
| `make run`              | Start the API locally                |
| `make test`             | Run unit tests (fast, no infra)      |
| `make test-integration` | Run integration tests (needs Docker) |
| `make lint`             | Run gofmt + go vet + golangci-lint   |
| `make seed`             | Run the idempotent seeder            |
| `make migrate-up`       | Apply database migrations            |
| `make migrate-down`     | Roll back database migrations        |
| `make swagger`          | Regenerate Swagger docs              |
| `make cover`            | Generate coverage report             |

## Project Structure

```
.
├── cmd/
│   ├── api/            # Application entry point
│   └── seed/           # Database seeder
├── internal/
│   ├── config/         # Environment configuration
│   ├── middleware/     # Gin middleware (request ID, CORS, auth, rate limit)
│   ├── router/         # Route registration
│   └── <feature>/      # Feature packages (user, auth, file)
│       ├── (inner)     # domain, dto, port, errors, service
│       ├── http/       # HTTP adapter (Gin handler)
│       └── adapter/    # Infrastructure adapters (GORM, SMTP, S3)
├── pkg/
│   ├── cache/          # Redis client wrapper
│   ├── database/       # GORM + PostgreSQL
│   ├── errors/         # AppError type
│   ├── id/             # UUID v7 generation
│   ├── logger/         # Zap wrapper
│   ├── mailer/         # Mailer port + fake
│   └── response/       # JSON response helpers
├── migrations/         # SQL migration files
├── swagger/            # Generated Swagger docs
├── docs/               # Architecture & plan docs
├── deploy/             # Dockerfile + docker-compose
└── testutil/           # Testcontainers helpers
```

## Architecture

This project uses **Hexagonal Architecture (Ports & Adapters)**. Each feature is split
into three packages:

1. **Inner package** (`internal/<feature>/`) — pure domain: entities, DTOs, ports, errors,
   service. Never imports `gin`, `gorm`, or `net/http`. Enforced by `depguard`.
2. **HTTP adapter** (`internal/<feature>/http/`) — Gin handlers + sentinel→AppError mapping.
3. **Infrastructure adapter** (`internal/<feature>/adapter/`) — GORM, SMTP, S3, Redis.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full design.

## Development

### Running tests

```bash
# Unit tests (fast, no infra)
make test

# Integration tests (requires Docker)
make test-integration

# With coverage
make cover
```

### Linting

```bash
make lint
```

### Generating Swagger docs

```bash
make swagger
```

Swagger UI is available at `http://localhost:8080/swagger/index.html`.

## License

MIT
