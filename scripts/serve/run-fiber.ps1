# Run Espyna server with Fiber HTTP framework (no mock data)
# Usage: .\scripts\run-fiber.ps1

Write-Host "Starting Espyna server with Fiber HTTP framework..." -ForegroundColor Green
Write-Host "Port: 8082" -ForegroundColor Cyan
Write-Host "Framework: fiber" -ForegroundColor Cyan
Write-Host "Mock Mode: disabled" -ForegroundColor Cyan
Write-Host ""

$env:SERVER_TYPE = "fiber"
$env:SERVER_PORT = "8082"
$env:MOCK_MODE = "false"

go run -tags "fiber,providers_bootstrap,postgres,jwt_auth,smtp_email,local_storage,noop,google,firebase,firestore,microsoft" ./cmd/server