#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build ultra-lightweight Espyna server for edge computing and resource-constrained environments

.DESCRIPTION
    This script creates the smallest possible build with minimal dependencies:
    - Vanilla HTTP framework (Go standard library only)
    - Mock providers for ultra-light deployment
    - Local file storage (no cloud dependencies)
    - JWT authentication (stateless, no external auth services)
    - Optimized for IoT, edge computing, and embedded systems

.PARAMETER Output
    Output binary name. Default: espyna-minimal-edge

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER IncludePostgres
    Include PostgreSQL support for local database. Default: false

.EXAMPLE
    .\build-minimal-edge.ps1
    Ultra-minimal build for edge deployment

.EXAMPLE
    .\build-minimal-edge.ps1 -IncludePostgres
    Minimal build with optional PostgreSQL support

.NOTES
    This build configuration is optimized for:
    - IoT and edge computing devices
    - Raspberry Pi and ARM-based systems
    - Docker containers with minimal base images
    - Development environments with limited resources
    - Air-gapped or offline deployments
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-minimal-edge",
    
    [Parameter(Mandatory=$false)]
    [switch]$VerboseBuild,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [bool]$IncludePostgres = $false
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "=== Espyna Minimal Edge Build ===" -ForegroundColor Cyan
    Write-Host "Building ultra-lightweight server for edge computing:" -ForegroundColor White
    Write-Host "  • HTTP Framework: Vanilla (Go standard library)" -ForegroundColor Green
    Write-Host "  • Database: Mock in-memory (ultra-fast startup)" -ForegroundColor Green
    Write-Host "  • Authentication: JWT (no external dependencies)" -ForegroundColor Green
    Write-Host "  • Storage: Local filesystem (no cloud services)" -ForegroundColor Green
    Write-Host "  • Email: Mock console logging (no SMTP)" -ForegroundColor Green
    Write-Host "  • Memory: Minimal footprint design" -ForegroundColor Green
    if ($IncludePostgres) {
        Write-Host "  • Optional: PostgreSQL support included" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Minimal build tags - only essential components
    $BuildTags = @("vanilla", "providers_bootstrap", "mock_db", "mock_email", "mock_storage", "local_storage", "jwt_auth", "noop")
    
    if ($IncludePostgres) {
        $BuildTags += @("postgres", "postgres_migrations")
        Write-Host "📊 PostgreSQL support: ENABLED" -ForegroundColor Blue
    } else {
        Write-Host "📊 PostgreSQL support: DISABLED (ultra-minimal mode)" -ForegroundColor Blue
    }
    
    # Note: Deliberately exclude cloud providers to minimize binary size
    Write-Host "☁️  Cloud providers: DISABLED (minimal build)" -ForegroundColor Blue
    Write-Host "📦 Dependencies: MINIMAL (edge computing optimized)" -ForegroundColor Blue
    Write-Host ""
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    Write-Host ""
    
    # Use the quick-build approach with minimal tag set
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne "vanilla" }) -join ','
    
    Write-Host "Executing: .\scripts\build-with-tags.ps1 -Framework vanilla -SecondaryTags '$SecondaryTagsString' -Output $Output" -ForegroundColor Magenta
    Write-Host ""
    
    # Build arguments for the main build script
    $BuildArgs = @("-Framework", "vanilla", "-SecondaryTags", $SecondaryTagsString, "-Output", $Output)
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
        Write-Host "✅ Minimal edge build completed!" -ForegroundColor Green
        
        # Show binary info with size comparison
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "✅ Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✅ Binary size: $([math]::Round($BinarySize, 2)) MB" -ForegroundColor Blue
            
            # Size comparison with other builds
            $FullBuildPath = "build/espyna-server"
            if (Test-Path $FullBuildPath) {
                $FullSize = (Get-Item $FullBuildPath).Length / 1MB
                $SizeReduction = [math]::Round((($FullSize - $BinarySize) / $FullSize) * 100, 1)
                Write-Host "📉 Size reduction: $SizeReduction% smaller than full build" -ForegroundColor Green
            }
        }
        
        Write-Host ""
        Write-Host "🚀 Edge Deployment Examples:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Raspberry Pi Deployment:" -ForegroundColor White
        Write-Host "    # Copy binary to Pi" -ForegroundColor Gray
        Write-Host "    scp $OutputPath pi@raspberrypi.local:/home/pi/" -ForegroundColor Gray
        Write-Host "    # Run on Pi" -ForegroundColor Gray
        Write-Host "    ssh pi@raspberrypi.local" -ForegroundColor Gray
        Write-Host "    chmod +x $Output && ./$Output" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Docker Alpine Container:" -ForegroundColor White
        Write-Host "    # Dockerfile" -ForegroundColor Gray
        Write-Host "    FROM alpine:latest" -ForegroundColor Gray
        Write-Host "    COPY $Output /usr/local/bin/" -ForegroundColor Gray
        Write-Host "    ENTRYPOINT [\"/usr/local/bin/$Output\"]" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  IoT Edge Device:" -ForegroundColor White
        Write-Host "    PORT=3000 JWT_SECRET=edge-device-secret ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Local Development:" -ForegroundColor White
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=fitness_center ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📋 Environment Variables:" -ForegroundColor Cyan
        Write-Host "  SERVER_PORT=8080                     # Server port" -ForegroundColor Gray
        Write-Host "  JWT_SECRET=your-edge-secret          # JWT signing key" -ForegroundColor Gray
        Write-Host "  STORAGE_PATH=/data/uploads           # Local file storage path" -ForegroundColor Gray
        Write-Host "  LOG_LEVEL=info                       # Logging level (debug|info|warn|error)" -ForegroundColor Gray
        Write-Host "  MOCK_MODE=true                       # Use in-memory mock data" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=education         # Mock data type" -ForegroundColor Gray
        if ($IncludePostgres) {
            Write-Host "  DATABASE_URL=postgres://...         # PostgreSQL connection (if enabled)" -ForegroundColor Gray
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
Write-Host "⚡ Minimal Edge Stack Benefits:" -ForegroundColor Cyan
Write-Host "   • Ultra-small binary size for resource-constrained devices" -ForegroundColor Gray
Write-Host "   • Zero cloud dependencies - runs completely offline" -ForegroundColor Gray
Write-Host "   • Instant startup with in-memory mock database" -ForegroundColor Gray
Write-Host "   • Perfect for IoT, edge computing, and embedded systems" -ForegroundColor Gray
Write-Host "   • Docker-optimized for minimal container images" -ForegroundColor Gray
Write-Host "   • ARM-compatible for Raspberry Pi and similar devices" -ForegroundColor Gray
Write-Host "   • Self-contained - no external service dependencies" -ForegroundColor Gray