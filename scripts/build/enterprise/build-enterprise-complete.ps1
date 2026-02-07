#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build enterprise-grade Espyna server with comprehensive business features and integrations

.DESCRIPTION
    This script creates a full-featured enterprise build with:
    - Gin HTTP framework for flexible middleware and API development
    - Multiple database support (PostgreSQL primary, with Firestore backup)
    - Multi-provider authentication (Azure AD, Firebase, JWT fallback)
    - Enterprise email integration (Microsoft Graph, Google Workspace)
    - Multi-cloud storage support (Azure Blob, Google Cloud, AWS S3)
    - Enhanced security, monitoring, and compliance features

.PARAMETER Output
    Output binary name. Default: espyna-enterprise-complete

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: true

.EXAMPLE
    .\build-enterprise-complete.ps1
    Full enterprise build with all providers

.EXAMPLE
    .\build-enterprise-complete.ps1 -VerboseBuild -Race -MockMode:$false
    Production enterprise build with debugging

.NOTES
    This build configuration is optimized for:
    - Large enterprise organizations
    - Multi-cloud hybrid deployments
    - High-availability production systems
    - Comprehensive integration requirements
    - Advanced security and compliance needs
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-enterprise-complete",
    
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
    Write-Host "=== Espyna Enterprise Complete Build ===" -ForegroundColor Cyan
    Write-Host "Building full-featured enterprise server with:" -ForegroundColor White
    Write-Host ""
    Write-Host "🌐 HTTP Framework:" -ForegroundColor Cyan
    Write-Host "  • Gin (enterprise API development with middleware)" -ForegroundColor Green
    Write-Host ""
    Write-Host "🗄️  Database Providers:" -ForegroundColor Cyan
    Write-Host "  • PostgreSQL (primary enterprise database)" -ForegroundColor Green
    Write-Host "  • Firestore (cloud-native backup/secondary)" -ForegroundColor Green
    Write-Host "  • Migration support (database versioning)" -ForegroundColor Green
    Write-Host ""
    Write-Host "🔐 Authentication Providers:" -ForegroundColor Cyan
    Write-Host "  • Azure Active Directory (enterprise SSO)" -ForegroundColor Green
    Write-Host "  • Firebase Auth (modern web/mobile)" -ForegroundColor Green
    Write-Host "  • JWT (stateless fallback)" -ForegroundColor Green
    Write-Host ""
    Write-Host "📧 Email & Communication:" -ForegroundColor Cyan
    Write-Host "  • Microsoft Graph API (Office 365, Teams integration)" -ForegroundColor Green
    Write-Host "  • Google Workspace API (Gmail, Calendar)" -ForegroundColor Green
    Write-Host ""
    Write-Host "☁️  Storage Providers:" -ForegroundColor Cyan
    Write-Host "  • Azure Blob Storage (Microsoft ecosystem)" -ForegroundColor Green
    Write-Host "  • Google Cloud Storage (Google ecosystem)" -ForegroundColor Green
    Write-Host "  • AWS S3 (Amazon ecosystem)" -ForegroundColor Green
    Write-Host "  • Local storage (on-premises fallback)" -ForegroundColor Green
    if ($MockMode) {
        Write-Host ""
        Write-Host "🧪 Development Features:" -ForegroundColor Cyan
        Write-Host "  • Mock providers for comprehensive testing" -ForegroundColor Yellow
        Write-Host "  • Multi-business-type support" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Enterprise-complete build tags - include everything
    $BuildTags = @(
        "gin", "providers_bootstrap",
        # Database providers
        "postgres", "firestore", "postgres_migrations",
        # Authentication providers  
        "firebase", "microsoft", "jwt_auth",
        # Cloud service providers
        "google", "aws", "azure",
        # Storage providers
        "gcp_storage", "s3", "local_storage", 
        # Email providers
        "gmail", "microsoftgraph",
        # Essential fallbacks
        "noop"
    )
    
    if ($MockMode) {
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    Write-Host "Total providers: $($BuildTags.Count)" -ForegroundColor Blue
    Write-Host ""
    
    # Use the quick-build approach with comprehensive tag set
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne "gin" }) -join ','
    
    Write-Host "Executing: .\scripts\build-with-tags.ps1 -Framework gin -SecondaryTags '$SecondaryTagsString' -Output $Output" -ForegroundColor Magenta
    Write-Host ""
    
    # Build arguments for the main build script
    $BuildArgs = @("-Framework", "gin", "-SecondaryTags", $SecondaryTagsString, "-Output", $Output)
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
        Write-Host "✅ Enterprise complete build finished!" -ForegroundColor Green
        
        # Show binary info
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "✅ Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✅ Binary size: $([math]::Round($BinarySize, 2)) MB (comprehensive feature set)" -ForegroundColor Blue
        }
        
        Write-Host ""
        Write-Host "🏢 Enterprise Deployment Scenarios:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Microsoft Enterprise Stack:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=gin \\" -ForegroundColor Gray
        Write-Host "    DATABASE_PROVIDER=postgres \\" -ForegroundColor Gray
        Write-Host "    AUTH_PROVIDER=microsoft \\" -ForegroundColor Gray
        Write-Host "    EMAIL_PROVIDER=microsoftgraph \\" -ForegroundColor Gray  
        Write-Host "    STORAGE_PROVIDER=azure_blob \\" -ForegroundColor Gray
        Write-Host "    ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Google Workspace Stack:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=gin \\" -ForegroundColor Gray
        Write-Host "    DATABASE_PROVIDER=firestore \\" -ForegroundColor Gray
        Write-Host "    AUTH_PROVIDER=firebase \\" -ForegroundColor Gray
        Write-Host "    EMAIL_PROVIDER=gmail \\" -ForegroundColor Gray
        Write-Host "    STORAGE_PROVIDER=gcs \\" -ForegroundColor Gray
        Write-Host "    ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Multi-Cloud Hybrid:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=gin \\" -ForegroundColor Gray
        Write-Host "    DATABASE_PROVIDER=postgres \\" -ForegroundColor Gray
        Write-Host "    AUTH_PROVIDER=microsoft \\" -ForegroundColor Gray
        Write-Host "    EMAIL_PROVIDER=gmail \\" -ForegroundColor Gray
        Write-Host "    STORAGE_PROVIDER=s3 \\" -ForegroundColor Gray
        Write-Host "    ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  High-Availability Setup:" -ForegroundColor White
        Write-Host "    SERVER_TYPE=gin \\" -ForegroundColor Gray
        Write-Host "    DATABASE_PROVIDER=postgres \\" -ForegroundColor Gray
        Write-Host "    DATABASE_BACKUP_PROVIDER=firestore \\" -ForegroundColor Gray
        Write-Host "    AUTH_PROVIDER=microsoft \\" -ForegroundColor Gray
        Write-Host "    AUTH_FALLBACK_PROVIDER=jwt \\" -ForegroundColor Gray
        Write-Host "    ./$OutputPath" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📋 Key Environment Variables:" -ForegroundColor Cyan
        Write-Host "  # Core Configuration" -ForegroundColor White
        Write-Host "  SERVER_TYPE=gin" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=8080" -ForegroundColor Gray
        Write-Host "  LOG_LEVEL=info" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Provider Selection (runtime switchable)" -ForegroundColor White
        Write-Host "  DATABASE_PROVIDER=postgres|firestore" -ForegroundColor Gray
        Write-Host "  AUTH_PROVIDER=microsoft|firebase|jwt" -ForegroundColor Gray
        Write-Host "  EMAIL_PROVIDER=microsoftgraph|gmail" -ForegroundColor Gray
        Write-Host "  STORAGE_PROVIDER=azure_blob|gcs|s3|local" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Microsoft Integration" -ForegroundColor White
        Write-Host "  AZURE_CLIENT_ID=your-client-id" -ForegroundColor Gray
        Write-Host "  AZURE_TENANT_ID=your-tenant-id" -ForegroundColor Gray
        Write-Host "  AZURE_CLIENT_SECRET=your-secret" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Google Integration" -ForegroundColor White
        Write-Host "  GOOGLE_APPLICATION_CREDENTIALS=path/to/service-key.json" -ForegroundColor Gray
        Write-Host "  FIREBASE_PROJECT_ID=your-project-id" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Database Connections" -ForegroundColor White
        Write-Host "  DATABASE_URL=postgres://..." -ForegroundColor Gray
        Write-Host "  FIRESTORE_PROJECT_ID=backup-project" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Development/Testing" -ForegroundColor White
        Write-Host "  MOCK_MODE=true" -ForegroundColor Gray
        Write-Host "  MOCK_BUSINESS_TYPE=office_leasing" -ForegroundColor Gray
        
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
Write-Host "🏆 Enterprise Complete Stack Benefits:" -ForegroundColor Cyan
Write-Host "   • Maximum flexibility - all providers included" -ForegroundColor Gray
Write-Host "   • Runtime provider switching - no recompilation needed" -ForegroundColor Gray
Write-Host "   • Multi-cloud deployment support - avoid vendor lock-in" -ForegroundColor Gray
Write-Host "   • Enterprise SSO integration - Azure AD, Google Workspace" -ForegroundColor Gray
Write-Host "   • High availability - primary + backup provider configurations" -ForegroundColor Gray
Write-Host "   • Comprehensive testing - mock providers for all services" -ForegroundColor Gray
Write-Host "   • Future-proof architecture - easy to add new providers" -ForegroundColor Gray