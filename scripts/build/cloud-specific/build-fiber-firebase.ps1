#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build Espyna server with Fiber HTTP framework and Firebase ecosystem (Firestore + Firebase Auth)

.DESCRIPTION
    This script creates a specialized build with:
    - Fiber HTTP framework for high-performance web serving
    - Firestore as the primary database provider
    - Firebase Authentication for user management
    - Google services integration (email, storage, etc.)

.PARAMETER Output
    Output binary name. Default: espyna-fiber-firebase

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: true

.EXAMPLE
    .\build-fiber-firebase.ps1
    Basic build with Fiber + Firebase stack

.EXAMPLE
    .\build-fiber-firebase.ps1 -VerboseBuild -Race -MockMode:$false
    Production build with verbose output and race detection

.NOTES
    This build configuration is optimized for:
    - High-performance HTTP serving with Fiber
    - Cloud-native deployment with Firebase/GCP
    - Modern authentication with Firebase Auth
    - Scalable NoSQL database with Firestore
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-fiber-firebase",
    
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
    Write-Host "=== Espyna Fiber + Firebase Build ===" -ForegroundColor Cyan
    Write-Host "Building specialized server with:" -ForegroundColor White
    Write-Host "  • HTTP Framework: Fiber (high-performance)" -ForegroundColor Green
    Write-Host "  • Database: Firestore (NoSQL cloud)" -ForegroundColor Green
    Write-Host "  • Authentication: Firebase Auth" -ForegroundColor Green
    Write-Host "  • Email Provider: Google Gmail API" -ForegroundColor Green
    Write-Host "  • Storage Provider: Google Cloud Storage" -ForegroundColor Green
    if ($MockMode) {
        Write-Host "  • Mock providers included for testing" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Build tags configuration - comprehensive working tag set
    $BuildTags = @("fiber", "providers_bootstrap", "firestore", "firebase", "google", "gmail", "gcp_storage", "postgres_migrations")
    if ($MockMode) {
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    # Always include essential providers for complete functionality
    $BuildTags += @("local_storage", "noop", "postgres")
    
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
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=education ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Production (Firebase services):" -ForegroundColor White
        Write-Host "    GOOGLE_APPLICATION_CREDENTIALS=path/to/service-key.json ./$OutputPath" -ForegroundColor Gray
        Write-Host "    FIREBASE_PROJECT_ID=your-project-id ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Environment Variables:" -ForegroundColor Cyan
        Write-Host "  SERVER_PORT=8080                    # Server port" -ForegroundColor Gray
        Write-Host "  FIREBASE_PROJECT_ID=project-id      # Firebase project" -ForegroundColor Gray
        Write-Host "  FIRESTORE_EMULATOR_HOST=localhost:8080  # Development emulator" -ForegroundColor Gray
        Write-Host "  MOCK_MODE=true                       # Enable mock providers" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=education         # Mock data type" -ForegroundColor Gray
        
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
Write-Host "Fiber + Firebase Stack Benefits:" -ForegroundColor Cyan
Write-Host "  • Ultra-fast HTTP performance with Fiber framework" -ForegroundColor Gray
Write-Host "  • Serverless-ready with Firebase cloud services" -ForegroundColor Gray
Write-Host "  • Real-time database capabilities with Firestore" -ForegroundColor Gray
Write-Host "  • Built-in authentication and user management" -ForegroundColor Gray
Write-Host "  • Auto-scaling and global CDN with Google Cloud" -ForegroundColor Gray