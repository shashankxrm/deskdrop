# CI/CD

Every push and pull request to `v2` runs Go, dashboard Vitest/build, Android unit tests, agent compile, compose E2E, and Playwright. Merge to `v2` deploys: **k3s `kubectl apply`** if the VM is running k3s, otherwise Compose. Forks cannot deploy (no secrets). No Windows `.exe`. k3d-in-CI is Post-v2 (A0 is Ampere).

## Why this way

- Same commands as local. Docker on `ubuntu-latest` covers testcontainers and Compose.
- Playwright talks to `npm run dev` with `API_BASE=http://127.0.0.1:8080` after compose is up.
- `permissions: contents: read`. Deploy uses SSH from repo secrets, not from pull_request secrets.
- Deploy rsync **excludes** `deploy/prod/.env` so VM secrets are not wiped.

## Jobs

| Job | What |
|---|---|
| `go-test` | `go vet ./...` and `go test ./...` |
| `dashboard` | `npm test` and `npm run build` in `dashboard/` |
| `android` | `./gradlew :app:testDebugUnitTest` |
| `agent` | `go build ./cmd/agent` |
| `e2e` | `docker compose up --build -d`, `tests/e2e/compose_api.sh` (includes agent), Playwright |
| `images` | distroless/non-root API + dashboard; `scripts/check-images.sh`; push `ghcr.io/<owner>/deskdrop/{api,dashboard}:v2` on push to `v2` |
| `deploy` | push to `v2` only; rsync; if k3s: `kubectl apply -k deploy/k8s` + Ingress + rollout; else Compose `up --build -d`; `curl /healthz` |

## Secrets (deploy)

Set on the GitHub repo (Settings → Secrets). Empty `DEPLOY_SSH_KEY` skips the rsync steps (job-level `if` cannot read `secrets`).

- `DEPLOY_HOST` — VM public IP or DuckDNS hostname
- `DEPLOY_USER` — `ubuntu`
- `DEPLOY_SSH_KEY` — private key that can SSH to that user (PEM)

## Alternatives

- Postgres service container instead of testcontainers: two ways to boot Postgres. testcontainers match local `go test`.
- k3s pulls `ghcr.io/shashankxrm/deskdrop/{api,dashboard}:v2`. Compose rollback still builds from Dockerfiles.

## Failure modes

- Any of `go-test` / `dashboard` / `android` / `agent` / `e2e` red blocks merge (required checks) and deploy.
- Playwright 403 in a sandbox: CI is a real Ubuntu VM; install Chromium with `--with-deps`.
- Playwright misses **Devices** after compose E2E: pairing create burst is 5/IP; E2E already used 4. Restart `api` before Playwright so the in-memory limiter resets.
- Android `Failed to find package 'tools'`: ubuntu-latest already has the SDK; do not install the obsolete `tools` package.
- Deploy skipped: the step sees empty `DEPLOY_*` env (GitHub forbids `secrets` in any `if`) and exits 0, or the event is a pull_request.
- GHCR package is private until you set it public (Package settings → Change visibility).
- API healthcheck must use exec-form `wget` (distroless has no shell).
- Deploy would delete VM `.env` if rsync `--delete` did not exclude it — the workflow excludes `deploy/prod/.env`.

## Verify or demo

Workflow file: [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)

Required check names: **go-test**, **dashboard**, **android**, **agent**, **e2e**, **images**.

```bash
go vet ./...
go test ./...
cd dashboard && npm test && npm run build
cd android && ./gradlew :app:testDebugUnitTest
go build -o /tmp/deskdrop-agent ./cmd/agent
docker compose up --build -d
tests/e2e/compose_api.sh
cd dashboard && npx playwright install chromium && npm run test:e2e
```

After this file is on `origin/v2`: [Actions](https://github.com/shashankxrm/deskdrop/actions). A PR that breaks a suite must show that job failed. After merge, `curl -sfS https://shashank-dropdesk.duckdns.org/healthz`.

## Images (D2)

Multi-stage **distroless** API (`static-debian12:nonroot`) and dashboard (`nodejs22-debian12:nonroot`). Busybox `wget` is copied into the API image only so Compose healthchecks work without a shell.

```bash
docker build -f deploy/compose/Dockerfile.api -t deskdrop-api:local .
docker build -f deploy/compose/Dockerfile.dashboard -t deskdrop-dashboard:local .
./scripts/check-images.sh deskdrop-api:local deskdrop-dashboard:local
docker inspect -f '{{.Config.User}}' deskdrop-api:local   # nonroot
docker pull ghcr.io/shashankxrm/deskdrop/api:v2
```

## Releases (D0)

```bash
./scripts/release-checksums.sh
cd dist && shasum -a 256 -c SHA256SUMS
gh release create v2-d0 dist/* --title 'v2 closed beta' --notes 'Unsigned Mac: Open Anyway.'
```
