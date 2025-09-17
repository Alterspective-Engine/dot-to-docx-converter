#!/bin/bash

# Azure Deployment Script for DOT to DOCX Converter
# This script automates the deployment of the container app to Azure

set -e

# Configuration
RESOURCE_GROUP="DocSpective"
LOCATION="eastus"
ACR_NAME="docspectiveacr"
APP_NAME="dot-to-docx-converter"
ENV_NAME="docspective-env"
REDIS_NAME="docspective-redis"

echo "Starting Azure deployment..."

# 1. Create Azure Container Registry if it doesn't exist
echo "Setting up Azure Container Registry..."
if ! az acr show --name $ACR_NAME --resource-group $RESOURCE_GROUP &>/dev/null; then
    az acr create \
        --name $ACR_NAME \
        --resource-group $RESOURCE_GROUP \
        --sku Basic \
        --admin-enabled true
fi

# 2. Get ACR credentials
ACR_USERNAME=$(az acr credential show --name $ACR_NAME --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name $ACR_NAME --query passwords[0].value -o tsv)

# 3. Login to ACR
echo "Logging into ACR..."
az acr login --name $ACR_NAME

# 4. Build and push Docker image
echo "Building and pushing Docker image..."
docker build -t $ACR_NAME.azurecr.io/$APP_NAME:latest .
docker push $ACR_NAME.azurecr.io/$APP_NAME:latest

# 5. Create Redis Cache for queue (if not exists)
echo "Setting up Redis Cache..."
if ! az redis show --name $REDIS_NAME --resource-group $RESOURCE_GROUP &>/dev/null; then
    az redis create \
        --name $REDIS_NAME \
        --resource-group $RESOURCE_GROUP \
        --location $LOCATION \
        --sku Basic \
        --vm-size c0 \
        --enable-non-ssl-port
fi

# Get Redis connection details
REDIS_HOST=$(az redis show --name $REDIS_NAME --resource-group $RESOURCE_GROUP --query hostName -o tsv)
REDIS_KEY=$(az redis list-keys --name $REDIS_NAME --resource-group $RESOURCE_GROUP --query primaryKey -o tsv)
REDIS_URL="redis://:${REDIS_KEY}@${REDIS_HOST}:6379"

# 6. Create Container App Environment (if not exists)
echo "Setting up Container App Environment..."
if ! az containerapp env show --name $ENV_NAME --resource-group $RESOURCE_GROUP &>/dev/null; then
    az containerapp env create \
        --name $ENV_NAME \
        --resource-group $RESOURCE_GROUP \
        --location $LOCATION
fi

# 7. Deploy or update Container App
echo "Deploying Container App..."
if az containerapp show --name $APP_NAME --resource-group $RESOURCE_GROUP &>/dev/null; then
    # Update existing app
    az containerapp update \
        --name $APP_NAME \
        --resource-group $RESOURCE_GROUP \
        --image $ACR_NAME.azurecr.io/$APP_NAME:latest \
        --cpu 2 \
        --memory 4 \
        --min-replicas 1 \
        --max-replicas 10 \
        --set-env-vars \
            PORT=8080 \
            WORKER_COUNT=10 \
            REDIS_URL="$REDIS_URL" \
            MAX_FILE_SIZE=50 \
            CONVERSION_TIMEOUT=60 \
            LOG_LEVEL=info
else
    # Create new app
    az containerapp create \
        --name $APP_NAME \
        --resource-group $RESOURCE_GROUP \
        --environment $ENV_NAME \
        --image $ACR_NAME.azurecr.io/$APP_NAME:latest \
        --target-port 8080 \
        --ingress external \
        --cpu 2 \
        --memory 4 \
        --min-replicas 1 \
        --max-replicas 10 \
        --registry-server $ACR_NAME.azurecr.io \
        --registry-username $ACR_USERNAME \
        --registry-password $ACR_PASSWORD \
        --env-vars \
            PORT=8080 \
            WORKER_COUNT=10 \
            REDIS_URL="$REDIS_URL" \
            MAX_FILE_SIZE=50 \
            CONVERSION_TIMEOUT=60 \
            LOG_LEVEL=info
fi

# 8. Get the app URL
APP_URL=$(az containerapp show --name $APP_NAME --resource-group $RESOURCE_GROUP --query properties.configuration.ingress.fqdn -o tsv)

echo ""
echo "Deployment completed successfully!"
echo "Application URL: https://$APP_URL"
echo ""
echo "To test the deployment:"
echo "curl https://$APP_URL/health"