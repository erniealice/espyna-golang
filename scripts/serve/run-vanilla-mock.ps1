# Run Espyna server with vanilla HTTP framework (with mock data) using air for hot reloading
# Usage: .\scripts\run-vanilla-mock.ps1

# Set environment variables
$env:SERVER_TYPE = "vanilla"
$env:SERVER_PORT = "8080"
$env:BUSINESS_TYPE = "education"

Write-Host "Starting Espyna server with vanilla HTTP framework and mock data (with air hot reloading)..." -ForegroundColor Green
Write-Host "Port: $env:SERVER_PORT" -ForegroundColor Cyan
Write-Host "Framework: $env:SERVER_TYPE" -ForegroundColor Cyan
Write-Host "Business Type: $env:BUSINESS_TYPE" -ForegroundColor Cyan
Write-Host "Hot Reloading: enabled" -ForegroundColor Cyan
Write-Host ""

air --build.cmd "go build -tags 'vanilla,providers_bootstrap,mock_db,mock_email,mock_storage,mock_auth,local_storage,noop,google,firebase,firestore,microsoft,postgres,jwt_auth,postgres_migrations' -o tmp/main.exe ./cmd/server/main_vanilla.go" --build.exclude_dir "scripts,files,tmp,logs,node_modules,.git,assets"