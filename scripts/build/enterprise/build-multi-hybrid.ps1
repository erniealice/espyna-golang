#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build Espyna server with multiple HTTP frameworks and hybrid cloud services

.DESCRIPTION
    This script creates a comprehensive build with:
    - All HTTP frameworks: Vanilla, Gin, and Fiber
    - Multiple database providers: PostgreSQL, Firestore, SQL Server
    - Multi-cloud authentication: Firebase, Azure AD, JWT
    - Hybrid email/storage providers: Google, Microsoft, AWS, local
    - Complete flexibility for runtime provider switching

.PARAMETER Output
    Output binary name. Default: espyna-multi-hybrid

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: true

.EXAMPLE
    .\build-multi-hybrid.ps1
    Full-featured build with all frameworks and providers

.EXAMPLE
    .\build-multi-hybrid.ps1 -VerboseBuild -Race
    Production build with all options and debugging

.NOTES
    This build configuration provides maximum flexibility:
    - Runtime switching between HTTP frameworks
    - Multiple database and auth providers
    - Multi-cloud service support
    - Comprehensive testing capabilities
    Warning: This creates a larger binary but offers complete flexibility
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-multi-hybrid",
    
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
    Write-Host "=== Espyna Multi-Framework Hybrid Build ===" -ForegroundColor Cyan
    Write-Host "Building comprehensive server with ALL capabilities:" -ForegroundColor White
    Write-Host ""
    Write-Host "HTTP Frameworks:" -ForegroundColor Cyan
    Write-Host "  • Vanilla (standard library) - minimal, reliable" -ForegroundColor Green
    Write-Host "  • Gin (middleware-rich) - flexible API development" -ForegroundColor Green
    Write-Host "  • Fiber (high-performance) - Express.js-like speed" -ForegroundColor Green
    Write-Host ""
    Write-Host "Database Providers:" -ForegroundColor Cyan
    Write-Host "  • PostgreSQL - ACID transactions, relational" -ForegroundColor Green
    Write-Host "  • Firestore - NoSQL, real-time, cloud-native" -ForegroundColor Green
    Write-Host "  • SQL Server - enterprise, Microsoft ecosystem" -ForegroundColor Green
    Write-Host ""
    Write-Host "Authentication Providers:" -ForegroundColor Cyan
    Write-Host "  • Firebase Auth - Google identity platform" -ForegroundColor Green
    Write-Host "  • Azure AD - Microsoft enterprise auth" -ForegroundColor Green
    Write-Host "  • JWT - stateless, self-hosted tokens" -ForegroundColor Green
    Write-Host ""
    Write-Host "Email/Communication:" -ForegroundColor Cyan
    Write-Host "  • Google Gmail API - G Suite integration" -ForegroundColor Green
    Write-Host "  • Microsoft Graph - Office 365 integration" -ForegroundColor Green
    Write-Host "  • SMTP - traditional email servers" -ForegroundColor Green
    Write-Host ""
    Write-Host "Storage Providers:" -ForegroundColor Cyan
    Write-Host "  • Google Cloud Storage - GCP object storage" -ForegroundColor Green
    Write-Host "  • Azure Blob Storage - Azure cloud storage" -ForegroundColor Green
    Write-Host "  • AWS S3 - Amazon object storage" -ForegroundColor Green
    Write-Host "  • Local Filesystem - self-hosted storage" -ForegroundColor Green
    if ($MockMode) {
        Write-Host ""
        Write-Host "  • Mock providers included for comprehensive testing" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Build tags configuration - include everything
    $BuildTags = @(
        # HTTP Frameworks
        "vanilla", "gin", "fiber",
        # Database providers
        "postgres", "firestore", "sqlserver",
        # Auth providers
        "firebase", "jwt", "microsoft",
        # Cloud providers
        "google", "azure", "aws",
        # Storage providers
        "gcp_storage", "azure_storage", "s3_storage", "local_storage",
        # Email providers
        "microsoft_graph", "smtp"
    )
    
    if ($MockMode) {
        $BuildTags += @("mock_database", "mock_auth", "mock_email", "mock_storage")
    }
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    Write-Host ""
    
    # Prepare build arguments
    $BuildArgs = @()
    $BuildArgs += "-tags", ($BuildTags -join ',')
    
    if ($VerboseBuild) {
        $BuildArgs += "-v"
        Write-Host "Verbose build enabled" -ForegroundColor Yellow
    }
    
    if ($Race) {
        $BuildArgs += "-race"
        Write-Host "Race detection enabled" -ForegroundColor Yellow
    }
    
    # Ensure build directory exists
    $BuildDir = "build"
    if (-not (Test-Path $BuildDir)) {
        New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null
        Write-Host "Created build directory: $BuildDir" -ForegroundColor Blue
    }
    
    # Set output path
    $OutputPath = "$BuildDir/$Output"
    $BuildArgs += "-o", $OutputPath
    $BuildArgs += "./cmd/server"
    
    Write-Host "Executing: go build $($BuildArgs -join ' ')" -ForegroundColor Magenta
    Write-Host ""
    
    # Execute build
    $AllArgs = @("build") + $BuildArgs
    $Process = Start-Process -FilePath "go" -ArgumentList $AllArgs -Wait -PassThru -NoNewWindow
    
    if ($Process.ExitCode -eq 0) {
        Write-Host "✓ Build completed successfully!" -ForegroundColor Green
        Write-Host "✓ Binary created: $OutputPath" -ForegroundColor Green
        
        if (Test-Path $OutputPath) {
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✓ Binary size: $([math]::Round($BinarySize, 2)) MB" -ForegroundColor Blue
            Write-Host "  Note: Larger binary due to comprehensive provider support" -ForegroundColor Yellow
        }
        
        Write-Host ""
        Write-Host "Runtime Configuration Examples:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Multi-framework development:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=multi FRAMEWORK_VANILLA_ENABLED=true" -ForegroundColor Gray
        Write-Host "    FRAMEWORK_GIN_ENABLED=true FRAMEWORK_FIBER_ENABLED=true ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Google Cloud Stack:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=fiber DATABASE_PROVIDER=firestore" -ForegroundColor Gray
        Write-Host "    AUTH_PROVIDER=firebase EMAIL_PROVIDER=google" -ForegroundColor Gray
        Write-Host "    STORAGE_PROVIDER=gcs ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Microsoft Enterprise Stack:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=gin DATABASE_PROVIDER=sqlserver" -ForegroundColor Gray
        Write-Host "    AUTH_PROVIDER=azure EMAIL_PROVIDER=microsoft_graph" -ForegroundColor Gray
        Write-Host "    STORAGE_PROVIDER=azure_blob ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Self-hosted Stack:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=vanilla DATABASE_PROVIDER=postgres" -ForegroundColor Gray
        Write-Host "    AUTH_PROVIDER=jwt EMAIL_PROVIDER=smtp" -ForegroundColor Gray
        Write-Host "    STORAGE_PROVIDER=local ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Hybrid Cloud:" -ForegroundColor White
        Write-Host "    DATABASE_PROVIDER=postgres AUTH_PROVIDER=firebase" -ForegroundColor Gray
        Write-Host "    EMAIL_PROVIDER=microsoft_graph STORAGE_PROVIDER=s3 ./$OutputPath" -ForegroundColor Gray
        
    } else {
        Write-Host "✗ Build failed with exit code: $($Process.ExitCode)" -ForegroundColor Red
        exit $Process.ExitCode
    }
    
} catch {
    Write-Host "✗ Build script error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "Multi-Hybrid Stack Benefits:" -ForegroundColor Cyan
Write-Host "  • Complete flexibility - choose providers at runtime" -ForegroundColor Gray
Write-Host "  • Multi-cloud deployment strategies" -ForegroundColor Gray
Write-Host "  • Gradual migration between providers" -ForegroundColor Gray
Write-Host "  • A/B testing different technology stacks" -ForegroundColor Gray
Write-Host "  • Comprehensive testing with all provider combinations" -ForegroundColor Gray
Write-Host "  • Future-proof architecture with provider agnostic design" -ForegroundColor Gray