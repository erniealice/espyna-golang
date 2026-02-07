# Run Espyna server with vanilla HTTP framework (with mock data) using dynamic configuration
# Usage: .\scripts\serve\run-vanilla-mock2.ps1

# Set provider configuration for vanilla + mock setup
$env:CONFIG_SERVER_FRAMEWORK = "vanilla"
$env:CONFIG_SERVER_PORT = "8080"
$env:CONFIG_DATABASE_PROVIDER = "mock_db"
$env:CONFIG_AUTH_PROVIDER = "mock_auth"
$env:CONFIG_EMAIL_PROVIDER = "mock_email"
$env:CONFIG_STORAGE_PROVIDER = "mock_storage"

# Set business type (not part of build tags)
$env:BUSINESS_TYPE = "education"

# Legacy environment variables for compatibility
$env:SERVER_TYPE = "vanilla"
$env:SERVER_PORT = "8080"

Write-Host "🚀 Starting Espyna with Vanilla + Mock providers (dynamic configuration)" -ForegroundColor Green

# Run the dynamic configuration script
& ".\scripts\serve\run-dynamic-config.ps1" -Mode development -HotReload