#!/usr/bin/env bash

# Port-forward commands collected from this workspace.
# Usage:
#   ./all-port-forward-commands.sh go-gin-server
#   ./all-port-forward-commands.sh promethius

set -e

case "${1:-}" in
  go-gin-server)
    # From go-gin-server/commands
    kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80 &
    kubectl port-forward svc/go-api 8080:8080 &
    kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090 &
    ;;
  promethius)
    # From promethius/testUtils/createPortForwards.sh
    kubectl port-forward svc/prometheus-grafana 3000:80 &
    kubectl port-forward svc/go-api-service 8080:80 &
    kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 &
    kubectl port-forward svc/jaeger 16686:16686 &
    ;;
  *)
    echo "Usage: $0 {go-gin-server|promethius}"
    exit 1
    ;;
esac

wait

