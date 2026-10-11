#!/usr/bin/env bash
# On the VM after rsync. k3s apply if the service is up; else Compose.
set -euo pipefail
cd "$(dirname "$0")/.."
if systemctl is-active --quiet k3s; then
  sudo k3s kubectl delete job -n deskdrop migrate --ignore-not-found
  sudo k3s kubectl apply -k deploy/k8s
  sudo k3s kubectl apply -f deploy/k8s/ingress.yaml
  sudo k3s kubectl -n deskdrop rollout restart deploy/api deploy/dashboard
  sudo k3s kubectl -n deskdrop rollout status deploy/api --timeout=180s
  sudo k3s kubectl -n deskdrop rollout status deploy/dashboard --timeout=180s
else
  docker compose -f deploy/prod/compose.yaml --env-file deploy/prod/.env up --build -d
fi
