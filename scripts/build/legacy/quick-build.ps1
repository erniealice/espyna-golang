#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Quick build script for Espyna server with working build tag combinations

.DESCRIPTION  
    This script provides pre-tested build tag combinations that are known to work.
    It uses the comprehensive tag set discovered through testing to ensure successful builds.

.PARAMETER Framework
    The HTTP framework to build. Valid values: vanilla, gin, fiber
    Default: vanilla

.PARAMETER Output
    Custom output binary name (optional)

.EXAMPLE
    .\quick-build.ps1 -Framework fiber
    Builds Fiber server with all providers

.EXAMPLE  
    .\quick-build.ps1 -Framework gin -Output my-gin-server
    Builds Gin server with custom name
#>

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("vanilla", "gin", "fiber")]
    [string]$Framework = "vanilla",
    
    [Parameter(Mandatory=$false)]
    [string]$Output
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host ""
    Write-Host "🚀 Espyna Quick Build" -ForegroundColor Cyan
    Write-Host "Framework: $Framework" -ForegroundColor Green
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    Write-Host ""
    
    # Comprehensive working tag set (tested and verified)
    $WorkingTags = @(
        $Framework,
        "providers_bootstrap",
        "mock_db", "mock_email", "mock_storage",
        "local_storage", "google", "aws", "s3", 
        "microsoft", "microsoftgraph", "gmail", "gcp_storage",
        "noop", "postgres", "firestore", "firebase", 
        "postgres_migrations"
    )
    
    # Set default output name with tags if not provided
    if (-not $Output) {
        # Create a descriptive filename based on key tags
        $KeyTags = @()
        $KeyTags += $Framework
        
        # Add major provider types to filename
        if ($WorkingTags -contains "firestore") { $KeyTags += "firestore" }
        if ($WorkingTags -contains "postgres") { $KeyTags += "postgres" }
        if ($WorkingTags -contains "firebase") { $KeyTags += "firebase" }
        if ($WorkingTags -contains "microsoft") { $KeyTags += "microsoft" }
        if ($WorkingTags -contains "google") { $KeyTags += "google" }
        if ($WorkingTags -contains "aws") { $KeyTags += "aws" }
        if ($WorkingTags -contains "mock_db") { $KeyTags += "mock" }
        
        $Output = "espyna-$($KeyTags -join '-')-tags"
        Write-Host "Generated filename: $Output" -ForegroundColor Blue
    }
    
    Write-Host "Build tags: $($WorkingTags -join ',')" -ForegroundColor Blue
    Write-Host ""
    
    # Use the original working build script with our tested tags
    $BuildScriptPath = ".\scripts\build-with-tags.ps1"
    $SecondaryTagsString = ($WorkingTags | Where-Object { $_ -ne $Framework }) -join ','
    
    Write-Host "Executing: $BuildScriptPath -Framework $Framework -SecondaryTags '$SecondaryTagsString' -Output $Output" -ForegroundColor Magenta
    Write-Host ""
    
    # Execute the build
    & $BuildScriptPath -Framework $Framework -SecondaryTags $SecondaryTagsString -Output $Output
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ Quick build completed!" -ForegroundColor Green
        Write-Host ""
        
        # Show binary location
        $BinaryPath = "build/$Output"
        if (Test-Path $BinaryPath) {
            Write-Host "📁 Binary location: $BinaryPath" -ForegroundColor Blue
            $BinarySize = (Get-Item $BinaryPath).Length / 1MB
            Write-Host "📏 Binary size: $([math]::Round($BinarySize, 2)) MB" -ForegroundColor Blue
        }
        
        Write-Host ""
        Write-Host "🧪 Test your build:" -ForegroundColor Cyan
        switch ($Framework) {
            "vanilla" {
                Write-Host "   ./$BinaryPath" -ForegroundColor White
                Write-Host "   MOCK_MODE=true ./$BinaryPath" -ForegroundColor Gray
            }
            "gin" {
                Write-Host "   SERVER_TYPE=gin ./$BinaryPath" -ForegroundColor White
                Write-Host "   SERVER_TYPE=gin MOCK_MODE=true ./$BinaryPath" -ForegroundColor Gray
            }
            "fiber" {
                Write-Host "   SERVER_TYPE=fiber ./$BinaryPath" -ForegroundColor White
                Write-Host "   SERVER_TYPE=fiber MOCK_MODE=true ./$BinaryPath" -ForegroundColor Gray
            }
        }
        
    } else {
        Write-Host ""
        Write-Host "❌ Build failed!" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
} catch {
    Write-Host "❌ Quick build error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "💡 Quick Build Benefits:" -ForegroundColor Cyan
Write-Host "   • Uses pre-tested working build tag combinations" -ForegroundColor Gray
Write-Host "   • Includes all essential providers for maximum functionality" -ForegroundColor Gray  
Write-Host "   • Reliable builds without tag dependency issues" -ForegroundColor Gray
Write-Host "   • Mock providers included for development and testing" -ForegroundColor Gray