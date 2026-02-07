# Run Espyna server with Gin HTTP framework (no mock data)
# Usage: .\scripts\run-gin.ps1

Write-Host "Starting Espyna server with Gin HTTP framework..." -ForegroundColor Green
Write-Host "Port: 8081" -ForegroundColor Cyan
Write-Host "Framework: gin" -ForegroundColor Cyan
Write-Host "Mock Mode: disabled" -ForegroundColor Cyan
Write-Host ""

$env:SERVER_TYPE = "gin"
$env:SERVER_PORT = "8081"
$env:MOCK_MODE = "false"

go run -tags "gin,providers_bootstrap,postgres,jwt_auth,smtp_email,local_storage,noop,google,firebase,firestore,microsoft" ./cmd/server