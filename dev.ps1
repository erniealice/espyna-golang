# Check if Air is installed, if not install it
if (!(Get-Command air -ErrorAction SilentlyContinue)) {
    Write-Host "Installing Air for hot reloading..."
    go install github.com/cosmtrek/air@latest
}

# Run Air for hot reloading
Write-Host "Starting development server with hot reload (dev-minimal profile)..."
Write-Host "Build tags: vanilla,google_uuidv7,mock_db,mock_auth,mock_email,local_storage,noop"
Write-Host "Server will be available at: http://localhost:8080"
Write-Host "Press Ctrl+C to stop"

# Use the same tags as dev-minimal build
# Build the entire cmd/server package, not just main_vanilla.go
air --build.cmd "go build -tags vanilla,google_uuidv7,mock_db,mock_auth,mock_email,local_storage,noop -o tmp/main.exe ./cmd/server" --build.exclude_dir "files,tmp,logs,node_modules,.git,assets" --build.bin "./tmp/main.exe"  --build.stop_on_error "true"