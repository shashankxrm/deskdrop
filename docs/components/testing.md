# Testing

How to run each suite. This file grows with B/C steps. No testify. No mocked Postgres.

## Schema (A2)

```bash
docker compose up -d
docker compose exec -T postgres \
  psql -U deskdrop -d deskdrop -v ON_ERROR_STOP=1 \
  < tests/schema/fk_delete.sql
```

## Go (B1)

Standard library `testing`, table-driven config tests, `httptest` for handlers, testcontainers-go for `/readyz` against real Postgres 16.

```bash
go test ./...
```

Docker must be running for `TestReadyz_ok`.

CI job **go-test** (B1b) runs `go vet ./...` and `go test ./...` on every push/PR to `v2`. See [ci-cd.md](ci-cd.md).

`github.com/stretchr/testify` may appear as an **indirect** module of testcontainers-go. Our tests must not import it.

Later steps add pairing/delivery tests in the same `go test ./...` invocation.
