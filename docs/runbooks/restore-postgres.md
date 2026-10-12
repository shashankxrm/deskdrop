# Runbook: restore Postgres from Object Storage (D6)

Live queue is k3s Postgres. Backups are gzip SQL in bucket `deskdrop-backups` (`pg/deskdrop-<UTC>.sql.gz`). The **drill** restores into a throwaway database `deskdrop_restore` and drops it. A real disaster restore overwrites `deskdrop` and is a different, destructive path.

## Drill (safe)

On the VM, after D6 env keys exist:

```bash
cd ~/deskdrop
./scripts/pg-restore-drill.sh
```

Expect `drill_ok` and equal `live_users` / `restore_users`. HTTPS stays up.

## Drill log

| When (UTC) | users | object key | notes |
|---|---|---|---|
| 2026-10-12T09:28:35Z | 6 | pg/deskdrop-20261012T092831Z.sql.gz | sidecar DB deskdrop_restore dropped |

## Disaster (destructive)

Stop writes, replace the live database, restart API. Only if the live volume is gone or corrupt.

```bash
# 1. Pick KEY from: aws s3 ls s3://deskdrop-backups/pg/ --endpoint-url …
# 2. Download and restore into deskdrop (this drops existing objects in the dump):
aws s3 cp "s3://deskdrop-backups/${KEY}" /tmp/deskdrop.sql.gz --endpoint-url "$ENDPOINT"
sudo k3s kubectl -n deskdrop scale deploy/api --replicas=0
gzip -dc /tmp/deskdrop.sql.gz | sudo k3s kubectl -n deskdrop exec -i deploy/postgres -- \
  psql -U deskdrop -d deskdrop -v ON_ERROR_STOP=1
sudo k3s kubectl -n deskdrop scale deploy/api --replicas=2
curl -sfS https://shashank-dropdesk.duckdns.org/readyz
```

Pairing tokens are HMAC of the pepper still in Secret `deskdrop`. Restoring an old dump with a **new** pepper means every device must pair again.

## Verify

```bash
./scripts/pg-restore-drill.sh
curl -sfS https://shashank-dropdesk.duckdns.org/healthz
```
