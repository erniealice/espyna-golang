#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build minimal Espyna API server optimized for small binary size and fast deployment

.DESCRIPTION
    This script creates a lightweight build with:
    - Vanilla HTTP framework (Go standard library - no external web framework)
    - PostgreSQL database only (most common production database)
    - JWT authentication (stateless, no external auth provider dependencies)
    - SMTP email (configurable, no cloud provider dependencies)
    - Local filesystem storage (no cloud storage dependencies)
    - Minimal dependencies for maximum performance and reliability

.PARAMETER Output
    Output binary name. Default: espyna-minimal-api

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER MockMode
    Include mock providers for testing. Default: false (production-focused)

.EXAMPLE
    .\build-minimal-api.ps1
    Minimal production-ready API server

.EXAMPLE
    .\build-minimal-api.ps1 -MockMode:$true -VerboseBuild
    Minimal build with mock providers for development

.NOTES
    This build configuration is optimized for:
    - Small binary size (target: 10-20MB vs 70-80MB enterprise builds)
    - Fast startup time and minimal memory usage
    - Self-hosted deployments and Docker containers
    - Cost-conscious cloud deployments
    - IoT and edge computing environments
    - Development environments with minimal resource usage
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-minimal-api",
    
    [Parameter(Mandatory=$false)]
    [switch]$VerboseBuild,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [bool]$MockMode = $false
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "=== Espyna Minimal API Build ===" -ForegroundColor Green
    Write-Host "Building lightweight server optimized for:" -ForegroundColor White
    Write-Host "  • Small binary size (target: 10-20MB)" -ForegroundColor Cyan
    Write-Host "  • Fast startup and minimal memory usage" -ForegroundColor Cyan
    Write-Host "  • Self-hosted and container deployments" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "🌐 HTTP Framework:" -ForegroundColor Cyan
    Write-Host "  • Vanilla (Go standard library - zero external dependencies)" -ForegroundColor Green
    Write-Host ""
    Write-Host "🗄️  Database:" -ForegroundColor Cyan
    Write-Host "  • PostgreSQL (single, reliable ACID database)" -ForegroundColor Green
    Write-Host "  • Database migrations support" -ForegroundColor Green
    Write-Host ""
    Write-Host "🔐 Authentication:" -ForegroundColor Cyan
    Write-Host "  • JWT (stateless tokens, horizontally scalable)" -ForegroundColor Green
    Write-Host ""
    Write-Host "📧 Email:" -ForegroundColor Cyan
    Write-Host "  • SMTP (configurable with any email service)" -ForegroundColor Green
    Write-Host ""
    Write-Host "💾 Storage:" -ForegroundColor Cyan
    Write-Host "  • Local filesystem (self-contained, no cloud dependencies)" -ForegroundColor Green
    if ($MockMode) {
        Write-Host ""
        Write-Host "🧪 Development Features:" -ForegroundColor Cyan
        Write-Host "  • Mock providers for offline development" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Minimal build tags - only essential components
    $BuildTags = @(
        "vanilla", "providers_bootstrap",
        # Single database
        "postgres", "postgres_migrations",
        # Single auth method
        "jwt_auth",
        # Simple email and storage
        "smtp", "local_storage",
        # Essential fallback
        "noop"
    )
    
    if ($MockMode) {
        Write-Host "Including mock providers for development..." -ForegroundColor Yellow
        $BuildTags += @("mock_db", "mock_email", "mock_storage")
    }
    
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host "Build tags: $($BuildTags -join ',')" -ForegroundColor Blue
    Write-Host "Total components: $($BuildTags.Count)" -ForegroundColor Blue
    Write-Host ""
    
    # Build with minimal tag set
    $SecondaryTagsString = ($BuildTags | Where-Object { $_ -ne "vanilla" }) -join ','
    
    Write-Host "Executing minimal build..." -ForegroundColor Magenta
    Write-Host "Command: .\scripts\build-with-tags.ps1 -Framework vanilla -SecondaryTags '$SecondaryTagsString' -Output $Output" -ForegroundColor Gray
    Write-Host ""
    
    # Build arguments with release optimizations
    $BuildArgs = @("-Framework", "vanilla", "-SecondaryTags", $SecondaryTagsString, "-Output", $Output)
    
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
    
    # Execute minimal build
    & ".\scripts\build-with-tags.ps1" @BuildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ Minimal API build finished!" -ForegroundColor Green
        
        # Show binary info with size comparison
        $OutputPath = "build/$Output"
        if (Test-Path $OutputPath) {
            Write-Host "✅ Binary created: $OutputPath" -ForegroundColor Green
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "✅ Binary size: $([math]::Round($BinarySize, 2)) MB (minimal feature set)" -ForegroundColor Blue
            
            # Size comparison with enterprise build
            $ComparisonSize = 70  # Typical enterprise build size
            $SizeReduction = [math]::Round(($ComparisonSize - $BinarySize) / $ComparisonSize * 100, 1)
            Write-Host "📊 Size reduction: $SizeReduction% smaller than enterprise builds" -ForegroundColor Green
        }
        
        Write-Host ""
        Write-Host "🚀 Minimal Deployment Examples:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Docker Container:" -ForegroundColor White
        Write-Host "    FROM alpine:latest" -ForegroundColor Gray
        Write-Host "    COPY build/$Output /app/$Output" -ForegroundColor Gray
        Write-Host "    RUN chmod +x /app/$Output" -ForegroundColor Gray
        Write-Host "    CMD [\"/app/$Output\"]" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Self-Hosted Server:" -ForegroundColor White
        Write-Host "    DATABASE_URL=postgres://user:pass@localhost:5432/db \\" -ForegroundColor Gray
        Write-Host "    JWT_SECRET=your-secret-key \\" -ForegroundColor Gray
        Write-Host "    SMTP_HOST=mail.example.com \\" -ForegroundColor Gray
        Write-Host "    SMTP_PORT=587 \\" -ForegroundColor Gray
        Write-Host "    STORAGE_PATH=/app/uploads \\" -ForegroundColor Gray
        Write-Host "    ./$Output" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Development with Mock Data:" -ForegroundColor White
        Write-Host "    MOCK_MODE=true \\" -ForegroundColor Gray
        Write-Host "    MOCK_BUSINESS_TYPE=education \\" -ForegroundColor Gray
        Write-Host "    LOG_LEVEL=debug \\" -ForegroundColor Gray
        Write-Host "    ./$Output" -ForegroundColor Gray
        Write-Host ""
        Write-Host "📋 Required Environment Variables:" -ForegroundColor Cyan
        Write-Host "  # Core Configuration" -ForegroundColor White
        Write-Host "  SERVER_TYPE=vanilla  # Uses Go standard library HTTP server" -ForegroundColor Gray
        Write-Host "  SERVER_PORT=8080     # HTTP port" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Database Connection" -ForegroundColor White
        Write-Host "  DATABASE_URL=postgres://user:password@host:5432/database" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Authentication" -ForegroundColor White
        Write-Host "  JWT_SECRET=your-secure-secret-key" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Email Configuration" -ForegroundColor White
        Write-Host "  SMTP_HOST=mail.example.com" -ForegroundColor Gray
        Write-Host "  SMTP_PORT=587" -ForegroundColor Gray
        Write-Host "  SMTP_USERNAME=your-email@example.com" -ForegroundColor Gray
        Write-Host "  SMTP_PASSWORD=your-email-password" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  # Storage" -ForegroundColor White
        Write-Host "  STORAGE_PATH=/app/uploads  # Local storage directory" -ForegroundColor Gray
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
Write-Host "💡 Minimal API Benefits:" -ForegroundColor Cyan
Write-Host "   • Ultra-small binary size - optimal for containers and edge deployment" -ForegroundColor Gray
Write-Host "   • Fast startup time - minimal dependency initialization" -ForegroundColor Gray
Write-Host "   • Low memory usage - single HTTP server, essential providers only" -ForegroundColor Gray
Write-Host "   • Self-contained - no external service dependencies required" -ForegroundColor Gray  
Write-Host "   • Cost-effective - reduced cloud resource usage" -ForegroundColor Gray
Write-Host "   • Production-ready - PostgreSQL + JWT is enterprise-proven stack" -ForegroundColor Gray
Write-Host "   • Horizontally scalable - stateless JWT authentication" -ForegroundColor Gray
Write-Host ""
Write-Host "🎯 Perfect for:" -ForegroundColor Cyan
Write-Host "   • Microservice architectures" -ForegroundColor Gray
Write-Host "   • Docker containers and Kubernetes pods" -ForegroundColor Gray
Write-Host "   • Edge computing and IoT deployments" -ForegroundColor Gray
Write-Host "   • Cost-conscious cloud deployments" -ForegroundColor Gray
Write-Host "   • Development and testing environments" -ForegroundColor Gray