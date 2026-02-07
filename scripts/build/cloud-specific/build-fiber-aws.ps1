#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build Espyna server with Fiber HTTP framework and AWS cloud services ecosystem

.DESCRIPTION
    This script creates a specialized build optimized for AWS deployment with:
    - Fiber HTTP framework for ultra-high performance
    - AWS S3 for scalable object storage
    - PostgreSQL on AWS RDS for managed database
    - JWT authentication for stateless scaling
    - SMTP integration for Amazon SES email services

.PARAMETER Output
    Output binary name. Default: espyna-fiber-aws

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: true

.EXAMPLE
    .\build-fiber-aws.ps1
    Basic build with Fiber + AWS stack

.EXAMPLE
    .\build-fiber-aws.ps1 -VerboseBuild -Race -MockMode:$false
    Production build with verbose output and race detection

.NOTES
    This build configuration is optimized for:
    - AWS cloud-native deployments (ECS, EKS, Lambda)
    - High-performance applications with Fiber framework
    - Scalable object storage with S3
    - Managed PostgreSQL with RDS
    - Cost-effective email with Amazon SES
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-fiber-aws",
    
    [Parameter(Mandatory=$false)]
    [switch]$VerboseBuild,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [bool]$MockMode = $true
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "=== Espyna Fiber + AWS Build ===" -ForegroundColor Cyan
    Write-Host "Building AWS-optimized server with:" -ForegroundColor White
    Write-Host "  • HTTP Framework: Fiber (ultra-high performance)" -ForegroundColor Green
    Write-Host "  • Database: PostgreSQL on AWS RDS (managed)" -ForegroundColor Green
    Write-Host "  • Authentication: JWT (stateless for auto-scaling)" -ForegroundColor Green
    Write-Host "  • Storage: AWS S3 (infinite scalability)" -ForegroundColor Green
    Write-Host "  • Email: SMTP (Amazon SES integration)" -ForegroundColor Green
    Write-Host "  • Deployment: ECS/EKS/Lambda ready" -ForegroundColor Green
    if ($MockMode) {
        Write-Host "  • Mock providers included for testing" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # AWS-optimized build tags
    $BuildTags = @("fiber", "providers_bootstrap", "postgres", "jwt_auth", "aws", "s3", "postgres_migrations")
    if ($MockMode) {
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    # Include essential fallback providers
    $BuildTags += @("local_storage", "noop", "google", "firebase", "firestore", "microsoft")
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    Write-Host ""
    
    # Use the quick-build approach with tested tag combinations
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne "fiber" }) -join ','
    
    Write-Host "Executing: .\scripts\build-with-tags.ps1 -Framework fiber -SecondaryTags '$SecondaryTagsString' -Output $Output" -ForegroundColor Magenta
    Write-Host ""
    
    # Build arguments for the main build script
    $BuildArgs = @("-Framework", "fiber", "-SecondaryTags", $SecondaryTagsString, "-Output", $Output)
    if ($VerboseBuild) {
        $BuildArgs += "-VerboseBuild"
    }
    if ($Race) {
        $BuildArgs += "-Race"
    }
    
    # Execute build using the working build script
    & ".\scripts\build-with-tags.ps1" @BuildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ AWS-optimized build completed!" -ForegroundColor Green
        
        # Show binary info
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "✅ Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✅ Binary size: $([math]::Round($BinarySize, 2)) MB" -ForegroundColor Blue
        }
        
        Write-Host ""
        Write-Host "🚀 AWS Deployment Examples:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Local Development:" -ForegroundColor White
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=education ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  AWS ECS Deployment:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=fiber SERVER_PORT=8080 \\" -ForegroundColor Gray
        Write-Host "    DATABASE_URL=\$RDS_DATABASE_URL \\" -ForegroundColor Gray
        Write-Host "    AWS_REGION=us-east-1 \\" -ForegroundColor Gray
        Write-Host "    S3_BUCKET_NAME=my-app-storage \\" -ForegroundColor Gray
        Write-Host "    JWT_SECRET=\$JWT_SECRET_FROM_SECRETS_MANAGER \\" -ForegroundColor Gray
        Write-Host "    ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  AWS Lambda (with adapter):" -ForegroundColor White
        Write-Host "    AWS_LAMBDA_RUNTIME_API=\$AWS_LAMBDA_RUNTIME_API \\" -ForegroundColor Gray
        Write-Host "    DATABASE_URL=\$RDS_PROXY_ENDPOINT \\" -ForegroundColor Gray
        Write-Host "    ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📋 Environment Variables:" -ForegroundColor Cyan
        Write-Host "  SERVER_TYPE=fiber                    # HTTP framework" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=8080                     # Server port" -ForegroundColor Gray
        Write-Host "  DATABASE_URL=postgres://...          # RDS connection string" -ForegroundColor Gray
        Write-Host "  JWT_SECRET=secret-from-ssm           # JWT signing key (use AWS SSM)" -ForegroundColor Gray
        Write-Host "  AWS_REGION=us-east-1                 # AWS region" -ForegroundColor Gray
        Write-Host "  AWS_ACCESS_KEY_ID=AKIAXXXXX          # AWS credentials (or use IAM roles)" -ForegroundColor Gray
        Write-Host "  AWS_SECRET_ACCESS_KEY=secret         # AWS secret (or use IAM roles)" -ForegroundColor Gray
        Write-Host "  S3_BUCKET_NAME=my-storage-bucket     # S3 bucket for file storage" -ForegroundColor Gray
        Write-Host "  SES_FROM_EMAIL=noreply@example.com   # Amazon SES sender email" -ForegroundColor Gray
        Write-Host "  MOCK_MODE=true                       # Enable mock providers (dev only)" -ForegroundColor Gray
        
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
Write-Host "☁️ AWS + Fiber Stack Benefits:" -ForegroundColor Cyan
Write-Host "   • Ultra-fast HTTP performance with Fiber framework" -ForegroundColor Gray
Write-Host "   • Infinite scalability with AWS S3 object storage" -ForegroundColor Gray
Write-Host "   • Managed PostgreSQL with AWS RDS (automated backups, scaling)" -ForegroundColor Gray
Write-Host "   • Cost-effective email delivery with Amazon SES" -ForegroundColor Gray
Write-Host "   • Stateless JWT authentication for auto-scaling groups" -ForegroundColor Gray
Write-Host "   • Container-ready for ECS, EKS, and Fargate deployments" -ForegroundColor Gray
Write-Host "   • Lambda-compatible for serverless architectures" -ForegroundColor Gray