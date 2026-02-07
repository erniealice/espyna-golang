#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build cloud-native Espyna server optimized for single cloud provider deployment

.DESCRIPTION
    This script creates a cloud-optimized build with:
    - Fiber HTTP framework (high performance for cloud workloads)
    - Single cloud provider stack (Google Cloud, AWS, or Azure)
    - Cloud-native database (managed database services)
    - Cloud authentication (Firebase Auth, AWS Cognito, or Azure AD)
    - Cloud email and storage services
    - Optimized for containerized deployment and auto-scaling

.PARAMETER CloudProvider
    Target cloud provider. Valid values: gcp, aws, azure
    Default: gcp

.PARAMETER Output
    Output binary name. Auto-generated based on cloud provider if not specified

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: false

.EXAMPLE
    .\build-cloud-native.ps1 -CloudProvider gcp
    Google Cloud Platform optimized build

.EXAMPLE
    .\build-cloud-native.ps1 -CloudProvider aws -VerboseBuild
    AWS optimized build with verbose output

.EXAMPLE
    .\build-cloud-native.ps1 -CloudProvider azure -MockMode:$true
    Azure optimized build with mock providers for testing

.NOTES
    This build configuration is optimized for:
    - Cloud-native deployment (25-35MB vs 70-80MB enterprise builds)
    - Single cloud provider integration (avoiding multi-cloud complexity)
    - Container-first architecture (Docker, Kubernetes)
    - Auto-scaling and serverless architectures
    - Cloud-managed services (databases, auth, storage)
#>

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("gcp", "aws", "azure")]
    [string]$CloudProvider = "gcp",
    
    [Parameter(Mandatory=$false)]
    [string]$Output = "",
    
    [Parameter(Mandatory=$false)]
    [switch]$VerboseBuild,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [bool]$MockMode = $false
)

# Auto-generate output name if not specified
if (-not $Output) {
    $Output = "espyna-cloud-$CloudProvider"
}

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "=== Espyna Cloud-Native Build ($($CloudProvider.ToUpper())) ===" -ForegroundColor Blue
    Write-Host "Building cloud-optimized server for:" -ForegroundColor White
    Write-Host "  • Single cloud provider integration" -ForegroundColor Cyan
    Write-Host "  • Container-first deployment (Docker/Kubernetes)" -ForegroundColor Cyan
    Write-Host "  • Auto-scaling and managed services" -ForegroundColor Cyan
    Write-Host ""
    
    # Define cloud-specific configurations
    $CloudConfigs = @{
        "gcp" = @{
            Name = "Google Cloud Platform"
            Color = "Blue"
            Framework = "fiber"
            Database = @("firestore")
            Auth = @("firebase")
            Email = @("gmail")
            Storage = @("gcp_storage")
            CloudTags = @("google")
            Description = "Firestore + Firebase Auth + Gmail + Cloud Storage"
        }
        "aws" = @{
            Name = "Amazon Web Services"
            Color = "Yellow"
            Framework = "fiber"
            Database = @("postgres")  # RDS PostgreSQL
            Auth = @("jwt_auth")      # JWT for auto-scaling
            Email = @("smtp")         # SES via SMTP
            Storage = @("s3")
            CloudTags = @("aws")
            Description = "RDS PostgreSQL + JWT + SES Email + S3 Storage"
        }
        "azure" = @{
            Name = "Microsoft Azure"
            Color = "Cyan"
            Framework = "fiber"
            Database = @("postgres")     # Azure PostgreSQL
            Auth = @("microsoft")        # Azure AD
            Email = @("microsoftgraph")  # Microsoft Graph
            Storage = @("azure_storage") # Azure Blob Storage
            CloudTags = @("azure", "microsoft")
            Description = "Azure PostgreSQL + Azure AD + Graph API + Blob Storage"
        }
    }
    
    $Config = $CloudConfigs[$CloudProvider]
    
    Write-Host "🌐 HTTP Framework:" -ForegroundColor Cyan
    Write-Host "  • Fiber (high-performance, cloud-optimized)" -ForegroundColor Green
    Write-Host ""
    Write-Host "☁️  Cloud Provider: $($Config.Name)" -ForegroundColor $Config.Color
    Write-Host "  • $($Config.Description)" -ForegroundColor Green
    Write-Host ""
    
    if ($MockMode) {
        Write-Host "🧪 Development Features:" -ForegroundColor Cyan
        Write-Host "  • Mock providers for offline development" -ForegroundColor Yellow
        Write-Host ""
    }
    
    # Build cloud-specific tags
    $BuildTags = @(
        $Config.Framework, "providers_bootstrap"
    )
    
    # Add database tags
    $BuildTags += $Config.Database
    
    # Add auth tags  
    $BuildTags += $Config.Auth
    
    # Add email tags
    $BuildTags += $Config.Email
    
    # Add storage tags
    $BuildTags += $Config.Storage
    
    # Add cloud provider tags
    $BuildTags += $Config.CloudTags
    
    # Add essential fallbacks
    $BuildTags += @("noop")
    
    if ($MockMode) {
        Write-Host "Including mock providers for development..." -ForegroundColor Yellow
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor $Config.Color
    Write-Host "Total components: $($BuildTags.Count)" -ForegroundColor $Config.Color
    Write-Host ""
    
    # Build with cloud-specific tag set
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne $Config.Framework }) -join ','
    
    Write-Host "Executing cloud-native build..." -ForegroundColor Magenta
    Write-Host "Command: .\scripts\build-with-tags.ps1 -Framework $($Config.Framework) -SecondaryTags '$SecondaryTagsString' -Output $Output" -ForegroundColor Gray
    Write-Host ""
    
    # Build arguments with release optimizations
    $BuildArgs = @("-Framework", $Config.Framework, "-SecondaryTags", $SecondaryTagsString, "-Output", $Output)
    
    # Add release-specific linker flags for size optimization
    $ReleaseLdFlags = "-s -w"  # Strip debugging info and symbol tables
    $BuildArgs += @("-LdFlags", $ReleaseLdFlags)
    
    if ($VerboseBuild) {
        $BuildArgs += "-VerboseBuild"
    }
    if ($Race) {
        $BuildArgs += "-Race"
        Write-Host "⚠️  Race detection enabled - binary will be larger" -ForegroundColor Yellow
    }
    
    # Execute cloud-native build
    & ".\scripts\build-with-tags.ps1" @BuildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ Cloud-native build finished!" -ForegroundColor Green
        
        # Show binary info with size comparison
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "✅ Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✅ Binary size: $([math]::Round($BinarySize, 2)) MB (cloud-native optimized)" -ForegroundColor $Config.Color
            
            # Size comparison with enterprise build
            $ComparisonSize = 70  # Typical enterprise build size
            $SizeReduction = [math]::Round(($ComparisonSize - $BinarySize) / $ComparisonSize * 100, 1)
            Write-Host "📊 Size reduction: $SizeReduction% smaller than enterprise builds" -ForegroundColor Green
        }
        
        Write-Host ""
        Write-Host "🚀 $($Config.Name) Deployment Examples:" -ForegroundColor $Config.Color
        Write-Host ""
        
        # Cloud-specific deployment examples
        switch ($CloudProvider) {
            "gcp" {
                Write-Host "  Google Cloud Run:" -ForegroundColor White
                Write-Host "    gcloud run deploy espyna-api \\" -ForegroundColor Gray
                Write-Host "      --image gcr.io/PROJECT_ID/espyna:latest \\" -ForegroundColor Gray
                Write-Host "      --platform managed \\" -ForegroundColor Gray
                Write-Host "      --region us-central1 \\" -ForegroundColor Gray
                Write-Host "      --set-env-vars FIREBASE_PROJECT_ID=PROJECT_ID" -ForegroundColor Gray
                Write-Host ""
                Write-Host "  Google Kubernetes Engine (GKE):" -ForegroundColor White
                Write-Host "    kubectl create deployment espyna-api --image=gcr.io/PROJECT_ID/espyna:latest" -ForegroundColor Gray
                Write-Host "    kubectl expose deployment espyna-api --port=8080 --type=LoadBalancer" -ForegroundColor Gray
            }
            "aws" {
                Write-Host "  AWS ECS Fargate:" -ForegroundColor White
                Write-Host "    aws ecs create-service \\" -ForegroundColor Gray
                Write-Host "      --cluster espyna-cluster \\" -ForegroundColor Gray
                Write-Host "      --service-name espyna-api \\" -ForegroundColor Gray
                Write-Host "      --task-definition espyna-api:1 \\" -ForegroundColor Gray
                Write-Host "      --launch-type FARGATE" -ForegroundColor Gray
                Write-Host ""
                Write-Host "  AWS EKS:" -ForegroundColor White
                Write-Host "    kubectl create deployment espyna-api --image=ACCOUNT.dkr.ecr.REGION.amazonaws.com/espyna:latest" -ForegroundColor Gray
                Write-Host "    kubectl expose deployment espyna-api --port=8080 --type=LoadBalancer" -ForegroundColor Gray
            }
            "azure" {
                Write-Host "  Azure Container Instances:" -ForegroundColor White
                Write-Host "    az container create \\" -ForegroundColor Gray
                Write-Host "      --resource-group espyna-rg \\" -ForegroundColor Gray
                Write-Host "      --name espyna-api \\" -ForegroundColor Gray
                Write-Host "      --image espyna.azurecr.io/espyna:latest \\" -ForegroundColor Gray
                Write-Host "      --ports 8080" -ForegroundColor Gray
                Write-Host ""
                Write-Host "  Azure Kubernetes Service (AKS):" -ForegroundColor White
                Write-Host "    kubectl create deployment espyna-api --image=espyna.azurecr.io/espyna:latest" -ForegroundColor Gray
                Write-Host "    kubectl expose deployment espyna-api --port=8080 --type=LoadBalancer" -ForegroundColor Gray
            }
        }
        
        Write-Host ""
        Write-Host "📋 Required Environment Variables:" -ForegroundColor Cyan
        Write-Host "  # Core Configuration" -ForegroundColor White
        Write-Host "  SERVER_TYPE=$($Config.Framework)" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=8080" -ForegroundColor Gray
        Write-Host ""
        
        # Cloud-specific environment variables
        switch ($CloudProvider) {
            "gcp" {
                Write-Host "  # Google Cloud Configuration" -ForegroundColor White
                Write-Host "  FIREBASE_PROJECT_ID=your-project-id" -ForegroundColor Gray
                Write-Host "  GOOGLE_APPLICATION_CREDENTIALS=path/to/service-key.json" -ForegroundColor Gray
                Write-Host "  GCS_BUCKET_NAME=your-storage-bucket" -ForegroundColor Gray
            }
            "aws" {
                Write-Host "  # AWS Configuration" -ForegroundColor White
                Write-Host "  AWS_REGION=us-east-1" -ForegroundColor Gray
                Write-Host "  DATABASE_URL=postgres://user:pass@rds-endpoint:5432/db" -ForegroundColor Gray
                Write-Host "  JWT_SECRET=your-jwt-secret" -ForegroundColor Gray
                Write-Host "  S3_BUCKET_NAME=your-storage-bucket" -ForegroundColor Gray
                Write-Host "  SMTP_HOST=email-smtp.us-east-1.amazonaws.com  # SES" -ForegroundColor Gray
            }
            "azure" {
                Write-Host "  # Azure Configuration" -ForegroundColor White
                Write-Host "  AZURE_CLIENT_ID=your-client-id" -ForegroundColor Gray
                Write-Host "  AZURE_TENANT_ID=your-tenant-id" -ForegroundColor Gray
                Write-Host "  AZURE_CLIENT_SECRET=your-secret" -ForegroundColor Gray
                Write-Host "  DATABASE_URL=postgres://user:pass@azure-postgres:5432/db" -ForegroundColor Gray
                Write-Host "  AZURE_STORAGE_ACCOUNT=yourstorageaccount" -ForegroundColor Gray
            }
        }
        
        if ($MockMode) {
            Write-Host ""
            Write-Host "  # Development (Mock Mode)" -ForegroundColor White
            Write-Host "  MOCK_MODE=true" -ForegroundColor Gray
            Write-Host "  MOCK_BUSINESS_TYPE=education|fitness_center|office_leasing" -ForegroundColor Gray
        }
        
    } else {
        Write-Host ""
        Write-Host "❌ Build failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
} catch {
    Write-Host "❌ Build script error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "💡 Cloud-Native Benefits:" -ForegroundColor Cyan
Write-Host "   • Single-cloud optimization - no unused provider bloat" -ForegroundColor Gray
Write-Host "   • Managed services integration - reduced operational overhead" -ForegroundColor Gray
Write-Host "   • Container-optimized - fast startup, minimal resource usage" -ForegroundColor Gray
Write-Host "   • Auto-scaling ready - stateless architecture" -ForegroundColor Gray
Write-Host "   • Cloud-native security - platform-managed authentication" -ForegroundColor Gray
Write-Host "   • Cost-effective - pay only for used cloud services" -ForegroundColor Gray
Write-Host ""
Write-Host "🎯 Perfect for:" -ForegroundColor Cyan
Write-Host "   • Cloud-first applications" -ForegroundColor Gray
Write-Host "   • Serverless and container deployments" -ForegroundColor Gray
Write-Host "   • Auto-scaling production workloads" -ForegroundColor Gray
Write-Host "   • Managed cloud service integration" -ForegroundColor Gray
Write-Host "   • Single-cloud architecture strategies" -ForegroundColor Gray