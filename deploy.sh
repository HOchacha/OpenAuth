#!/bin/bash

# deploy.sh
set -e

# Check required environment variables
if [ -z "$HUB" ] || [ -z "$TAG" ]; then
    echo "Error: HUB and TAG environment variables must be set"
    echo "Usage: HUB=your-registry TAG=version ./deploy.sh"
    exit 1
fi

# Deploy Service Accounts
echo "Deploy Service Accounts for OpenAuth & oauthctl"
kubectl apply -f configs/service-account.yaml

# Replace placeholders in deployment YAML
echo "Preparing deployment YAML..."
sed "s#\${HUB}#$HUB#g; s#\${TAG}#$TAG#g" configs/deployment/openauth-depl.yaml > configs/deployment/openauth-generated.yaml

# Apply Kubernetes configurations
echo "Applying Kubernetes configurations..."
kubectl apply -f configs/deployment/openauth-generated.yaml

echo "Deployment completed successfully!"