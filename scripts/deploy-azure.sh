#!/bin/bash

# Azure Deployment Script for DOT to DOCX Converter
# This script deploys the converter service to Azure Container Instances

set -e

# Configuration
RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-dot-converter-rg}"
LOCATION="${AZURE_LOCATION:-eastus}"
CONTAINER_NAME="${CONTAINER_NAME:-dot-to-docx-converter}"
REGISTRY_NAME="${REGISTRY_NAME:-alterspective}"
IMAGE_NAME="dot-to-docx-converter"
IMAGE_TAG="${IMAGE_TAG:-latest}"

# Redis Cache configuration
REDIS_NAME="${REDIS_NAME:-dot-converter-redis}"
REDIS_SKU="${REDIS_SKU:-Basic}"
REDIS_SIZE="${REDIS_SIZE:-C0}"

# Storage Account configuration
STORAGE_ACCOUNT="${STORAGE_ACCOUNT:-dotconverterstorage}"
STORAGE_CONTAINER="conversions"

# Container configuration
CPU_CORES="${CPU_CORES:-4}"
MEMORY_GB="${MEMORY_GB:-8}"
WORKER_COUNT="${WORKER_COUNT:-20}"

echo "🚀 Deploying DOT to DOCX Converter to Azure"

# Check if logged in to Azure
echo "Checking Azure login status..."
if ! az account show &>/dev/null; then
    echo "Please login to Azure first:"
    az login
fi

# Create Resource Group
echo "Creating resource group: $RESOURCE_GROUP"
az group create \
    --name "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --output none

# Create Storage Account
echo "Creating storage account: $STORAGE_ACCOUNT"
az storage account create \
    --name "$STORAGE_ACCOUNT" \
    --resource-group "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --sku Standard_LRS \
    --kind StorageV2 \
    --output none

# Get Storage Connection String
echo "Getting storage connection string..."
STORAGE_CONNECTION=$(az storage account show-connection-string \
    --name "$STORAGE_ACCOUNT" \
    --resource-group "$RESOURCE_GROUP" \
    --query connectionString \
    --output tsv)

# Create Storage Container
echo "Creating storage container: $STORAGE_CONTAINER"
az storage container create \
    --name "$STORAGE_CONTAINER" \
    --account-name "$STORAGE_ACCOUNT" \
    --output none

# Create Redis Cache
echo "Creating Redis Cache: $REDIS_NAME (this may take several minutes)"
az redis create \
    --name "$REDIS_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --sku "$REDIS_SKU" \
    --vm-size "$REDIS_SIZE" \
    --output none

# Get Redis connection details
echo "Getting Redis connection details..."
REDIS_HOST=$(az redis show \
    --name "$REDIS_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --query hostName \
    --output tsv)

REDIS_KEY=$(az redis list-keys \
    --name "$REDIS_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --query primaryKey \
    --output tsv)

REDIS_URL="redis://:${REDIS_KEY}@${REDIS_HOST}:6380?ssl=true"

# Create Container Registry (if not exists)
echo "Checking Container Registry..."
if ! az acr show --name "$REGISTRY_NAME" &>/dev/null; then
    echo "Creating Container Registry: $REGISTRY_NAME"
    az acr create \
        --name "$REGISTRY_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --sku Basic \
        --output none
fi

# Build and push Docker image
echo "Building and pushing Docker image..."
az acr build \
    --registry "$REGISTRY_NAME" \
    --image "${IMAGE_NAME}:${IMAGE_TAG}" \
    --file Dockerfile \
    .

# Get Registry credentials
REGISTRY_SERVER="${REGISTRY_NAME}.azurecr.io"
REGISTRY_USERNAME=$(az acr credential show \
    --name "$REGISTRY_NAME" \
    --query username \
    --output tsv)
REGISTRY_PASSWORD=$(az acr credential show \
    --name "$REGISTRY_NAME" \
    --query passwords[0].value \
    --output tsv)

# Deploy Container Instance
echo "Deploying Container Instance: $CONTAINER_NAME"
az container create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CONTAINER_NAME" \
    --image "${REGISTRY_SERVER}/${IMAGE_NAME}:${IMAGE_TAG}" \
    --cpu "$CPU_CORES" \
    --memory "$MEMORY_GB" \
    --registry-login-server "$REGISTRY_SERVER" \
    --registry-username "$REGISTRY_USERNAME" \
    --registry-password "$REGISTRY_PASSWORD" \
    --ports 8080 \
    --environment-variables \
        PORT=8080 \
        WORKER_COUNT="$WORKER_COUNT" \
        REDIS_URL="$REDIS_URL" \
        AZURE_STORAGE_CONNECTION_STRING="$STORAGE_CONNECTION" \
        LOG_LEVEL=info \
        CONVERSION_TIMEOUT=60 \
    --dns-name-label "${CONTAINER_NAME}-${RANDOM}" \
    --restart-policy OnFailure \
    --output none

# Get container details
echo "Getting container details..."
CONTAINER_FQDN=$(az container show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CONTAINER_NAME" \
    --query ipAddress.fqdn \
    --output tsv)

CONTAINER_STATUS=$(az container show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CONTAINER_NAME" \
    --query instanceView.state \
    --output tsv)

echo "✅ Deployment Complete!"
echo ""
echo "Service Details:"
echo "  URL: http://${CONTAINER_FQDN}:8080"
echo "  Status: ${CONTAINER_STATUS}"
echo "  Resource Group: ${RESOURCE_GROUP}"
echo "  Container Name: ${CONTAINER_NAME}"
echo ""
echo "API Endpoints:"
echo "  Health: http://${CONTAINER_FQDN}:8080/health"
echo "  Metrics: http://${CONTAINER_FQDN}:8080/metrics"
echo "  Convert: POST http://${CONTAINER_FQDN}:8080/api/v1/convert"
echo "  Batch: POST http://${CONTAINER_FQDN}:8080/api/v1/batch"
echo ""
echo "To view logs:"
echo "  az container logs --resource-group ${RESOURCE_GROUP} --name ${CONTAINER_NAME}"
echo ""
echo "To scale the service:"
echo "  Update CPU_CORES and MEMORY_GB environment variables and redeploy"