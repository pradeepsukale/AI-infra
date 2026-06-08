#!/bin/bash

# Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 &

# Go API
kubectl port-forward svc/go-api-service 8080:80 &

# Prometheus
kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 &

#jagure
kubectl port-forward svc/jaeger 16686:16686

# Wait so script doesn’t exit
wait
