#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build Espyna server with Vanilla HTTP framework and PostgreSQL + Self-hosted services

.DESCRIPTION
    This script creates a specialized build with:
    - Vanilla HTTP framework for lightweight, standard library approach
    - PostgreSQL as the primary database provider
    - Self-hosted authentication with JWT
    - Local file storage and SMTP email
    - Minimal dependencies for containerized deployment

.PARAMETER Output
    Output binary name. Default: espyna-vanilla-postgres

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: true

.EXAMPLE
    .\build-vanilla-postgres.ps1
    Basic build with Vanilla + PostgreSQL stack

.EXAMPLE
    .\build-vanilla-postgres.ps1 -VerboseBuild -Race -MockMode:$false
    Production build with verbose output and race detection

.NOTES
    This build configuration is optimized for:
    - Minimal dependencies and small binary size
    - Self-hosted deployment with Docker/Kubernetes
    - PostgreSQL for reliable ACID transactions
    - JWT-based authentication for stateless scaling
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-vanilla-postgres",
    
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
    Write-Host "=== Espyna Vanilla + PostgreSQL Build ===" -ForegroundColor Cyan
    Write-Host "Building specialized server with:" -ForegroundColor White
    Write-Host "  • HTTP Framework: Vanilla (standard library)" -ForegroundColor Green
    Write-Host "  • Database: PostgreSQL (ACID transactions)" -ForegroundColor Green
    Write-Host "  • Authentication: JWT (stateless)" -ForegroundColor Green
    Write-Host "  • Email Provider: SMTP (configurable)" -ForegroundColor Green
    Write-Host "  • Storage Provider: Local filesystem" -ForegroundColor Green
    if ($MockMode) {
        Write-Host "  • Mock providers included for testing" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Build tags configuration - comprehensive working tag set
    $BuildTags = @("vanilla", "providers_bootstrap", "postgres", "jwt_auth", "postgres_migrations")
    if ($MockMode) {
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    # Always include essential providers for complete functionality
    $BuildTags += @("local_storage", "noop", "google", "firebase", "firestore", "microsoft")
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    
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
        }
        
        Write-Host ""
        Write-Host "Usage Examples:" -ForegroundColor Cyan
        Write-Host "  Development (with mock data):" -ForegroundColor White
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=fitness_center ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Production (self-hosted):" -ForegroundColor White
        Write-Host "    DATABASE_URL='postgres://user:pass@localhost/db' ./$OutputPath" -ForegroundColor Gray
        Write-Host "    JWT_SECRET=your-secret-key ./$OutputPath" -ForegroundColor Gray
        Write-Host "    SMTP_HOST=mail.example.com ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Environment Variables:" -ForegroundColor Cyan
        Write-Host "  SERVER_PORT=8080                    # Server port" -ForegroundColor Gray
        Write-Host "  DATABASE_URL=postgres://...         # PostgreSQL connection" -ForegroundColor Gray
        Write-Host "  JWT_SECRET=secret-key               # JWT signing key" -ForegroundColor Gray
        Write-Host "  SMTP_HOST=mail.example.com          # Email server" -ForegroundColor Gray
        Write-Host "  SMTP_PORT=587                       # Email port" -ForegroundColor Gray
        Write-Host "  STORAGE_PATH=/app/uploads           # File storage path" -ForegroundColor Gray
        Write-Host "  MOCK_MODE=true                      # Enable mock providers" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=fitness_center   # Mock data type" -ForegroundColor Gray
        
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
Write-Host "Vanilla + PostgreSQL Stack Benefits:" -ForegroundColor Cyan
Write-Host "  • Minimal binary size with standard library HTTP" -ForegroundColor Gray
Write-Host "  • Rock-solid ACID transactions with PostgreSQL" -ForegroundColor Gray
Write-Host "  • Stateless scaling with JWT authentication" -ForegroundColor Gray
Write-Host "  • Self-hosted deployment with Docker/K8s" -ForegroundColor Gray
Write-Host "  • No vendor lock-in with open-source stack" -ForegroundColor Gray