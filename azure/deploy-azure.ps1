# Azure Deployment Script for DOT to DOCX Converter using Bicep
param(
    [Parameter(Mandatory=$false)]
    [string]$Environment = "prod",

    [Parameter(Mandatory=$false)]
    [string]$ResourceGroup = "DocSpective",

    [Parameter(Mandatory=$false)]
    [string]$Location = "eastus",

    [Parameter(Mandatory=$false)]
    [switch]$SkipBuild = $false,

    [Parameter(Mandatory=$false)]
    [string]$ImageTag = "latest"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  DOT to DOCX Converter - Azure Deploy" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Environment: $Environment" -ForegroundColor Yellow
Write-Host "Resource Group: $ResourceGroup" -ForegroundColor Yellow
Write-Host "Location: $Location" -ForegroundColor Yellow
Write-Host ""

# Variables
$ACR_NAME = "docspectiveacr"
$APP_NAME = "dot-to-docx-converter"
$IMAGE_NAME = "$ACR_NAME.azurecr.io/$APP_NAME"
$BICEP_FILE = "$PSScriptRoot\main.bicep"
$PARAMS_FILE = "$PSScriptRoot\parameters.$Environment.json"

# Check if parameters file exists
if (-not (Test-Path $PARAMS_FILE)) {
    Write-Host "Parameters file not found: $PARAMS_FILE" -ForegroundColor Red
    Write-Host "Using parameters.prod.json as fallback" -ForegroundColor Yellow
    $PARAMS_FILE = "$PSScriptRoot\parameters.prod.json"
}

# Step 1: Build and push Docker image (unless skipped)
if (-not $SkipBuild) {
    Write-Host "Building and pushing Docker image..." -ForegroundColor Green
    Write-Host "This may take several minutes due to LibreOffice installation..." -ForegroundColor Yellow

    # Build in ACR (cloud build) from project root
    $projectRoot = Split-Path -Parent $PSScriptRoot
    $buildResult = az acr build `
        --registry $ACR_NAME `
        --image "${APP_NAME}:$ImageTag" `
        --file "$projectRoot/Dockerfile" `
        $projectRoot `
        --platform linux/amd64 `
        --no-logs `
        2>&1

    if ($LASTEXITCODE -ne 0) {
        Write-Host "Docker build failed!" -ForegroundColor Red
        Write-Host $buildResult
        exit 1
    }

    Write-Host "Docker image built and pushed successfully" -ForegroundColor Green

    # Get the build ID and show logs
    $buildId = ($buildResult | Select-String -Pattern "Queued a build with ID: (.+)").Matches.Groups[1].Value
    if ($buildId) {
        Write-Host "Build ID: $buildId" -ForegroundColor Yellow
        Write-Host "Waiting for build to complete..." -ForegroundColor Yellow

        # Wait for build completion and get status
        $maxAttempts = 60  # 30 minutes max wait
        $attempts = 0

        while ($attempts -lt $maxAttempts) {
            $status = az acr task show-run --registry $ACR_NAME --run-id $buildId --query status -o tsv

            if ($status -eq "Succeeded") {
                Write-Host "Build completed successfully!" -ForegroundColor Green
                break
            }
            elseif ($status -eq "Failed") {
                Write-Host "Build failed!" -ForegroundColor Red
                az acr task logs --registry $ACR_NAME --run-id $buildId
                exit 1
            }
            else {
                Write-Host "Build status: $status (attempt $($attempts + 1)/$maxAttempts)" -ForegroundColor Yellow
                Start-Sleep -Seconds 30
                $attempts++
            }
        }

        if ($attempts -eq $maxAttempts) {
            Write-Host "Build timeout!" -ForegroundColor Red
            exit 1
        }
    }
}
else {
    Write-Host "Skipping Docker build (using existing image)" -ForegroundColor Yellow
}

# Step 2: Deploy using Bicep
Write-Host ""
Write-Host "Deploying infrastructure with Bicep..." -ForegroundColor Green

# Check if resource group exists, create if not
$rgExists = az group exists --name $ResourceGroup
if ($rgExists -eq "false") {
    Write-Host "Creating resource group: $ResourceGroup" -ForegroundColor Yellow
    az group create --name $ResourceGroup --location $Location
}

# Deploy Bicep template
$deploymentName = "dot-to-docx-deploy-$(Get-Date -Format 'yyyyMMddHHmmss')"
$containerImage = "${IMAGE_NAME}:${ImageTag}"

Write-Host "Starting deployment: $deploymentName" -ForegroundColor Yellow
Write-Host "Container image: $containerImage" -ForegroundColor Yellow

$deployResult = az deployment group create `
    --name $deploymentName `
    --resource-group $ResourceGroup `
    --template-file $BICEP_FILE `
    --parameters $PARAMS_FILE `
    --parameters containerImage=$containerImage `
    --query "{AppName:properties.outputs.containerAppName.value, Url:properties.outputs.containerAppUrl.value}" `
    -o json

if ($LASTEXITCODE -ne 0) {
    Write-Host "Deployment failed!" -ForegroundColor Red
    exit 1
}

# Parse deployment outputs
$outputs = $deployResult | ConvertFrom-Json

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Deployment Completed Successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Container App Name: $($outputs.AppName)" -ForegroundColor Cyan
Write-Host "Application URL: $($outputs.Url)" -ForegroundColor Cyan
Write-Host ""
Write-Host "Test endpoints:" -ForegroundColor Yellow
Write-Host "  Health Check: $($outputs.Url)/health" -ForegroundColor White
Write-Host "  Readiness: $($outputs.Url)/health/ready" -ForegroundColor White
Write-Host "  Liveness: $($outputs.Url)/health/live" -ForegroundColor White
Write-Host "  Metrics: $($outputs.Url)/metrics" -ForegroundColor White
Write-Host ""
Write-Host "API Endpoints:" -ForegroundColor Yellow
Write-Host "  POST $($outputs.Url)/api/v1/convert - Convert single file" -ForegroundColor White
Write-Host "  POST $($outputs.Url)/api/v1/batch - Batch conversion" -ForegroundColor White
Write-Host "  GET  $($outputs.Url)/api/v1/jobs/{id} - Check job status" -ForegroundColor White
Write-Host ""

# Test the health endpoint
Write-Host "Testing health endpoint..." -ForegroundColor Yellow
$healthCheck = Invoke-RestMethod -Uri "$($outputs.Url)/health" -Method Get -ErrorAction SilentlyContinue

if ($healthCheck) {
    Write-Host "Health check passed!" -ForegroundColor Green
}
else {
    Write-Host "Health check failed or service is still starting up" -ForegroundColor Yellow
    Write-Host "Please wait a few minutes for the service to fully initialize" -ForegroundColor Yellow
}