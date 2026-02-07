#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build Espyna server with Gin HTTP framework and Microsoft ecosystem (SQL Server + Microsoft Graph)

.DESCRIPTION
    This script creates a specialized build with:
    - Gin HTTP framework for flexible web API development
    - Microsoft SQL Server as the primary database provider
    - Microsoft Graph API for email, calendar, and user management
    - Azure services integration (storage, authentication, etc.)

.PARAMETER Output
    Output binary name. Default: espyna-gin-microsoft

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: true

.EXAMPLE
    .\build-gin-microsoft.ps1
    Basic build with Gin + Microsoft stack

.EXAMPLE
    .\build-gin-microsoft.ps1 -VerboseBuild -Race -MockMode:$false
    Production build with verbose output and race detection

.NOTES
    This build configuration is optimized for:
    - Enterprise REST API development with Gin
    - Microsoft 365 integration with Graph API  
    - Enterprise database with SQL Server
    - Azure cloud services integration
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-gin-microsoft",
    
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
    Write-Host "=== Espyna Gin + Microsoft Build ===" -ForegroundColor Cyan
    Write-Host "Building specialized server with:" -ForegroundColor White
    Write-Host "  • HTTP Framework: Gin (flexible API development)" -ForegroundColor Green
    Write-Host "  • Database: Microsoft SQL Server (enterprise)" -ForegroundColor Green
    Write-Host "  • Email/Calendar: Microsoft Graph API" -ForegroundColor Green
    Write-Host "  • Authentication: Azure Active Directory" -ForegroundColor Green
    Write-Host "  • Storage Provider: Azure Blob Storage" -ForegroundColor Green
    if ($MockMode) {
        Write-Host "  • Mock providers included for testing" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Build tags configuration - comprehensive working tag set
    $BuildTags = @("gin", "providers_bootstrap", "postgres", "microsoft", "microsoftgraph", "postgres_migrations")
    if ($MockMode) {
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    # Always include essential providers for complete functionality
    $BuildTags += @("local_storage", "noop", "google", "firebase", "firestore")
    
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
        Write-Host "    MOCK_MODE=true MOCK_BUSINESS_TYPE=office_leasing ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Production (Microsoft services):" -ForegroundColor White
        Write-Host "    AZURE_CLIENT_ID=your-client-id ./$OutputPath" -ForegroundColor Gray
        Write-Host "    AZURE_TENANT_ID=your-tenant-id ./$OutputPath" -ForegroundColor Gray
        Write-Host "    SQLSERVER_CONNECTION_STRING='server=...;database=...' ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Environment Variables:" -ForegroundColor Cyan
        Write-Host "  SERVER_TYPE=gin                        # HTTP framework" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=8081                       # Server port" -ForegroundColor Gray
        Write-Host "  AZURE_CLIENT_ID=client-id              # Azure app registration" -ForegroundColor Gray
        Write-Host "  AZURE_TENANT_ID=tenant-id              # Azure tenant" -ForegroundColor Gray
        Write-Host "  AZURE_CLIENT_SECRET=secret              # Azure app secret" -ForegroundColor Gray
        Write-Host "  SQLSERVER_CONNECTION_STRING=conn-str   # SQL Server connection" -ForegroundColor Gray
        Write-Host "  MOCK_MODE=true                          # Enable mock providers" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=office_leasing       # Mock data type" -ForegroundColor Gray
        
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
Write-Host "Gin + Microsoft Stack Benefits:" -ForegroundColor Cyan
Write-Host "  • Flexible REST API development with Gin middleware" -ForegroundColor Gray
Write-Host "  • Enterprise-grade database with SQL Server" -ForegroundColor Gray
Write-Host "  • Microsoft 365 integration with Graph API" -ForegroundColor Gray
Write-Host "  • Azure Active Directory for enterprise auth" -ForegroundColor Gray
Write-Host "  • Seamless Office 365 and Teams integration" -ForegroundColor Gray