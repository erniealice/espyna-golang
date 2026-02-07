#!/usr/bin/env pwsh

# This script runs the new data-driven unit and integration tests for the Espyna package.
# It executes `go test` from the `tests/unit` directory with the required build tags for
# the mock provider system and saves the output to a timestamped log file.
#
# Build Tags Used (matches dev-minimal profile):
# - mock_db: Enables mock database provider
# - mock_auth: Enables mock authentication provider  
# - mock_storage: Enables mock storage provider
# - mock_email: Enables mock email provider
# - noop: Enables no-op implementations for disabled services
# - vanilla: Enables vanilla HTTP framework support
# - fiber: Enables Fiber framework support (for HTTP helpers)
# - google_uuidv7: Enables Google UUID v7 ID generation service

# Set UTF-8 encoding for console output to handle Unicode characters properly
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$env:CHCP = "65001"

# Get the script's directory to build absolute paths
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$espynaRoot = Resolve-Path (Join-Path (Join-Path $scriptDir "..") "..")

$testDir = Join-Path $espynaRoot "tests/unit"
$resultsDir = Join-Path $testDir "results"

# Create results directory if it doesn't exist
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir -Force | Out-Null
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$logFile = Join-Path $resultsDir "$timestamp-unit-test-results.log"

Write-Host "Changing directory to $testDir"
Set-Location $testDir

Write-Host "Running Go tests..."
Write-Host "Log file will be saved to: $logFile"

# Execute go test with the required build tags for mock provider system
# These tags match the dev-minimal profile build configuration
$buildTags = "google_uuidv7,mock_auth,mock_db,mock_email,mock_storage,noop,vanilla,fiber"
$testArgs = @("test", "-v", "-tags=$buildTags", "./tests/unit/api/...")

Write-Host "Build tags: $buildTags"

# Execute go test and capture output with proper UTF-8 handling
try {
    Write-Host "Executing: go $($testArgs -join ' ')"
    
    # Use Start-Process to properly handle UTF-8 output
    $processInfo = New-Object System.Diagnostics.ProcessStartInfo
    $processInfo.FileName = "go"
    $processInfo.Arguments = $testArgs -join " "
    $processInfo.RedirectStandardOutput = $true
    $processInfo.RedirectStandardError = $true
    $processInfo.UseShellExecute = $false
    $processInfo.StandardOutputEncoding = [System.Text.Encoding]::UTF8
    $processInfo.StandardErrorEncoding = [System.Text.Encoding]::UTF8
    
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $processInfo
    $process.Start() | Out-Null
    
    $output = $process.StandardOutput.ReadToEnd()
    $errorOutput = $process.StandardError.ReadToEnd()
    $process.WaitForExit()
    $exitCode = $process.ExitCode
    
    # Combine stdout and stderr
    $combinedOutput = $output + $errorOutput
    
    # Write output to log file with UTF-8 encoding
    $combinedOutput | Out-File -FilePath $logFile -Encoding UTF8
    
    # Display output to console with proper UTF-8 handling
    Write-Host $combinedOutput
    
} catch {
    Write-Error "Failed to execute go test: $_"
    exit 1
}

Write-Host "Go tests finished. Exit code: $exitCode"

# Optional: Output the log file content to the console
# Get-Content $logFile

if ($exitCode -ne 0) {
    Write-Host "Tests failed. Check the log file for details: $logFile" -ForegroundColor Red
    exit $exitCode
} else {
    Write-Host "All tests passed successfully." -ForegroundColor Green
}
