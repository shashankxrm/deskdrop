# Grafana (D0)

v2 observability is slog + Prometheus `/metrics` on the live API. Grafana Cloud **free** stores the series. Do not run Grafana on the 6 GB VM.

## Panels (one dashboard)

Import [deploy/grafana/deskdrop.json](../../deploy/grafana/deskdrop.json):

| Panel | PromQL |
|---|---|
| Shares | `deskdrop_shares_total` |
| Acks | `deskdrop_acks_total` |
| Ack rate | `deskdrop_acks_total / clamp_min(deskdrop_shares_total, 1)` |
| Latency p95 | `histogram_quantile(0.95, sum by (le) (rate(deskdrop_http_request_duration_seconds_bucket[5m])))` |

## Wire-up

1. Sign up at [Grafana Cloud](https://grafana.com/products/cloud/) (free).
2. Connections → add **Alloy** / Prometheus remote_write. Copy:
   - remote_write URL (`…/api/prom/push`)
   - numeric username
   - access policy token with `metrics:write`
3. Put those in `deploy/prod/.env` as `GRAFANA_CLOUD_RW_URL`, `GRAFANA_CLOUD_USER`, `GRAFANA_CLOUD_TOKEN`.
4. On the VM: `docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env --profile metrics up -d`
5. Grafana Explore: `deskdrop_shares_total` should appear after a share.

Public scrape (optional, basic auth):

```bash
curl -u scrape:$METRICS_PASSWORD https://YOUR.duckdns.org/metrics | grep deskdrop_shares_total
```

Without credentials that URL is 401.

## Verify

Share a link from Android, ack on the Mac, then the Shares and Acks stats increment. p95 is the HTTP histogram, not a synthetic SLO (Post-v2).
