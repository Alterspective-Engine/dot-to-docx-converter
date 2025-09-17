# Azure Deployment Script for DOT to DOCX Converter - AUSTRALIA EAST
# This script creates a new deployment in Australia East region

Write-Host "🚀 Starting Azure Deployment to AUSTRALIA EAST..." -ForegroundColor Cyan

# Configuration
$ACR_NAME = "alterspectiveacr"
$IMAGE_NAME = "dot-to-docx-converter"
$RESOURCE_GROUP = "DocSpective"
$LOCATION = "australiaeast"
$ENV_NAME = "dot-to-docx-converter-au-env"
$CONTAINER_APP = "dot-to-docx-converter-au"
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

# Check if Container Apps Environment exists in Australia East
Write-Host "🔍 Checking for Container Apps Environment in Australia East..." -ForegroundColor Yellow
$envExists = az containerapp env show --name $ENV_NAME --resource-group $RESOURCE_GROUP --query name -o tsv 2>$null

if (-not $envExists) {
    Write-Host "📍 Creating Container Apps Environment in Australia East..." -ForegroundColor Yellow
    az containerapp env create `
        --name $ENV_NAME `
        --resource-group $RESOURCE_GROUP `
        --location $LOCATION

    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Failed to create Container Apps Environment." -ForegroundColor Red
        exit 1
    }
}

# Check if Container App exists
Write-Host "🔍 Checking for Container App in Australia East..." -ForegroundColor Yellow
$appExists = az containerapp show --name $CONTAINER_APP --resource-group $RESOURCE_GROUP --query name -o tsv 2>$null

if (-not $appExists) {
    Write-Host "📱 Creating Container App in Australia East..." -ForegroundColor Yellow
    az containerapp create `
        --name $CONTAINER_APP `
        --resource-group $RESOURCE_GROUP `
        --environment $ENV_NAME `
        --image "$ACR_NAME.azurecr.io/${IMAGE_NAME}:latest" `
        --target-port 8080 `
        --ingress 'external' `
        --cpu 0.5 `
        --memory 1 `
        --min-replicas 1 `
        --max-replicas 10 `
        --registry-server "$ACR_NAME.azurecr.io" `
        --env-vars `
            AZURE_STORAGE_CONNECTION_STRING=secretref:azure-storage `
            REDIS_URL=secretref:redis-url `
            WORKER_COUNT=10 `
            MAX_FILE_SIZE_MB=50 `
            CONVERSION_TIMEOUT=300s

    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Failed to create Container App." -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "🔄 Updating existing Container App in Australia East..." -ForegroundColor Yellow
    az containerapp update `
        --name $CONTAINER_APP `
        --resource-group $RESOURCE_GROUP `
        --image "$ACR_NAME.azurecr.io/${IMAGE_NAME}:latest"

    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Container App update failed." -ForegroundColor Red
        exit 1
    }
}

# Get the application URL
$appUrl = az containerapp show --name $CONTAINER_APP --resource-group $RESOURCE_GROUP --query "properties.configuration.ingress.fqdn" -o tsv

Write-Host ""
Write-Host "✅ Deployment to AUSTRALIA EAST completed successfully!" -ForegroundColor Green
Write-Host "🌐 Application URL: https://$appUrl" -ForegroundColor Cyan
Write-Host ""
Write-Host "📊 Container App Details:" -ForegroundColor Yellow
Write-Host "   Region: Australia East" -ForegroundColor White
Write-Host "   Resource Group: $RESOURCE_GROUP" -ForegroundColor White
Write-Host "   Container App: $CONTAINER_APP" -ForegroundColor White
Write-Host "   Environment: $ENV_NAME" -ForegroundColor White
Write-Host ""
Write-Host "📊 Check deployment status:" -ForegroundColor Yellow
Write-Host "   az containerapp show --name $CONTAINER_APP --resource-group $RESOURCE_GROUP --query properties.latestRevisionName"