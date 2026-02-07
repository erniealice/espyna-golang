# Run Espyna server with vanilla HTTP framework (no mock data)
# Usage: .\scripts\run-vanilla.ps1

Write-Host "Starting Espyna server with vanilla HTTP framework..." -ForegroundColor Green
Write-Host "Port: 8080" -ForegroundColor Cyan
Write-Host "Framework: vanilla" -ForegroundColor Cyan
Write-Host "Mock Mode: disabled" -ForegroundColor Cyan
Write-Host ""

$env:SERVER_TYPE = "vanilla"
$env:SERVER_PORT = "8080"
$env:MOCK_MODE = "false"

go run -tags "vanilla,providers_bootstrap,postgres,jwt_auth,smtp_email,local_storage,noop,google,firebase,firestore,microsoft" ./cmd/server