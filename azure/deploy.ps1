# Azure Deployment Script for DOT to DOCX Converter (PowerShell)
# This script automates the deployment of the container app to Azure

$ErrorActionPreference = "Stop"

# Configuration
$RESOURCE_GROUP = "DocSpective"
$LOCATION = "eastus"
$ACR_NAME = "docspectiveacr"
$APP_NAME = "dot-to-docx-converter"
$ENV_NAME = "docspective-env"
$REDIS_NAME = "docspective-redis"

Write-Host "Starting Azure deployment..." -ForegroundColor Green

# 1. Create Azure Container Registry if it doesn't exist
Write-Host "Setting up Azure Container Registry..." -ForegroundColor Yellow
$acrExists = az acr show --name $ACR_NAME --resource-group $RESOURCE_GROUP 2>$null
if (-not $acrExists) {
    az acr create `
        --name $ACR_NAME `
        --resource-group $RESOURCE_GROUP `
        --sku Basic `
        --admin-enabled true
}

# 2. Get ACR credentials
$ACR_USERNAME = az acr credential show --name $ACR_NAME --query username -o tsv
$ACR_PASSWORD = az acr credential show --name $ACR_NAME --query "passwords[0].value" -o tsv

# 3. Login to ACR
Write-Host "Logging into ACR..." -ForegroundColor Yellow
az acr login --name $ACR_NAME

# 4. Build and push Docker image
Write-Host "Building and pushing Docker image..." -ForegroundColor Yellow
docker build -t "$ACR_NAME.azurecr.io/${APP_NAME}:latest" .
docker push "$ACR_NAME.azurecr.io/${APP_NAME}:latest"

# 5. Create Redis Cache for queue (if not exists)
Write-Host "Setting up Redis Cache..." -ForegroundColor Yellow
$redisExists = az redis show --name $REDIS_NAME --resource-group $RESOURCE_GROUP 2>$null
if (-not $redisExists) {
    az redis create `
        --name $REDIS_NAME `
        --resource-group $RESOURCE_GROUP `
        --location $LOCATION `
        --sku Basic `
        --vm-size c0 `
        --enable-non-ssl-port
}

# Get Redis connection details
$REDIS_HOST = az redis show --name $REDIS_NAME --resource-group $RESOURCE_GROUP --query hostName -o tsv
$REDIS_KEY = az redis list-keys --name $REDIS_NAME --resource-group $RESOURCE_GROUP --query primaryKey -o tsv
$REDIS_URL = "redis://:${REDIS_KEY}@${REDIS_HOST}:6379"

# 6. Create Container App Environment (if not exists)
Write-Host "Setting up Container App Environment..." -ForegroundColor Yellow
$envExists = az containerapp env show --name $ENV_NAME --resource-group $RESOURCE_GROUP 2>$null
if (-not $envExists) {
    az containerapp env create `
        --name $ENV_NAME `
        --resource-group $RESOURCE_GROUP `
        --location $LOCATION
}

# 7. Deploy or update Container App
Write-Host "Deploying Container App..." -ForegroundColor Yellow
$appExists = az containerapp show --name $APP_NAME --resource-group $RESOURCE_GROUP 2>$null
if ($appExists) {
    # Update existing app
    az containerapp update `
        --name $APP_NAME `
        --resource-group $RESOURCE_GROUP `
        --image "$ACR_NAME.azurecr.io/${APP_NAME}:latest" `
        --cpu 2 `
        --memory 4 `
        --min-replicas 1 `
        --max-replicas 10 `
        --set-env-vars `
            PORT=8080 `
            WORKER_COUNT=10 `
            REDIS_URL="$REDIS_URL" `
            MAX_FILE_SIZE=50 `
            CONVERSION_TIMEOUT=60 `
            LOG_LEVEL=info
} else {
    # Create new app
    az containerapp create `
        --name $APP_NAME `
        --resource-group $RESOURCE_GROUP `
        --environment $ENV_NAME `
        --image "$ACR_NAME.azurecr.io/${APP_NAME}:latest" `
        --target-port 8080 `
        --ingress external `
        --cpu 2 `
        --memory 4 `
        --min-replicas 1 `
        --max-replicas 10 `
        --registry-server "$ACR_NAME.azurecr.io" `
        --registry-username $ACR_USERNAME `
        --registry-password $ACR_PASSWORD `
        --env-vars `
            PORT=8080 `
            WORKER_COUNT=10 `
            REDIS_URL="$REDIS_URL" `
            MAX_FILE_SIZE=50 `
            CONVERSION_TIMEOUT=60 `
            LOG_LEVEL=info
}

# 8. Get the app URL
$APP_URL = az containerapp show --name $APP_NAME --resource-group $RESOURCE_GROUP --query "properties.configuration.ingress.fqdn" -o tsv

Write-Host ""
Write-Host "Deployment completed successfully!" -ForegroundColor Green
Write-Host "Application URL: https://$APP_URL" -ForegroundColor Cyan
Write-Host ""
Write-Host "To test the deployment:" -ForegroundColor Yellow
Write-Host "Invoke-RestMethod -Uri https://$APP_URL/health" -ForegroundColor White