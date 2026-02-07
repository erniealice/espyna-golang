#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Build Espyna server with specific framework and secondary adapter tags.

.DESCRIPTION
    This script builds the Espyna server using Go build tags to conditionally compile
    only the specified HTTP framework adapters (vanilla, gin, fiber) and secondary adapters.
    
    The build tags are already implemented in the source files:
    - packages/espyna/internal/infrastructure/adapters/primary/http/vanilla/server.go (//go:build vanilla)
    - packages/espyna/internal/infrastructure/adapters/primary/http/gin/server.go (//go:build gin)  
    - packages/espyna/internal/infrastructure/adapters/primary/http/fiber/server.go (//go:build fiber)
    - Secondary adapters now also have build tags (e.g., //go:build firestore, //go:build google && gcp_storage).

.PARAMETER Framework
    The HTTP framework to build with. Valid values: vanilla, gin, fiber, all
    Default: vanilla

.PARAMETER SecondaryTags
    Comma-separated list of secondary adapter build tags (e.g., "firestore,google,aws").
    These tags will be combined with the framework tags.

.PARAMETER Output
    Output binary name. Default: espyna-server

.PARAMETER VerboseBuild
    Enable verbose build output

.PARAMETER Race
    Enable race condition detection

.PARAMETER LdFlags
    Additional linker flags

.EXAMPLE
    .build-with-tags.ps1 -Framework fiber -SecondaryTags "firestore,google"
    Builds server with Fiber framework and Firestore/Google Cloud adapters.

.EXAMPLE
    .build-with-tags.ps1 -Framework gin -Output espyna-gin -VerboseBuild -SecondaryTags "postgres,mock_email"
    Builds Gin-only server with verbose output, PostgreSQL, and mock email adapters.

.EXAMPLE
    .build-with-tags.ps1 -Framework all -Race -SecondaryTags "firestore,aws,microsoft"
    Builds server with all frameworks, race detection, and Firestore, AWS, and Microsoft adapters.

.NOTES
    Build tags reduce binary size and eliminate unused dependencies.
    The main.go file will conditionally import only the tagged frameworks.
#>

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("vanilla", "gin", "fiber")]
    [string]$Framework = "vanilla",
    
    [Parameter(Mandatory=$false)]
    [string[]]$SecondaryTags = @(),
    
    [Parameter(Mandatory=$false)]
    [string]$Output = "espyna-server",
    
    [Parameter(Mandatory=$false)]
    [switch]$VerboseBuild,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [string]$LdFlags = ""
)

# Set build directory to packages/espyna
$ScriptDir = $PSScriptRoot
$EspynaDir = Split-Path -Parent $ScriptDir
Push-Location $EspynaDir

try {
    Write-Host "Building Espyna Server with framework: $Framework" -ForegroundColor Cyan
    Write-Host "Working directory: $(Get-Location)" -ForegroundColor Gray
    
    # Prepare build command components
    $BuildCmd = "go build"
    $BuildArgs = @()
    
    # Collect all tags
    $AllTags = @()
    
    # Add framework tags
    if ($Framework -eq "all") {
        $AllTags += "vanilla", "gin", "fiber"
        Write-Host "Framework tags: vanilla,gin,fiber (all frameworks)" -ForegroundColor Green
    } else {
        $AllTags += $Framework
        Write-Host "Framework tags: $Framework" -ForegroundColor Green
    }
    
    # Add secondary tags
    if ($SecondaryTags.Count -gt 0) {
        $AllTags += $SecondaryTags
        Write-Host "Secondary tags: $($SecondaryTags -join ',')" -ForegroundColor Green
    }
    
    # Join all tags with commas
    $BuildArgs += "-tags", ($AllTags -join ',')
    
    # Add verbose flag if requested
    if ($VerboseBuild) {
        $BuildArgs += "-v"
        Write-Host "Verbose output enabled" -ForegroundColor Yellow
    }
    
    # Add race detection if requested
    if ($Race) {
        $BuildArgs += "-race"
        Write-Host "Race condition detection enabled" -ForegroundColor Yellow
    }
    
    # Add linker flags if provided
    if ($LdFlags) {
        $BuildArgs += "-ldflags"
        $BuildArgs += "`"$LdFlags`""
        Write-Host "Linker flags: $LdFlags" -ForegroundColor Yellow
    } else {
        # Suggest optimization flags if none provided
        Write-Host "Tip: Add -LdFlags '-s -w' for smaller binaries (strips debug symbols)" -ForegroundColor Blue
    }
    
    # Ensure build directory exists
    $BuildDir = "build"
    if (-not (Test-Path $BuildDir)) {
        New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null
        Write-Host "Created build directory: $BuildDir" -ForegroundColor Blue
    }
    
    # Set output path to build directory
    $OutputPath = "$BuildDir/$Output"
    $BuildArgs += "-o", $OutputPath
    
    # Add main package path
    $BuildArgs += "./cmd/server"
    
    Write-Host "Executing: $BuildCmd $($BuildArgs -join ' ')" -ForegroundColor Magenta
    Write-Host ""
    
    # Execute the build command
    $AllArgs = @("build") + $BuildArgs
    $Process = Start-Process -FilePath "go" -ArgumentList $AllArgs -Wait -PassThru -NoNewWindow
    
    if ($Process.ExitCode -eq 0) {
        Write-Host "Build completed successfully!" -ForegroundColor Green
        Write-Host "Binary created: $OutputPath" -ForegroundColor Green
        
        # Show enhanced binary info with size analysis
        if (Test-Path $OutputPath) {
            $BinarySize = (Get-Item $OutputPath).Length / 1MB
            Write-Host "Binary size: $([math]::Round($BinarySize, 2)) MB" -ForegroundColor Blue
            
            # Size category analysis
            if ($BinarySize -lt 15) {
                Write-Host "Size category: Optimized (< 15MB)" -ForegroundColor Green
            } elseif ($BinarySize -lt 35) {
                Write-Host "Size category: Moderate (15-35MB)" -ForegroundColor Yellow
            } elseif ($BinarySize -lt 60) {
                Write-Host "Size category: Large (35-60MB)" -ForegroundColor Magenta
            } else {
                Write-Host "Size category: Very Large (> 60MB)" -ForegroundColor Red
                Write-Host "💡 Consider using targeted builds (build-minimal-api.ps1 or build-cloud-native.ps1) for smaller size" -ForegroundColor Yellow
            }
            
            # Provider count estimation
            $TagCount = $AllTags.Count
            $EstimatedProviders = 0
            foreach ($tag in $AllTags) {
                if ($tag -match "^(postgres|firestore|firebase|google|aws|azure|microsoft|mock_)") {
                    $EstimatedProviders++
                }
            }
            
            if ($EstimatedProviders -gt 0) {
                Write-Host "Active providers: ~$EstimatedProviders (build tags: $($AllTags -join ','))" -ForegroundColor Gray
            }
            
            # Size optimization suggestions
            if ($BinarySize -gt 50 -and -not $LdFlags) {
                Write-Host ""
                Write-Host "Size optimization suggestions:" -ForegroundColor Cyan
                Write-Host "   • Add -LdFlags `"-s -w`" to strip debug symbols (~10-15% reduction)" -ForegroundColor Yellow
                Write-Host "   • Use build-minimal-api.ps1 for essential features only" -ForegroundColor Yellow
                Write-Host "   • Use build-cloud-native.ps1 -CloudProvider [gcp|aws|azure] for single-cloud builds" -ForegroundColor Yellow
            }
        }
        
        Write-Host ""
        Write-Host "Usage examples:" -ForegroundColor Cyan
        switch ($Framework) {
            "vanilla" {
                Write-Host "   ./$OutputPath" -ForegroundColor White
                Write-Host "   SERVER_PORT=8080 ./$OutputPath" -ForegroundColor White
            }
            "gin" {
                Write-Host "   ./$OutputPath" -ForegroundColor White  
                Write-Host "   SERVER_TYPE=gin SERVER_PORT=8081 ./$OutputPath" -ForegroundColor White
            }
            "fiber" {
                Write-Host "   ./$OutputPath" -ForegroundColor White
                Write-Host "   SERVER_TYPE=fiber SERVER_PORT=8082 ./$OutputPath" -ForegroundColor White
            }
            "all" {
                Write-Host "   SERVER_TYPE=vanilla ./$OutputPath" -ForegroundColor White
                Write-Host "   SERVER_TYPE=gin ./$OutputPath" -ForegroundColor White
                Write-Host "   SERVER_TYPE=fiber ./$OutputPath" -ForegroundColor White
                Write-Host "   SERVER_TYPE=multi ./$OutputPath" -ForegroundColor White
            }
        }
        
    } else {
        Write-Host "Build failed with exit code: $($Process.ExitCode)" -ForegroundColor Red
        exit $Process.ExitCode
    }
    
} catch {
    Write-Host "Build script error: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "Build Tag Benefits:" -ForegroundColor Cyan
Write-Host "   • Smaller binary size (excludes unused frameworks)" -ForegroundColor Gray
Write-Host "   • Faster compilation (fewer dependencies to build)" -ForegroundColor Gray  
Write-Host "   • Reduced memory footprint at runtime" -ForegroundColor Gray
Write-Host "   • Eliminates unused framework dependencies" -ForegroundColor Gray