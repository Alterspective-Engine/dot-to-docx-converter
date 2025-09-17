# Azure Deployment Script for DOT to DOCX Converter (PowerShell)
# Run this script to deploy the latest changes to Azure

Write-Host "🚀 Starting Azure Deployment..." -ForegroundColor Cyan

# Configuration
$ACR_NAME = "alterspectiveacr"
$IMAGE_NAME = "dot-to-docx-converter"
$RESOURCE_GROUP = "DocSpective"
$CONTAINER_APP = "dot-to-docx-converter-prod"
$VERSION_TAG = "v1.0.5"

Write-Host "📦 Building Docker image..." -ForegroundColor Yellow
$buildResult = docker build -t "$ACR_NAME.azurecr.io/${IMAGE_NAME}:latest" `
                           -t "$ACR_NAME.azurecr.io/${IMAGE_NAME}:$VERSION_TAG" .

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker build failed. Make sure Docker Desktop is running." -ForegroundColor Red
    exit 1
}

Write-Host "🔐 Logging into Azure Container Registry..." -ForegroundColor Yellow
az acr login --name $ACR_NAME

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ ACR login failed. Make sure you're logged into Azure CLI." -ForegroundColor Red
    Write-Host "   Run: az login" -ForegroundColor Yellow
    exit 1
}

Write-Host "📤 Pushing image to ACR..." -ForegroundColor Yellow
docker push "$ACR_NAME.azurecr.io/${IMAGE_NAME}:latest"
docker push "$ACR_NAME.azurecr.io/${IMAGE_NAME}:$VERSION_TAG"

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker push failed." -ForegroundColor Red
    exit 1
}

Write-Host "🔄 Updating Container App..." -ForegroundColor Yellow
az containerapp update `
    --name $CONTAINER_APP `
    --resource-group $RESOURCE_GROUP `
    --image "$ACR_NAME.azurecr.io/${IMAGE_NAME}:latest"

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Container App update failed." -ForegroundColor Red
    exit 1
}

Write-Host "✅ Deployment completed successfully!" -ForegroundColor Green
Write-Host "🌐 Application URL: https://dot-to-docx-converter-prod.lemondesert-9ded9ffc.eastus.azurecontainerapps.io" -ForegroundColor Cyan
Write-Host ""
Write-Host "📊 Check deployment status:" -ForegroundColor Yellow
Write-Host "   az containerapp show --name $CONTAINER_APP --resource-group $RESOURCE_GROUP --query properties.latestRevisionName"