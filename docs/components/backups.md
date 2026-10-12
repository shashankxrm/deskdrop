# Backups (D6)

Nightly `pg_dump` of k3s Postgres to OCI Object Storage bucket **`deskdrop-backups`** (Always Free, same tenancy as Terraform state). Dump is taken with `kubectl exec` so it does not need a NetworkPolicy hole to Postgres.

Scripts: [`scripts/pg-backup.sh`](../../scripts/pg-backup.sh), [`scripts/pg-restore-drill.sh`](../../scripts/pg-restore-drill.sh). Restore steps: [restore-postgres.md](../runbooks/restore-postgres.md).

## One-time on the VM

1. Console → Identity → your user → **Customer secret keys** (S3-compat, not the API PEM). Put access + secret in `deploy/prod/.env`:

```bash
OCI_S3_ACCESS_KEY=
OCI_S3_SECRET_KEY=
OCI_S3_NAMESPACE=   # oci os ns get
OCI_S3_REGION=us-phoenix-1
OCI_S3_BUCKET=deskdrop-backups
```

`chmod 600 deploy/prod/.env`

2. `sudo apt-get install -y awscli`

3. Cron (02:00 UTC):

```bash
sudo tee /etc/cron.d/deskdrop-backup >/dev/null <<'EOF'
0 2 * * * ubuntu /home/ubuntu/deskdrop/scripts/pg-backup.sh >> /home/ubuntu/deskdrop-backup.log 2>&1
EOF
sudo chmod 644 /etc/cron.d/deskdrop-backup
```

## Verify or demo

```bash
ssh -i ~/Downloads/ssh-key-2026-10-10.key ubuntu@137.131.14.30
cd ~/deskdrop
./scripts/pg-restore-drill.sh
# drill_ok <UTC timestamp>  live_users must equal restore_users
aws s3 ls s3://deskdrop-backups/pg/ \
  --endpoint-url "https://${OCI_S3_NAMESPACE}.compat.objectstorage.us-phoenix-1.oraclecloud.com"
```

Paste the drill line into [restore-postgres.md](../runbooks/restore-postgres.md). Do not restore over the live `deskdrop` database in the drill.

## Failure modes

- Empty `OCI_S3_*`: script exits before dump upload.
- `aws: command not found`: install `awscli`.
- 403: wrong customer secret key, namespace, or bucket region.
- Cron user must be `ubuntu` with passwordless `sudo k3s kubectl`.
- NetworkPolicy is unused here; a future in-cluster CronJob would need an allow rule to Postgres.

## OCI S3 checksum setting (required)

AWS CLI v2.23+ sends chunked checksums that OCI's S3-compat endpoint rejects ("AWS chunked encoding not supported"). On the VM, `~/.aws/config` must contain:

```ini
[default]
request_checksum_calculation = when_required
response_checksum_validation = when_required
```

Recreate this on any rebuilt VM before running `pg-backup.sh` or the cron.
