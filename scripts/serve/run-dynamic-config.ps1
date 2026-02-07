# Dynamic Espyna Server Runner with Environment-Based Build Tags
# Usage: .\scripts\serve\run-dynamic-config.ps1
# Configure via environment variables before running

param(
    [string]$Mode = "development",  # development, production, test
    [switch]$HotReload = $true,     # Enable/disable air hot reloading
    [switch]$Verbose = $false       # Enable verbose build output
)

# =============================================================================
# CONFIGURATION ENVIRONMENT VARIABLES
# =============================================================================
# Set default values if not provided via environment

# Server Configuration
if (-not $env:CONFIG_SERVER_FRAMEWORK) { $env:CONFIG_SERVER_FRAMEWORK = "vanilla" }
if (-not $env:CONFIG_SERVER_PORT) { $env:CONFIG_SERVER_PORT = "8080" }

# Provider Configuration
if (-not $env:CONFIG_DATABASE_PROVIDER) { $env:CONFIG_DATABASE_PROVIDER = "mock_db" }
if (-not $env:CONFIG_AUTH_PROVIDER) { $env:CONFIG_AUTH_PROVIDER = "mock_auth" }
if (-not $env:CONFIG_EMAIL_PROVIDER) { $env:CONFIG_EMAIL_PROVIDER = "mock_email" }
if (-not $env:CONFIG_STORAGE_PROVIDER) { $env:CONFIG_STORAGE_PROVIDER = "mock_storage" }

# Business Type (separate from build tags)
if (-not $env:BUSINESS_TYPE) { $env:BUSINESS_TYPE = "education" }

# =============================================================================
# DYNAMIC BUILD TAG GENERATION
# =============================================================================

function Get-BuildTags {
    $tags = @()

    # Always include providers_bootstrap for provider system
    $tags += "providers_bootstrap"

    # Add framework tag
    $framework = $env:CONFIG_SERVER_FRAMEWORK.ToLower()
    if ($framework -in @("vanilla", "gin", "fiber")) {
        $tags += $framework
    } else {
        Write-Warning "Unknown framework: $framework, defaulting to vanilla"
        $tags += "vanilla"
        $env:CONFIG_SERVER_FRAMEWORK = "vanilla"
    }

    # Add database provider tag
    $dbProvider = $env:CONFIG_DATABASE_PROVIDER.ToLower()
    switch ($dbProvider) {
        "postgres" {
            $tags += "postgres"
        }
        "firestore" {
            $tags += "firestore"
            $tags += "firebase"  # Auto-add firebase for firestore
        }
        "mock_db" {
            $tags += "mock_db"
        }
        default {
            Write-Warning "Unknown database provider: $dbProvider, defaulting to mock_db"
            $tags += "mock_db"
            $env:CONFIG_DATABASE_PROVIDER = "mock_db"
        }
    }

    # Add authentication provider tag
    $authProvider = $env:CONFIG_AUTH_PROVIDER.ToLower()
    switch ($authProvider) {
        "firebase_auth" {
            $tags += "firebase"  # Auto-add firebase for firebase_auth
        }
        "jwt_auth" {
            $tags += "jwt_auth"
        }
        "mock_auth" {
            $tags += "mock_auth"
        }
        "noop" {
            $tags += "noop"
        }
        default {
            Write-Warning "Unknown auth provider: $authProvider, defaulting to mock_auth"
            $tags += "mock_auth"
            $env:CONFIG_AUTH_PROVIDER = "mock_auth"
        }
    }

    # Add email provider tag
    $emailProvider = $env:CONFIG_EMAIL_PROVIDER.ToLower()
    switch ($emailProvider) {
        "gmail" {
            $tags += "google"
            $tags += "gmail"
        }
        "microsoft_email" {
            $tags += "microsoft"      # Auto-add microsoft for microsoft_email
            $tags += "microsoftgraph"
        }
        "mock_email" {
            $tags += "mock_email"
        }
        default {
            Write-Warning "Unknown email provider: $emailProvider, defaulting to mock_email"
            $tags += "mock_email"
            $env:CONFIG_EMAIL_PROVIDER = "mock_email"
        }
    }

    # Add storage provider tag
    $storageProvider = $env:CONFIG_STORAGE_PROVIDER.ToLower()
    switch ($storageProvider) {
        "s3" {
            $tags += "aws"
            $tags += "s3"
        }
        "gcp_storage" {
            $tags += "google"
            $tags += "gcp_storage"
        }
        "local_storage" {
            $tags += "local_storage"
        }
        "mock_storage" {
            $tags += "mock_storage"
        }
        default {
            Write-Warning "Unknown storage provider: $storageProvider, defaulting to mock_storage"
            $tags += "mock_storage"
            $env:CONFIG_STORAGE_PROVIDER = "mock_storage"
        }
    }

    # Remove duplicates and return as comma-separated string
    $uniqueTags = $tags | Sort-Object | Get-Unique
    return ($uniqueTags -join ",")
}

# =============================================================================
# DISPLAY CONFIGURATION
# =============================================================================

function Show-Configuration {
    param($buildTags)

    Write-Host "🚀 Starting Espyna Server with Dynamic Configuration" -ForegroundColor Green
    Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Gray
    Write-Host "📊 Server Configuration:" -ForegroundColor Cyan
    Write-Host "  Framework: $env:CONFIG_SERVER_FRAMEWORK" -ForegroundColor White
    Write-Host "  Port: $env:CONFIG_SERVER_PORT" -ForegroundColor White
    Write-Host ""
    Write-Host "🔧 Provider Configuration:" -ForegroundColor Cyan
    Write-Host "  Database: $env:CONFIG_DATABASE_PROVIDER" -ForegroundColor White
    Write-Host "  Auth: $env:CONFIG_AUTH_PROVIDER" -ForegroundColor White
    Write-Host "  Email: $env:CONFIG_EMAIL_PROVIDER" -ForegroundColor White
    Write-Host "  Storage: $env:CONFIG_STORAGE_PROVIDER" -ForegroundColor White
    Write-Host ""
    Write-Host "🏢 Business Configuration:" -ForegroundColor Cyan
    Write-Host "  Business Type: $env:BUSINESS_TYPE" -ForegroundColor White
    Write-Host ""
    Write-Host "🏗️ Build Tags:" -ForegroundColor Magenta
    Write-Host "  $buildTags" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "⚙️ Mode: $Mode" -ForegroundColor Cyan
    Write-Host "🔥 Hot Reload: $HotReload" -ForegroundColor Cyan
    Write-Host "═══════════════════════════════════════════════════════" -ForegroundColor Gray
    Write-Host ""
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

# Generate build tags based on environment configuration
$buildTags = Get-BuildTags

# Display current configuration
Show-Configuration -buildTags $buildTags

# Set additional runtime environment variables
$env:SERVER_TYPE = $env:CONFIG_SERVER_FRAMEWORK
$env:SERVER_PORT = $env:CONFIG_SERVER_PORT

# Determine main file based on framework
$mainFile = switch ($env:CONFIG_SERVER_FRAMEWORK.ToLower()) {
    "vanilla" { "./cmd/server/main_vanilla.go" }
    "gin" { "./cmd/server/main_gin.go" }
    "fiber" { "./cmd/server/main_fiber.go" }
    default { "./cmd/server/main_vanilla.go" }
}

# Build command construction
$buildCmd = "go build -tags '$buildTags' -o tmp/main.exe $mainFile"

if ($Verbose) {
    $buildCmd += " -v"
}

# Execution based on mode
if ($HotReload -and $Mode -eq "development") {
    Write-Host "🔄 Starting with air hot reloading..." -ForegroundColor Green
    $airExcludes = "scripts,files,tmp,logs,node_modules,.git,assets,docs"
    air --build.cmd $buildCmd --build.exclude_dir $airExcludes
} else {
    Write-Host "🔨 Building and running directly..." -ForegroundColor Green

    # Build the server
    Invoke-Expression $buildCmd

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Build successful, starting server..." -ForegroundColor Green
        # Run the built executable
        ./tmp/main.exe
    } else {
        Write-Host "❌ Build failed with exit code: $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
}