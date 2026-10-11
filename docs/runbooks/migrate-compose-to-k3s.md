# Runbook: Compose → k3s (D4)

Same Ampere VM. Same DuckDNS name. Postgres dump/restore so pairing rows survive. Do not `terraform destroy`. Do not `compose down -v`.

## Before

```bash
ssh -i ~/Downloads/ssh-key-2026-10-10.key ubuntu@137.131.14.30
cd ~/deskdrop
git pull origin v2
curl -sfS https://shashank-dropdesk.duckdns.org/healthz
docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env exec -T postgres \
  psql -U deskdrop -d deskdrop -c 'SELECT count(*) FROM users;'
```

`deploy/prod/.env` must include `METRICS_PASSWORD` (plaintext; Traefik basic auth). `chmod 600`.

## Cut

```bash
./scripts/k3s-cutover.sh
```

The script: counts users → installs k3s (Traefik off) + cert-manager → creates namespace/secrets → stops Compose API/Caddy → `pg_dump` → k3s Postgres restore → migrate + API×2 + dashboard → checks user count → `compose down` → enables Traefik → Ingress + Let’s Encrypt → `curl https://…/healthz`.

HTTPS is down from `compose stop` until Traefik has a cert (a few minutes). Agents keep polling; pending rows stay in Postgres.

## After

```bash
sudo k3s kubectl -n deskdrop get pods
sudo k3s kubectl -n deskdrop get certificate deskdrop-tls
curl -sfS https://shashank-dropdesk.duckdns.org/readyz
# Share a YouTube link from the phone; the Mac agent should still open it.
```

Grafana: if `GRAFANA_CLOUD_TOKEN` is in `.env`, the script applies [`deploy/k8s/alloy.yaml`](../../deploy/k8s/alloy.yaml). Panels stay the same scrape of `deskdrop_*`.

Compose volumes remain on disk until you choose to `docker volume rm`. Do not remove them until a share has delivered on k3s.

## Rollback (Compose still has pgdata)

```bash
sudo systemctl stop k3s
# free 80/443
cd ~/deskdrop
docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env up -d
curl -sfS https://shashank-dropdesk.duckdns.org/healthz
```

That returns to the **pre-cut** Postgres volume (not the k3s PVC). Only use it if cutover failed before you deleted volumes.

## CI after cutover

Merge to `v2` rsyncs the repo and, if `k3s` is active, `kubectl apply -k deploy/k8s` plus Ingress (no Compose). Secrets on the cluster are not replaced by CI.
