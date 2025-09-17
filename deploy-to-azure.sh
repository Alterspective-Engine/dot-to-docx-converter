#!/bin/bash

# Azure Deployment Script for DOT to DOCX Converter
# Run this script to deploy the latest changes to Azure

echo "🚀 Starting Azure Deployment..."

# Configuration
ACR_NAME="alterspectiveacr"
IMAGE_NAME="dot-to-docx-converter"
RESOURCE_GROUP="DocSpective"
CONTAINER_APP="dot-to-docx-converter-prod"
VERSION_TAG="v1.0.5"

echo "📦 Building Docker image..."
docker build -t $ACR_NAME.azurecr.io/$IMAGE_NAME:latest \
             -t $ACR_NAME.azurecr.io/$IMAGE_NAME:$VERSION_TAG .

if [ $? -ne 0 ]; then
    echo "❌ Docker build failed. Make sure Docker Desktop is running."
    exit 1
fi

echo "🔐 Logging into Azure Container Registry..."
az acr login --name $ACR_NAME

if [ $? -ne 0 ]; then
    echo "❌ ACR login failed. Make sure you're logged into Azure CLI."
    exit 1
fi

echo "📤 Pushing image to ACR..."
docker push $ACR_NAME.azurecr.io/$IMAGE_NAME:latest
docker push $ACR_NAME.azurecr.io/$IMAGE_NAME:$VERSION_TAG

if [ $? -ne 0 ]; then
    echo "❌ Docker push failed."
    exit 1
fi

echo "🔄 Updating Container App..."
az containerapp update \
    --name $CONTAINER_APP \
    --resource-group $RESOURCE_GROUP \
    --image $ACR_NAME.azurecr.io/$IMAGE_NAME:latest

if [ $? -ne 0 ]; then
    echo "❌ Container App update failed."
    exit 1
fi

echo "✅ Deployment completed successfully!"
echo "🌐 Application URL: https://dot-to-docx-converter-prod.lemondesert-9ded9ffc.eastus.azurecontainerapps.io"
echo ""
echo "📊 Check deployment status:"
echo "   az containerapp show --name $CONTAINER_APP --resource-group $RESOURCE_GROUP --query properties.latestRevisionName"