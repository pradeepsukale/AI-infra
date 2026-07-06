#!/bin/bash

# Exit on any error
set -e

echo "1. Adding Helm repositories..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update

echo "2. Installing PostgreSQL..."
# We install postgresql in the 'default' namespace (or you can change it)
# We set an admin password, and create a specific user/db for Vault.
# You can use the admin credentials to create other databases later!
helm upgrade --install shared-postgres bitnami/postgresql \
  --namespace default \
  --set global.postgresql.auth.postgresPassword=postgres \
  --set auth.database=vaultdb \
  --set auth.username=vault \
  --set auth.password=vaultpassword

echo "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=postgresql -n default --timeout=120s

echo "3. Re-configuring Vault to use PostgreSQL (disabling dev mode)..."
# We need to disable dev mode because dev mode forces in-memory storage.
# We configure the PostgreSQL storage block to point to our new DB.
cat <<EOF > vault-values.yaml
server:
  dev:
    enabled: false
  standalone:
    enabled: true
    config: |
      ui = true
      listener "tcp" {
        tls_disable = 1
        address = "[::]:8200"
        cluster_address = "[::]:8201"
      }
      storage "postgresql" {
        connection_url = "postgres://vault:vaultpassword@shared-postgres-postgresql.default.svc.cluster.local:5432/vaultdb?sslmode=disable"
      }
EOF

echo "Uninstalling existing Dev Mode Vault to ensure a clean slate..."
helm uninstall vault -n vault || true
kubectl delete pvc -l app.kubernetes.io/name=vault -n vault || true

helm upgrade --install vault hashicorp/vault \
  --namespace vault \
  --values vault-values.yaml

echo "Waiting for Vault pod to be created..."
sleep 5
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=vault -n vault --timeout=120s || echo "Vault is running but sealed. We need to initialize it."

echo "4. Ensuring External Secrets Operator (ESO) is installed..."
helm repo add external-secrets https://charts.external-secrets.io
helm repo update external-secrets

helm upgrade --install external-secrets external-secrets/external-secrets \
  --namespace external-secrets \
  --create-namespace \
  --set installCRDs=true

echo "Waiting for External Secrets Operator to be ready..."
kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=external-secrets -n external-secrets --timeout=120s || true

echo "=========================================================================="
echo "DEPLOYMENT COMPLETE!"
echo ""
echo "Because Vault is no longer in Dev Mode, it starts SEALED and UNINITIALIZED."
echo "You must run the following commands manually to initialize and unseal it:"
echo ""
echo "1. Initialize Vault (save the output, it contains your unseal keys and root token!):"
echo "   kubectl exec -it vault-0 -n vault -- vault operator init -key-shares=1 -key-threshold=1"
echo ""
echo "2. Unseal Vault (using the Unseal Key from step 1):"
echo "   kubectl exec -it vault-0 -n vault -- vault operator unseal <UNSEAL_KEY>"
echo ""
echo "3. Log in to Vault (using the Initial Root Token from step 1):"
echo "   kubectl exec -it vault-0 -n vault -- vault login <ROOT_TOKEN>"
echo "=========================================================================="
