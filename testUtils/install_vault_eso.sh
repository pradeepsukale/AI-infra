#!/bin/bash

# Exit on any error
set -e

echo "Installing HashiCorp Vault..."
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update hashicorp

# Install vault in dev mode for testing purposes
helm upgrade --install vault hashicorp/vault \
  --namespace vault \
  --create-namespace \
  --set "server.dev.enabled=true"

echo "Waiting for Vault to be ready..."
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=vault -n vault --timeout=120s

echo "Installing External Secrets Operator (ESO)..."
helm repo add external-secrets https://charts.external-secrets.io
helm repo update external-secrets

helm upgrade --install external-secrets external-secrets/external-secrets \
  --namespace external-secrets \
  --create-namespace \
  --set installCRDs=true

echo "Waiting for External Secrets Operator to be ready..."
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=external-secrets -n external-secrets --timeout=120s

echo "Vault and ESO installed successfully!"
