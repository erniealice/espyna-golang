#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Interactive menu for building Espyna server variations with different technology stacks

.DESCRIPTION
    This script provides a user-friendly menu to choose from pre-configured build variations:
    1. Fiber + Firebase (High-performance + Cloud-native)
    2. Gin + Microsoft (Enterprise + Office 365)
    3. Vanilla + PostgreSQL (Minimal + Self-hosted)
    4. Multi-Hybrid (All frameworks + All providers)
    5. Custom build with manual tag selection

.PARAMETER AutoBuild
    Skip menu and build specific variation (1-5)

.PARAMETER MockMode
    Include mock providers for testing. Default: true

.EXAMPLE
    .\build-menu.ps1
    Interactive menu for build selection

.EXAMPLE
    .\build-menu.ps1 -AutoBuild 1 -MockMode:$false
    Auto-build Fiber + Firebase without mock providers

.NOTES
    Each build variation is optimized for specific use cases:
    - Choose based on deployment target and technology preferences
    - Mock providers are included by default for development/testing
#>

param(
    [Parameter(Mandatory=$false)]
    [ValidateRange(1,10)]
    [int]$AutoBuild,
    
    [Parameter(Mandatory=$false)]
    [bool]$MockMode = $true
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host ""
    Write-Host "╔══════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║                    Espyna Server Build Menu                      ║" -ForegroundColor Cyan
    Write-Host "╚══════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    
    if (-not $AutoBuild) {
        Write-Host "Choose your technology stack:" -ForegroundColor White
        Write-Host ""
        Write-Host "=== CLOUD-NATIVE STACKS ===" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "1. 🚀 Fiber + Firebase Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Fiber (ultra-fast, Express.js-like)" -ForegroundColor Gray
        Write-Host "   ├─ Database: Firestore (NoSQL, real-time)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: Firebase Auth (Google identity)" -ForegroundColor Gray
        Write-Host "   └─ Services: Google Cloud (Gmail, Storage)" -ForegroundColor Gray
        Write-Host "   💡 Best for: Modern web apps, real-time features, Google ecosystem" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "2. 🏢 Gin + Microsoft Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Gin (flexible, middleware-rich)" -ForegroundColor Gray
        Write-Host "   ├─ Database: PostgreSQL (enterprise grade)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: Azure Active Directory" -ForegroundColor Gray
        Write-Host "   └─ Services: Microsoft Graph (Office 365, Teams)" -ForegroundColor Gray
        Write-Host "   💡 Best for: Enterprise apps, Office integration, Microsoft ecosystem" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "3. ☁️  Fiber + AWS Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Fiber (ultra-high performance)" -ForegroundColor Gray
        Write-Host "   ├─ Database: PostgreSQL on RDS (managed)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: JWT (stateless for auto-scaling)" -ForegroundColor Gray
        Write-Host "   └─ Services: AWS S3, Amazon SES" -ForegroundColor Gray
        Write-Host "   💡 Best for: AWS cloud deployments, ECS/EKS, Lambda" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "=== DEPLOYMENT-OPTIMIZED STACKS ===" -ForegroundColor Cyan
        Write-Host ""
        
        Write-Host "4. ⚡ Minimal Edge Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Vanilla (Go standard library)" -ForegroundColor Gray
        Write-Host "   ├─ Database: Mock in-memory (ultra-fast)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: JWT (no external dependencies)" -ForegroundColor Gray
        Write-Host "   └─ Services: Local storage, Mock email" -ForegroundColor Gray
        Write-Host "   💡 Best for: IoT, edge computing, Raspberry Pi, minimal containers" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "5. 🐳 Container + K8s Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Fiber (container-optimized performance)" -ForegroundColor Gray
        Write-Host "   ├─ Database: Multi-cloud (PostgreSQL, Firestore)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: JWT (horizontal scaling ready)" -ForegroundColor Gray
        Write-Host "   └─ Services: Multi-cloud storage, Health checks" -ForegroundColor Gray
        Write-Host "   💡 Best for: Docker containers, Kubernetes, service mesh" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "=== DEVELOPMENT & ENTERPRISE ===" -ForegroundColor Cyan
        Write-Host ""
        
        Write-Host "6. 🛠️  Development Debug Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Gin (hot-reload friendly)" -ForegroundColor Gray
        Write-Host "   ├─ Database: Mock providers (offline development)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: All providers (integration testing)" -ForegroundColor Gray
        Write-Host "   └─ Services: Mock + real providers, race detection" -ForegroundColor Gray
        Write-Host "   💡 Best for: Local development, API testing, debugging" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "7. 🏆 Enterprise Complete Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Gin (enterprise middleware)" -ForegroundColor Gray
        Write-Host "   ├─ Database: Multi-provider (PostgreSQL + Firestore backup)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: All providers (Azure AD, Firebase, JWT)" -ForegroundColor Gray
        Write-Host "   └─ Services: Full multi-cloud support" -ForegroundColor Gray
        Write-Host "   💡 Best for: Large enterprises, maximum flexibility, hybrid cloud" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "=== LEGACY & CUSTOM ===" -ForegroundColor Cyan
        Write-Host ""
        
        Write-Host "8. ⚡ Vanilla + PostgreSQL Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: Vanilla (standard library, minimal)" -ForegroundColor Gray
        Write-Host "   ├─ Database: PostgreSQL (reliable, ACID)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: JWT (stateless, self-hosted)" -ForegroundColor Gray
        Write-Host "   └─ Services: SMTP, Local storage" -ForegroundColor Gray
        Write-Host "   💡 Best for: Self-hosted, minimal dependencies, traditional deployment" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "9. 🌐 Multi-Hybrid Stack" -ForegroundColor Green
        Write-Host "   ├─ HTTP: All frameworks (runtime switchable)" -ForegroundColor Gray
        Write-Host "   ├─ Database: Multiple providers (runtime switchable)" -ForegroundColor Gray
        Write-Host "   ├─ Auth: All providers (runtime switchable)" -ForegroundColor Gray
        Write-Host "   └─ Services: Everything included" -ForegroundColor Gray
        Write-Host "   💡 Best for: Maximum flexibility, A/B testing, migration scenarios" -ForegroundColor Yellow
        Write-Host ""
        
        Write-Host "10. 🔧 Custom Build" -ForegroundColor Green
        Write-Host "    └─ Manual tag selection with build-with-tags.ps1" -ForegroundColor Gray
        Write-Host "    💡 Best for: Specific combinations not covered above" -ForegroundColor Yellow
        Write-Host ""
        
        if ($MockMode) {
            Write-Host "🧪 Mock providers will be included for testing/development" -ForegroundColor Cyan
        } else {
            Write-Host "🔧 Production build - mock providers excluded" -ForegroundColor Magenta
        }
        Write-Host ""
        
        $Choice = Read-Host "Enter your choice (1-10)"
    } else {
        $Choice = $AutoBuild
        Write-Host "Auto-building option $Choice" -ForegroundColor Cyan
    }
    
    $ScriptPath = ""
    $Description = ""
    
    switch ($Choice) {
        1 {
            $ScriptPath = ".\scripts\build\build-fiber-firebase.ps1"
            $Description = "Fiber + Firebase Stack"
        }
        2 {
            $ScriptPath = ".\scripts\build\build-gin-microsoft.ps1"
            $Description = "Gin + Microsoft Stack"
        }
        3 {
            $ScriptPath = ".\scripts\build\build-fiber-aws.ps1"
            $Description = "Fiber + AWS Stack"
        }
        4 {
            $ScriptPath = ".\scripts\build\build-minimal-edge.ps1"
            $Description = "Minimal Edge Stack"
        }
        5 {
            $ScriptPath = ".\scripts\build\build-container-k8s.ps1"
            $Description = "Container + K8s Stack"
        }
        6 {
            $ScriptPath = ".\scripts\build\build-development-debug.ps1"
            $Description = "Development Debug Stack"
        }
        7 {
            $ScriptPath = ".\scripts\build\build-enterprise-complete.ps1"
            $Description = "Enterprise Complete Stack"
        }
        8 {
            $ScriptPath = ".\scripts\build\build-vanilla-postgres.ps1"
            $Description = "Vanilla + PostgreSQL Stack"
        }
        9 {
            $ScriptPath = ".\scripts\build\build-multi-hybrid.ps1"
            $Description = "Multi-Hybrid Stack"
        }
        10 {
            Write-Host ""
            Write-Host "Launching custom build script..." -ForegroundColor Cyan
            Write-Host "Use: .\scripts\build-with-tags.ps1 -Framework <framework> -SecondaryTags <tags>" -ForegroundColor White
            Write-Host ""
            Write-Host "Available frameworks: vanilla, gin, fiber" -ForegroundColor Gray
            Write-Host "Available secondary tags: firestore, firebase, google, microsoft, azure, postgres, jwt, etc." -ForegroundColor Gray
            Write-Host ""
            Write-Host "Example: .\scripts\build-with-tags.ps1 -Framework fiber -SecondaryTags 'firestore,firebase'" -ForegroundColor Yellow
            exit 0
        }
        default {
            Write-Host "Invalid choice. Please select 1-10." -ForegroundColor Red
            exit 1
        }
    }
    
    Write-Host ""
    Write-Host "🔨 Building: $Description" -ForegroundColor Cyan
    Write-Host "📁 Script: $ScriptPath" -ForegroundColor Gray
    
    # Build the arguments for the script
    $BuildArgs = @()
    if ($MockMode) {
        $BuildArgs += "-MockMode", $MockMode
    } else {
        $BuildArgs += "-MockMode:$false"
    }
    
    Write-Host "⚙️  Executing build script..." -ForegroundColor Blue
    Write-Host ""
    
    # Execute the selected build script
    & $ScriptPath @BuildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ Build completed successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "1. Test your build with the provided usage examples" -ForegroundColor White
        Write-Host "2. Configure environment variables for your target deployment" -ForegroundColor White
        Write-Host "3. Review the build-specific documentation above" -ForegroundColor White
        Write-Host ""
        Write-Host "📂 Built binaries are located in: packages/espyna/build/" -ForegroundColor Blue
    } else {
        Write-Host ""
        Write-Host "❌ Build failed. Check the error messages above." -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
} catch {
    Write-Host "❌ Menu script error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "🎯 Build Menu Benefits:" -ForegroundColor Cyan
Write-Host "   • Simplified build process with pre-configured stacks" -ForegroundColor Gray
Write-Host "   • Technology-specific optimizations for each use case" -ForegroundColor Gray
Write-Host "   • Easy switching between development and production builds" -ForegroundColor Gray
Write-Host "   • Clear documentation for each stack configuration" -ForegroundColor Gray