# Local Setup & Tooling

This document explains how to apply migrations, seed data, and use the local
database/email helper services (Adminer and Maildev).

## Migrations

Migrations live in `migrations/` and are applied with `golang-migrate`.

```bash
# Apply all up migrations
make migrate-up

# Roll back all migrations
make migrate-down
```

The database connection defaults to:

```
postgres://postgres:postgres@localhost:5432/gin_boilerplate?sslmode=disable
```

Override via the `migrate` CLI `-database` flag if needed.

## Seeding

The seeder is idempotent and safe to re-run. It creates one user per role:

```bash
make seed
```

Default seeded accounts:

| Email               | Password    | Role  |
| ------------------- | ----------- | ----- |
| admin@example.com   | admin123    | admin |
| user@example.com    | user12345   | user  |

Re-running `make seed` skips accounts that already exist.

## Docker Compose Helper Services

Start the full stack (app + PostgreSQL + Redis + Adminer + Maildev):

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Or start only the helpers:

```bash
docker compose -f deploy/docker-compose.yml up -d postgres redis adminer maildev
```

### Adminer (database GUI)

- URL: http://localhost:8081
- System: PostgreSQL
- Server: `postgres` (or `localhost` when running outside Compose)
- Username: `postgres`
- Password: `postgres`
- Database: `gin_boilerplate`

Use Adminer to inspect tables, run SQL, and verify seeded data.

### Maildev (email GUI)

Maildev captures emails sent via SMTP so you can inspect them without a real
mail server.

- Web UI: http://localhost:1080
- SMTP host: `localhost`
- SMTP port: `1025`

Point the app at Maildev by setting:

```bash
export MAIL_HOST=localhost
export MAIL_PORT=1025
```

Trigger an email (for example by registering a user) and then open
http://localhost:1080 to view the captured message.

## Swagger

Regenerate the Swagger docs after changing handler annotations:

```bash
make swagger
```

The generated files live in `swagger/` and the UI is served at:

```
http://localhost:8080/swagger/index.html
```

## Makefile Reference

| Target             | Description                                 |
| ------------------ | ------------------------------------------- |
| `make migrate-up`  | Apply database migrations                   |
| `make migrate-down`| Roll back database migrations                |
| `make seed`        | Seed default admin and user accounts        |
| `make run`         | Start the API locally                       |
| `make test`        | Run unit tests                              |
| `make test-integration` | Run integration tests (needs Docker)   |