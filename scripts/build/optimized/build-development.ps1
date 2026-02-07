#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build ultra-lightweight Espyna server for development with mock providers only

.DESCRIPTION
    This script creates the smallest possible build with:
    - Vanilla HTTP framework (Go standard library)
    - Mock database provider (in-memory, no external database required)
    - Mock authentication (test tokens, no auth service required)
    - Mock email provider (console logging, no email service required)
    - Mock storage provider (temporary files, no storage service required)
    - All business type mock data included (education, fitness_center, office_leasing)
    - Zero external dependencies for complete offline development

.PARAMETER Output
    Output binary name. Default: espyna-development

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER AllBusinessTypes
    Include mock data for all business types. Default: true

.EXAMPLE
    .\build-development.ps1
    Ultra-minimal development server with all mock providers

.EXAMPLE
    .\build-development.ps1 -VerboseBuild -Race
    Development build with debugging options

.NOTES
    This build configuration is optimized for:
    - Ultra-small binary size (target: 5-15MB)
    - Instant startup with no external dependencies
    - Complete offline development capability
    - Comprehensive test data for all business scenarios
    - Hot-reload friendly development workflow
    - CI/CD pipeline testing without external services
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-development",
    
    [Parameter(Mandatory=$false)]
    [switch]$VerboseBuild,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [bool]$AllBusinessTypes = $true
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "=== Espyna Development Build ===" -ForegroundColor Magenta
    Write-Host "Building ultra-lightweight server for development:" -ForegroundColor White
    Write-Host "  • Zero external dependencies (100% offline capable)" -ForegroundColor Cyan
    Write-Host "  • Ultra-small binary size (target: 5-15MB)" -ForegroundColor Cyan
    Write-Host "  • Instant startup with comprehensive mock data" -ForegroundColor Cyan
    Write-Host "  • Perfect for development, testing, and CI/CD" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "🌐 HTTP Framework:" -ForegroundColor Cyan
    Write-Host "  • Vanilla (Go standard library - zero dependencies)" -ForegroundColor Green
    Write-Host ""
    Write-Host "🧪 Mock Providers (Development Only):" -ForegroundColor Cyan
    Write-Host "  • Mock Database (in-memory with full business logic)" -ForegroundColor Yellow
    Write-Host "  • Mock Authentication (test tokens and user sessions)" -ForegroundColor Yellow
    Write-Host "  • Mock Email (console logging with realistic templates)" -ForegroundColor Yellow
    Write-Host "  • Mock Storage (temporary files with full API support)" -ForegroundColor Yellow
    Write-Host ""
    if ($AllBusinessTypes) {
        Write-Host "Business Type Support:" -ForegroundColor Cyan
        Write-Host "  • Education (students, teachers, courses, grades)" -ForegroundColor Green
        Write-Host "  • Fitness Center (members, trainers, classes, equipment)" -ForegroundColor Green
        Write-Host "  • Office Leasing (tenants, properties, leases, maintenance)" -ForegroundColor Green
        Write-Host ""
    }
    
    # Development build tags - only mock providers
    $BuildTags = @(
        "vanilla", "providers_bootstrap",
        # Mock providers only - no real external dependencies
        "mock_db", "mock_email", "mock_storage", "mock_auth",
        # Essential system components
        "noop"
    )
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Magenta
    Write-Host "Total components: $($BuildTags.Count) (minimal)" -ForegroundColor Magenta
    Write-Host ""
    
    # Build with development-only tag set
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne "vanilla" }) -join ','
    
    Write-Host "Executing development build..." -ForegroundColor Magenta
    Write-Host "Command: .\scripts\build-with-tags.ps1 -Framework vanilla -SecondaryTags '$SecondaryTagsString' -Output $Output" -ForegroundColor Gray
    Write-Host ""
    
    # Build arguments with development optimizations
    $BuildArgs = @{
        Framework = "vanilla"
        SecondaryTags = $SecondaryTagsString
        Output = $Output
        LdFlags = "-s -w"  # Strip for minimal size
    }
    
    if ($VerboseBuild) {
        $BuildArgs.VerboseBuild = $true
    }
    if ($Race) {
        $BuildArgs.Race = $true
        Write-Host "Race detection enabled - great for development debugging!" -ForegroundColor Yellow
    }
    
    # Execute development build
    & ".\scripts\build-with-tags.ps1" @BuildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "Development build finished!" -ForegroundColor Green
        
        # Show binary info with size comparison
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "Binary size: $([math]::Round($BinarySize, 2)) MB (development-optimized)" -ForegroundColor Magenta
            
            # Size comparison with enterprise build
            $ComparisonSize = 70  # Typical enterprise build size
            $SizeReduction = [math]::Round(($ComparisonSize - $BinarySize) / $ComparisonSize * 100, 1)
            Write-Host "Size reduction: $SizeReduction% smaller than enterprise builds" -ForegroundColor Green
            
            # Compare with minimal API build
            $MinimalSize = 15  # Expected minimal API build size
            if ($BinarySize -lt $MinimalSize) {
                $DevAdvantage = [math]::Round(($MinimalSize - $BinarySize) / $MinimalSize * 100, 1)
                Write-Host "Even smaller: $DevAdvantage% smaller than minimal API builds" -ForegroundColor Cyan
            }
        }
        
        Write-Host ""
        Write-Host "🚀 Development Usage Examples:" -ForegroundColor Magenta
        Write-Host ""
        Write-Host "  Instant Development Server:" -ForegroundColor White
        Write-Host "    ./$Output" -ForegroundColor Gray
        Write-Host "    # No configuration needed - runs immediately with mock data" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Business Type Testing:" -ForegroundColor White
        Write-Host "    MOCK_BUSINESS_TYPE=education ./$Output    # Student/teacher data" -ForegroundColor Gray
        Write-Host "    MOCK_BUSINESS_TYPE=fitness_center ./$Output  # Gym/trainer data" -ForegroundColor Gray
        Write-Host "    MOCK_BUSINESS_TYPE=office_leasing ./$Output  # Tenant/property data" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Development with Debugging:" -ForegroundColor White
        Write-Host "    LOG_LEVEL=debug \\" -ForegroundColor Gray
        Write-Host "    MOCK_MODE=true \\" -ForegroundColor Gray
        Write-Host "    MOCK_BUSINESS_TYPE=education \\" -ForegroundColor Gray
        Write-Host "    ./$Output" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  CI/CD Pipeline Testing:" -ForegroundColor White
        Write-Host "    # Perfect for automated testing - no external dependencies" -ForegroundColor Green
        Write-Host "    docker run --rm -p 8080:8080 espyna-dev:latest" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  API Development and Testing:" -ForegroundColor White
        Write-Host "    # All 40 entities with realistic mock data" -ForegroundColor Green
        Write-Host "    curl http://localhost:8080/api/entities/client" -ForegroundColor Gray
        Write-Host "    curl http://localhost:8080/api/entities/subscription" -ForegroundColor Gray
        Write-Host "    curl http://localhost:8080/api/entities/event" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📋 Environment Variables (All Optional):" -ForegroundColor Cyan
        Write-Host "  # Core Configuration" -ForegroundColor White
        Write-Host "  SERVER_TYPE=vanilla  # Always vanilla for development" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=8080     # HTTP port (default: 8080)" -ForegroundColor Gray
        Write-Host "  LOG_LEVEL=debug      # Verbose logging for development" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Business Type Selection" -ForegroundColor White
        Write-Host "  MOCK_BUSINESS_TYPE=education      # Default business scenario" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=fitness_center # Alternative scenario" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=office_leasing # Alternative scenario" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Development Features" -ForegroundColor White
        Write-Host "  MOCK_MODE=true              # Always true for this build" -ForegroundColor Gray
        Write-Host "  MOCK_DELAY_MS=100           # Add realistic API delays" -ForegroundColor Gray
        Write-Host "  MOCK_ERROR_RATE=0.05        # Simulate 5% error rate for testing" -ForegroundColor Gray
        
    } else {
        Write-Host ""
        Write-Host "Build failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
} catch {
    Write-Host "Build script error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "Development Build Benefits:" -ForegroundColor Cyan
Write-Host "   • Zero external dependencies - runs anywhere instantly" -ForegroundColor Gray
Write-Host "   • Ultra-fast startup - perfect for development iteration" -ForegroundColor Gray
Write-Host "   • Comprehensive mock data - all 40 entities with realistic content" -ForegroundColor Gray
Write-Host "   • Complete business scenarios - education, fitness, office leasing" -ForegroundColor Gray
Write-Host "   • CI/CD friendly - no database setup or external services needed" -ForegroundColor Gray
Write-Host "   • Hot-reload ready - minimal resource usage for development loops" -ForegroundColor Gray
Write-Host "   • Perfect for onboarding - new developers can start immediately" -ForegroundColor Gray
Write-Host ""
Write-Host "Perfect for:" -ForegroundColor Cyan
Write-Host "   • Local development and prototyping" -ForegroundColor Gray
Write-Host "   • API development and testing" -ForegroundColor Gray
Write-Host "   • CI/CD pipeline testing" -ForegroundColor Gray
Write-Host "   • Frontend development with realistic backend" -ForegroundColor Gray
Write-Host "   • Demo and presentation environments" -ForegroundColor Gray
Write-Host "   • New developer onboarding" -ForegroundColor Gray
Write-Host "   • Offline development environments" -ForegroundColor Gray
Write-Host ""
Write-Host "🏃‍♂️ Quick Start:" -ForegroundColor Green
Write-Host "   1. Run: ./$Output" -ForegroundColor White
Write-Host "   2. Open: http://localhost:8080/health" -ForegroundColor White  
Write-Host "   3. Test API: http://localhost:8080/api/entities/client" -ForegroundColor White
Write-Host "   4. That's it! No setup required." -ForegroundColor White