// Main Bicep template for DOT to DOCX Converter Service
// This template creates all required infrastructure including ACR, Container Apps, Redis, and monitoring

@description('Base name for the application')
param appName string = 'dot-to-docx-converter'

@description('Environment name (dev, test, prod)')
@allowed(['dev', 'test', 'prod'])
param environment string = 'prod'

@description('Azure region for resources')
param location string = resourceGroup().location

@description('Azure Container Registry name')
param acrName string = 'docspectiveacr'

@description('Resource group where ACR exists (if different from deployment RG)')
param acrResourceGroup string = resourceGroup().name

@description('ACR SKU')
@allowed(['Basic', 'Standard', 'Premium'])
param acrSku string = 'Basic'

@description('Container image with tag')
param containerImage string = ''

@description('CPU allocation (e.g., 0.25, 0.5, 1.0, 2.0)')
param cpu string = '2.0'

@description('Memory allocation (e.g., 0.5Gi, 1.0Gi, 2.0Gi, 4.0Gi)')
param memory string = '4.0Gi'

@description('Minimum number of replicas')
@minValue(0)
@maxValue(30)
param minReplicas int = 1

@description('Maximum number of replicas')
@minValue(1)
@maxValue(30)
param maxReplicas int = 10

@description('Log Analytics retention in days')
@minValue(30)
@maxValue(730)
param logRetentionDays int = 30

@description('Enable Application Insights')
param enableAppInsights bool = true

@description('Create new ACR or use existing')
param createAcr bool = false

@description('Create Redis Cache for job queue')
param createRedis bool = true

// ============ Service Configuration Parameters ============
@description('Service port')
param servicePort string = '8080'

@description('Worker count for concurrent processing')
param workerCount string = '10'

@description('Maximum file size in MB')
param maxFileSize string = '50'

@description('Conversion timeout in seconds')
param conversionTimeout string = '60'

@description('Log level')
@allowed(['debug', 'info', 'warn', 'error'])
param logLevel string = 'info'

@description('Redis configuration')
param redisPort string = '6379'
param redisEnableNonSslPort bool = true

@description('Azure Storage Connection String (optional)')
@secure()
param azureStorageConnectionString string = ''

// ============ Variables ============
var uniqueSuffix = uniqueString(resourceGroup().id)
var acrLoginServer = createAcr ? containerRegistry.properties.loginServer : '${acrName}.azurecr.io'

// Environment-specific naming
var containerAppName = '${appName}-${environment}'
var containerEnvName = '${appName}-${environment}-env'
var logAnalyticsName = '${appName}-${environment}-law'
var appInsightsName = '${appName}-${environment}-ai'
var redisName = toLower('${appName}-${environment}-redis-${uniqueSuffix}')

// Redis URL construction
var redisUrl = createRedis ? 'redis://:${redisCache.listKeys().primaryKey}@${redisCache.properties.hostName}:${redisPort}' : ''

// ============ Resources ============

// Azure Container Registry (optional - only create if needed)
resource containerRegistry 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' = if (createAcr) {
  name: acrName
  location: location
  sku: {
    name: acrSku
  }
  properties: {
    adminUserEnabled: true
    publicNetworkAccess: 'Enabled'
    networkRuleBypassOptions: 'AzureServices'
  }
}

// Redis Cache for job queue
resource redisCache 'Microsoft.Cache/redis@2023-08-01' = if (createRedis) {
  name: redisName
  location: location
  properties: {
    sku: {
      name: 'Basic'
      family: 'C'
      capacity: 0
    }
    enableNonSslPort: redisEnableNonSslPort
    minimumTlsVersion: '1.2'
    publicNetworkAccess: 'Enabled'
  }
}

// Log Analytics Workspace
resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: logAnalyticsName
  location: location
  properties: {
    retentionInDays: logRetentionDays
    publicNetworkAccessForIngestion: 'Enabled'
    publicNetworkAccessForQuery: 'Enabled'
    sku: {
      name: 'PerGB2018'
    }
  }
}

// Application Insights
resource appInsights 'Microsoft.Insights/components@2020-02-02' = if (enableAppInsights) {
  name: appInsightsName
  location: location
  kind: 'web'
  properties: {
    Application_Type: 'web'
    WorkspaceResourceId: logAnalytics.id
    publicNetworkAccessForIngestion: 'Enabled'
    publicNetworkAccessForQuery: 'Enabled'
  }
}

// Container Apps Environment
resource containerEnvironment 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: containerEnvName
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalytics.properties.customerId
        sharedKey: logAnalytics.listKeys().primarySharedKey
      }
    }
    workloadProfiles: [
      {
        name: 'Consumption'
        workloadProfileType: 'Consumption'
      }
    ]
  }
}

// Get existing ACR credentials if not creating new
resource existingAcr 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' existing = if (!createAcr) {
  name: acrName
  scope: resourceGroup(acrResourceGroup)
}

// Container App
resource containerApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: containerAppName
  location: location
  properties: {
    managedEnvironmentId: containerEnvironment.id
    workloadProfileName: 'Consumption'
    configuration: {
      ingress: {
        external: true
        targetPort: int(servicePort)
        traffic: [
          {
            latestRevision: true
            weight: 100
          }
        ]
        corsPolicy: {
          allowedOrigins: ['*']
          allowedMethods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS']
          allowedHeaders: ['*']
          maxAge: 86400
        }
      }
      registries: createAcr ? [
        {
          server: acrLoginServer
          username: containerRegistry.listCredentials().username
          passwordSecretRef: 'acr-password'
        }
      ] : [
        {
          server: acrLoginServer
          username: existingAcr.listCredentials().username
          passwordSecretRef: 'acr-password'
        }
      ]
      secrets: concat([
        {
          name: 'acr-password'
          value: createAcr ? containerRegistry.listCredentials().passwords[0].value : existingAcr.listCredentials().passwords[0].value
        }
      ], createRedis ? [
        {
          name: 'redis-key'
          value: redisCache.listKeys().primaryKey
        }
      ] : [], azureStorageConnectionString != '' ? [
        {
          name: 'azure-storage-connection'
          value: azureStorageConnectionString
        }
      ] : [])
      activeRevisionsMode: 'Single'
      maxInactiveRevisions: 5
    }
    template: {
      containers: [
        {
          name: 'converter'
          image: containerImage != '' ? containerImage : '${acrLoginServer}/${appName}:latest'
          resources: {
            cpu: json(cpu)
            memory: memory
          }
          env: concat([
            // Core Service Configuration
            { name: 'PORT', value: servicePort }
            { name: 'WORKER_COUNT', value: workerCount }
            { name: 'MAX_FILE_SIZE', value: maxFileSize }
            { name: 'CONVERSION_TIMEOUT', value: conversionTimeout }
            { name: 'LOG_LEVEL', value: logLevel }
          ], createRedis ? [
            // Redis Configuration
            { name: 'REDIS_URL', value: redisUrl }
          ] : [], azureStorageConnectionString != '' ? [
            // Azure Storage Configuration
            { name: 'AZURE_STORAGE_CONNECTION_STRING', secretRef: 'azure-storage-connection' }
          ] : [], enableAppInsights ? [
            // Application Insights
            { name: 'APPLICATIONINSIGHTS_CONNECTION_STRING', value: appInsights.properties.ConnectionString }
          ] : [])
          probes: [
            {
              type: 'Liveness'
              httpGet: {
                path: '/health/live'
                port: int(servicePort)
              }
              initialDelaySeconds: 30
              periodSeconds: 30
            }
            {
              type: 'Readiness'
              httpGet: {
                path: '/health/ready'
                port: int(servicePort)
              }
              initialDelaySeconds: 10
              periodSeconds: 10
            }
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
        rules: [
          {
            name: 'http-scaling'
            http: {
              metadata: {
                concurrentRequests: '100'
              }
            }
          }
          {
            name: 'cpu-scaling'
            custom: {
              type: 'cpu'
              metadata: {
                type: 'Utilization'
                value: '70'
              }
            }
          }
        ]
      }
    }
  }
}

// ============ Outputs ============
output containerAppName string = containerApp.name
output containerAppUrl string = 'https://${containerApp.properties.configuration.ingress.fqdn}'
output acrLoginServer string = acrLoginServer
output redisHostName string = createRedis ? redisCache.properties.hostName : ''
output logAnalyticsWorkspaceId string = logAnalytics.properties.customerId
output appInsightsConnectionString string = enableAppInsights ? appInsights.properties.ConnectionString : ''
output appInsightsInstrumentationKey string = enableAppInsights ? appInsights.properties.InstrumentationKey : ''
output containerEnvironmentId string = containerEnvironment.id