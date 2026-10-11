# Runbook: Compose + Caddy on the Oracle VM (D0)

Closed beta. Same origin: `/` dashboard, `/v1` API, `/healthz` `/readyz`, `/metrics` (basic auth). Postgres is not on a public port.

## 0. One-time on your laptop

1. DuckDNS: create a subdomain (free), set the A record to **137.131.14.30**.
2. OCI console: NSG/security list **ingress TCP 80 and 443** from `0.0.0.0/0` (SSH 22 can stay your IP).
3. Grafana Cloud free: copy Prometheus **remote_write** URL, instance id (username), and API token (see [grafana.md](../components/grafana.md)).

## 1. VM: Docker + ports

```bash
ssh -i ~/.ssh/oci-deskdrop ubuntu@137.131.14.30   # or ~/Downloads/ssh-key-2026-10-10.key

sudo apt-get update
sudo apt-get install -y ca-certificates curl git
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker ubuntu
# re-login so docker works without sudo

# Ubuntu on OCI often filters 80/443 in iptables even when the NSG is open:
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 80 -j ACCEPT
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 443 -j ACCEPT
sudo apt-get install -y iptables-persistent || true
sudo netfilter-persistent save || sudo iptables-save | sudo tee /etc/iptables/rules.v4
```

## 2. Clone and secrets

```bash
git clone -b v2 https://github.com/shashankxrm/deskdrop.git
cd deskdrop
cp deploy/prod/env.example deploy/prod/.env
nano deploy/prod/.env
```

Fill:

- `DESKDROP_HOST` — the DuckDNS FQDN (no `https://`)
- `CADDY_EMAIL` — Let’s Encrypt notices
- `POSTGRES_PASSWORD` and `DEVICE_TOKEN_PEPPER` — `openssl rand -hex 32` each, **not** `compose-local-pepper`
- `METRICS_HASH` — from `docker run --rm caddy:2-alpine caddy hash-password --plaintext '…'`
- Grafana remote_write vars if using `--profile metrics`

`chmod 600 deploy/prod/.env`

## 3. Up

```bash
cd ~/deskdrop
docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env up --build -d
docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env ps
curl -fsS https://$DESKDROP_HOST/healthz
# ok
curl -fsS https://$DESKDROP_HOST/readyz
# ok
```

Certs: first `up` needs DNS already pointing here. If Caddy logs `NXDOMAIN` or `unauthorized`, wait for DuckDNS and `docker compose ... restart caddy`.

Grafana scrape (optional):

```bash
docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env --profile metrics up -d
```

Idle reclaim: Oracle may reclaim Always Free VMs that stay under ~20% for 7 days.

```bash
# crontab -e on the VM (uses the public hostname so traffic hits the NIC):
*/30 * * * * curl -fsS https://YOUR.duckdns.org/healthz >/dev/null
```

## 4. Clients

- Dashboard: `https://YOUR.duckdns.org` — **Start a new pairing**, then Pair the phone (Forget first if needed), then **Add a phone or laptop** for the agent.
- Android API URL: `https://YOUR.duckdns.org`
- Agent: `/tmp/deskdrop-agent pair -api https://YOUR.duckdns.org -code ABCD-EFGH` then `run` / `install`
- `/metrics` without auth is 401. With auth: `curl -u scrape:PASSWORD https://YOUR.duckdns.org/metrics`

## 5. Releases

On the Mac (unsigned agent + optional APK):

```bash
./scripts/release-checksums.sh
cd dist && shasum -a 256 -c SHA256SUMS
gh release create v2-d0 dist/* --title 'v2 closed beta' --notes 'Unsigned macOS: Open Anyway. Android APK is debug.'
```

## Rollback

```bash
docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env down
# volumes pgdata / caddy_data kept unless you -v
```
