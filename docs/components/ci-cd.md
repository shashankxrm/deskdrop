# CI/CD

B1b is test-only. Every push and pull request to `v2` runs Go `vet` and `go test ./...` on GitHub Actions with Docker (so testcontainers can start Postgres). No deploy, no GHCR, no client builds yet — those are D1/D2.

## Why this way

- Catch a red `go test` before pairing (B2) lands on the branch.
- Same command as local: `go test ./...`. Docker is already on `ubuntu-latest` runners; no extra Compose in CI yet (E2E compose script is B7/D1).
- `permissions: contents: read` — this workflow does not need to push or deploy. Fork PRs can still run tests; they cannot deploy (there is no deploy job).

## Alternatives

- A Postgres service container in Actions instead of testcontainers: two ways to boot Postgres. testcontainers match local `go test` and apply `000001_init.up.sql` in-process.
- Skipping Docker in CI: `/readyz` would be untested on PRs.

## Failure modes

- Job red: `go vet` or `go test` failed. Read the `vet` / `test` step logs.
- testcontainers cannot start: runner Docker is broken (rare). Re-run the job.
- `go-version-file: go.mod` must match a toolchain Actions can install (`go 1.26` today).

## Verify or demo

Workflow file: [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)

Required check name: **go-test** (job `go-test`).

```bash
# Same as CI, locally:
go vet ./...
go test ./...

# After this file is on origin/v2, open:
#   https://github.com/shashankxrm/deskdrop/actions
# Confirm the latest run on branch v2 is green.
# A PR that breaks a test must show go-test as failed.
```

D1 will add Next, Gradle, Playwright, compose E2E, and deploy-on-merge to this doc.
