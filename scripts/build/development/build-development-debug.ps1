#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build development-optimized Espyna server with enhanced debugging and testing features

.DESCRIPTION
    This script creates a developer-friendly build optimized for local development with:
    - Gin HTTP framework for hot-reload and middleware flexibility
    - Comprehensive mock providers for offline development
    - Enhanced logging and debugging capabilities
    - Race condition detection enabled by default
    - All business types and test data included
    - Multiple provider support for testing integration scenarios

.PARAMETER Output
    Output binary name. Default: espyna-dev-debug

.PARAMETER VerboseBuild
    Enable verbose build output. Default: true for development

.PARAMETER Race
    Enable race condition detection. Default: true for development

.PARAMETER SymbolTable
    Include debugging symbols. Default: true

.EXAMPLE
    .\build-development-debug.ps1
    Development build with all debugging features

.EXAMPLE
    .\build-development-debug.ps1 -SymbolTable:$false
    Development build without debug symbols (smaller size)

.NOTES
    This build configuration is optimized for:
    - Local development and testing
    - API development and debugging
    - Integration testing with multiple providers
    - Hot-reload development workflows
    - Comprehensive test data scenarios
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-dev-debug",
    
    [Parameter(Mandatory=$false)]
    [bool]$VerboseBuild = $true,
    
    [Parameter(Mandatory=$false)]
    [bool]$Race = $true,
    
    [Parameter(Mandatory=$false)]
    [bool]$SymbolTable = $true
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "=== Espyna Development Debug Build ===" -ForegroundColor Cyan
    Write-Host "Building developer-optimized server with:" -ForegroundColor White
    Write-Host ""
    Write-Host "🛠️  Development Features:" -ForegroundColor Cyan
    Write-Host "  • HTTP Framework: Gin (hot-reload friendly)" -ForegroundColor Green
    Write-Host "  • Mock Providers: ALL included (offline development)" -ForegroundColor Green
    Write-Host "  • Business Types: ALL test data scenarios" -ForegroundColor Green
    Write-Host "  • Debug Symbols: $(if($SymbolTable){'ENABLED'}else{'DISABLED'})" -ForegroundColor $(if($SymbolTable){'Green'}else{'Yellow'})
    Write-Host "  • Race Detection: $(if($Race){'ENABLED'}else{'DISABLED'})" -ForegroundColor $(if($Race){'Green'}else{'Yellow'})
    Write-Host "  • Verbose Logging: $(if($VerboseBuild){'ENABLED'}else{'DISABLED'})" -ForegroundColor $(if($VerboseBuild){'Green'}else{'Yellow'})
    Write-Host ""
    Write-Host "🧪 Testing Capabilities:" -ForegroundColor Cyan
    Write-Host "  • Multiple provider combinations for integration testing" -ForegroundColor Green
    Write-Host "  • Business type switching at runtime" -ForegroundColor Green
    Write-Host "  • Mock data for all 40+ business entities" -ForegroundColor Green
    Write-Host "  • API endpoint testing across all domains" -ForegroundColor Green
    Write-Host ""
    
    # Development-optimized build tags - prioritize mock providers and debugging
    $BuildTags = @(
        "gin", "providers_bootstrap",
        # Mock providers first (primary for development)
        "mock_db", "mock_email", "mock_storage", 
        # Essential real providers for integration testing
        "postgres", "postgres_migrations", "local_storage", "jwt_auth", 
        # Cloud providers for testing cloud integrations
        "firebase", "firestore", "google", "gmail", "gcp_storage",
        "microsoft", "microsoftgraph", "aws", "s3",
        # Fallback providers
        "noop"
    )
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    Write-Host "Development mode: ENABLED" -ForegroundColor Green
    Write-Host ""
    
    # Use the quick-build approach with development-optimized tags
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne "gin" }) -join ','
    
    # Prepare build command with development-specific flags
    $BuildCommand = ".\scripts\build-with-tags.ps1 -Framework gin -SecondaryTags '$SecondaryTagsString' -Output $Output"
    if ($VerboseBuild) {
        $BuildCommand += " -VerboseBuild"
    }
    if ($Race) {
        $BuildCommand += " -Race" 
    }
    
    Write-Host "Executing: $BuildCommand" -ForegroundColor Magenta
    Write-Host ""
    
    # Build arguments for the main build script
    $BuildArgs = @("-Framework", "gin", "-SecondaryTags", $SecondaryTagsString, "-Output", $Output)
    if ($VerboseBuild) {
        $BuildArgs += "-VerboseBuild"
    }
    if ($Race) {
        $BuildArgs += "-Race"
    }
    if ($SymbolTable) {
        # Add debug symbols for better debugging experience
        $BuildArgs += "-LdFlags", "-X main.buildMode=development"
    }
    
    # Execute build using the working build script
    & ".\scripts\build-with-tags.ps1" @BuildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ Development debug build completed!" -ForegroundColor Green
        
        # Show binary info
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "✅ Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✅ Binary size: $([math]::Round($BinarySize, 2)) MB $(if($SymbolTable){'(includes debug symbols)'}else{''})" -ForegroundColor Blue
        }
        
        Write-Host ""
        Write-Host "🚀 Development Usage Examples:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Quick Start (Mock Data):" -ForegroundColor White
        Write-Host "    MOCK_MODE=true LOG_LEVEL=debug ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Test Different Business Types:" -ForegroundColor White
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=education ./$OutputPath" -ForegroundColor Gray
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=fitness_center ./$OutputPath" -ForegroundColor Gray
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=office_leasing ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  API Testing with Real Providers:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=gin DATABASE_PROVIDER=postgres \\" -ForegroundColor Gray
        Write-Host "    DATABASE_URL=postgres://localhost:5432/dev_db ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Multi-Provider Integration Testing:" -ForegroundColor White
        Write-Host "    DATABASE_PROVIDER=postgres AUTH_PROVIDER=jwt \\" -ForegroundColor Gray
        Write-Host "    EMAIL_PROVIDER=mock STORAGE_PROVIDER=local ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Hot Development with File Watching:" -ForegroundColor White
        Write-Host "    # Use with air or similar file watcher" -ForegroundColor Gray
        Write-Host "    air -c .air.toml" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📋 Development Environment Variables:" -ForegroundColor Cyan
        Write-Host "  # Development Settings" -ForegroundColor White
        Write-Host "  SERVER_TYPE=gin                      # Use Gin for development" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=3000                     # Development port" -ForegroundColor Gray  
        Write-Host "  LOG_LEVEL=debug                      # Verbose logging" -ForegroundColor Gray
        Write-Host "  GIN_MODE=debug                       # Gin debug mode" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Mock Data Configuration" -ForegroundColor White
        Write-Host "  MOCK_MODE=true                       # Enable mock providers" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=education         # Business scenario" -ForegroundColor Gray
        Write-Host "  MOCK_USER_COUNT=50                   # Number of mock users" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Database Development" -ForegroundColor White
        Write-Host "  DATABASE_PROVIDER=mock               # For pure offline dev" -ForegroundColor Gray
        Write-Host "  DATABASE_URL=postgres://localhost:5432/testdb  # For DB integration testing" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # API Testing" -ForegroundColor White
        Write-Host "  CORS_ENABLED=true                    # Enable CORS for frontend dev" -ForegroundColor Gray
        Write-Host "  API_TIMEOUT=30s                      # Extended timeouts for debugging" -ForegroundColor Gray
        Write-Host ""
        Write-Host "🔧 Debugging Tips:" -ForegroundColor Cyan
        Write-Host "  • Use LOG_LEVEL=debug for verbose output" -ForegroundColor Gray
        Write-Host "  • Mock providers allow offline development" -ForegroundColor Gray
        Write-Host "  • Switch business types to test different data scenarios" -ForegroundColor Gray
        Write-Host "  • Race detection helps catch concurrency issues early" -ForegroundColor Gray
        Write-Host "  • Use curl or Postman to test API endpoints" -ForegroundColor Gray
        if ($SymbolTable) {
            Write-Host "  • Debug symbols included for GDB/Delve debugging" -ForegroundColor Gray
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
Write-Host "🎯 Development Build Benefits:" -ForegroundColor Cyan
Write-Host "   • Optimized for rapid development and testing cycles" -ForegroundColor Gray
Write-Host "   • Comprehensive mock providers for offline development" -ForegroundColor Gray
Write-Host "   • Multiple business type scenarios for thorough testing" -ForegroundColor Gray
Write-Host "   • Race condition detection prevents concurrency bugs" -ForegroundColor Gray
Write-Host "   • Enhanced logging for debugging complex issues" -ForegroundColor Gray
Write-Host "   • Provider switching allows integration testing scenarios" -ForegroundColor Gray
Write-Host "   • Hot-reload compatible with development tools" -ForegroundColor Gray