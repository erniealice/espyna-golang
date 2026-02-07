# Run Espyna server with Fiber HTTP framework (with mock data)
# Usage: .\scripts\run-fiber-mock.ps1

# Set environment variables
$env:SERVER_TYPE = "fiber"
$env:SERVER_PORT = "8082"
$env:BUSINESS_TYPE = "education"

Write-Host "Starting Espyna server with Fiber HTTP framework and mock data (with air hot reloading)..." -ForegroundColor Green
Write-Host "Port: $env:SERVER_PORT" -ForegroundColor Cyan
Write-Host "Framework: $env:SERVER_TYPE" -ForegroundColor Cyan
Write-Host "Business Type: $env:BUSINESS_TYPE" -ForegroundColor Cyan
Write-Host "Hot Reloading: enabled" -ForegroundColor Cyan
Write-Host ""

air --build.cmd "go build -tags 'fiber,providers_bootstrap,mock_db,mock_email,mock_storage,mock_auth,local_storage,noop,google,firebase,firestore,microsoft,postgres,jwt_auth,postgres_migrations' -o tmp/main.exe ./cmd/server/main_fiber.go" --build.exclude_dir "scripts,files,tmp,logs,node_modules,.git,assets"