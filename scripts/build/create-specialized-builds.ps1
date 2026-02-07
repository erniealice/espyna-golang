#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Create multiple specialized Espyna builds with descriptive tag-based names

.DESCRIPTION
    This script creates several specialized builds using the working tag combinations
    but with descriptive filenames that clearly indicate what capabilities each build has.
#>

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Push-Location $EspynaDir

try {
    Write-Host "🏭 Creating Specialized Espyna Builds" -ForegroundColor Cyan
    Write-Host "Each build will have a descriptive name showing its capabilities" -ForegroundColor White
    Write-Host ""
    
    # Working tag set that we know compiles successfully
    $BaseWorkingTags = @(
        "providers_bootstrap",
        "mock_db", "mock_email", "mock_storage",
        "local_storage", "google", "aws", "s3", 
        "microsoft", "microsoftgraph", "gmail", "gcp_storage",
        "noop", "postgres", "firestore", "firebase", 
        "postgres_migrations"
    )
    
    # Define specialized builds with their intended focus
    $SpecializedBuilds = @(
        @{
            Name = "espyna-vanilla-selfhosted-postgres-jwt-local-tags"
            Framework = "vanilla"
            Description = "Self-hosted vanilla server with PostgreSQL and JWT"
            FocusTags = @("postgres", "jwt_auth", "local_storage")
        },
        @{
            Name = "espyna-gin-enterprise-microsoft-azure-graph-tags"
            Framework = "gin"  
            Description = "Enterprise Gin server with Microsoft ecosystem"
            FocusTags = @("microsoft", "microsoftgraph")
        },
        @{
            Name = "espyna-fiber-cloud-google-firebase-firestore-tags"
            Framework = "fiber"
            Description = "High-performance Fiber with Google Cloud services"  
            FocusTags = @("google", "firebase", "firestore", "gmail", "gcp_storage")
        },
        @{
            Name = "espyna-fiber-aws-postgres-s3-lambda-tags"
            Framework = "fiber"
            Description = "AWS-optimized Fiber server for Lambda/ECS"
            FocusTags = @("aws", "s3", "postgres")
        },
        @{
            Name = "espyna-gin-hybrid-multicloud-enterprise-tags" 
            Framework = "gin"
            Description = "Multi-cloud enterprise Gin server"
            FocusTags = @("google", "microsoft", "aws", "postgres", "firestore")
        }
    )
    
    foreach ($Build in $SpecializedBuilds) {
        Write-Host "🔨 Building: $($Build.Description)" -ForegroundColor Green
        Write-Host "   Framework: $($Build.Framework)" -ForegroundColor Gray
        Write-Host "   Focus: $($Build.FocusTags -join ', ')" -ForegroundColor Gray
        Write-Host "   Output: $($Build.Name)" -ForegroundColor Blue
        
        # Combine base working tags with any focus-specific tags
        $AllTags = $BaseWorkingTags + $Build.FocusTags | Select-Object -Unique
        $SecondaryTagsString = ($AllTags | Where-Object { $_ -ne $Build.Framework }) -join ','
        
        # Execute build
        $BuildArgs = @(
            "-Framework", $Build.Framework,
            "-SecondaryTags", $SecondaryTagsString, 
            "-Output", $Build.Name
        )
        
        Write-Host "   Executing build..." -ForegroundColor Gray
        & ".\scripts\build-with-tags.ps1" @BuildArgs
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "   ✅ Success!" -ForegroundColor Green
            
            # Show binary info
            $BinaryPath = "build/$($Build.Name)"
            if (Test-Path $BinaryPath) {
                $BinarySize = (Get-Item $BinaryPath).Length / 1MB
                Write-Host "   📏 Size: $([math]::Round($BinarySize, 2)) MB" -ForegroundColor Blue
            }
        } else {
            Write-Host "   ❌ Failed!" -ForegroundColor Red
        }
        
        Write-Host ""
    }
    
    Write-Host "🎉 Specialized builds complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📂 All binaries are in the build/ directory:" -ForegroundColor Cyan
    
    # List all binaries with sizes
    if (Test-Path "build") {
        Get-ChildItem "build" -File | ForEach-Object {
            $Size = ($_.Length / 1MB)
            Write-Host "   $($_.Name) - $([math]::Round($Size, 2)) MB" -ForegroundColor White
        }
    }
    
} catch {
    Write-Host "❌ Script error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "💡 Usage Tips:" -ForegroundColor Cyan
Write-Host "   • Filenames clearly indicate capabilities (framework + key providers)" -ForegroundColor Gray
Write-Host "   • Choose builds based on your deployment target and requirements" -ForegroundColor Gray
Write-Host "   • All builds include mock providers for development/testing" -ForegroundColor Gray