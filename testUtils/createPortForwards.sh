#!/bin/bash

# Grafana
kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80 &

# Go API
kubectl port-forward svc/go-api 8080:8080 &

# Prometheus
kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090 &

#jagure
kubectl port-forward svc/jaeger 16686:16686

# Wait so script doesn’t exit
wait
