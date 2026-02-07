#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Display all Espyna builds with their sizes and capabilities
#>

$EspynaDir = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
Push-Location $EspynaDir

try {
    Write-Host ""
    Write-Host "🏭 Espyna Build Summary" -ForegroundColor Cyan
    Write-Host "=" * 80 -ForegroundColor Gray
    Write-Host ""
    
    if (Test-Path "build") {
        $Builds = Get-ChildItem "build" -File | Sort-Object Name
        
        Write-Host "📦 Legacy Short Names:" -ForegroundColor White
        $Builds | Where-Object { $_.Name -notlike "*tags*" } | ForEach-Object {
            $SizeMB = [math]::Round($_.Length / 1MB, 1)
            Write-Host "   $($_.Name.PadRight(25)) $($SizeMB.ToString().PadLeft(6)) MB" -ForegroundColor Gray
        }
        
        Write-Host ""
        Write-Host "🏷️  Descriptive Tag-based Names:" -ForegroundColor White
        $Builds | Where-Object { $_.Name -like "*tags*" } | ForEach-Object {
            $SizeMB = [math]::Round($_.Length / 1MB, 1)
            Write-Host "   $($_.Name.PadRight(70)) $($SizeMB.ToString().PadLeft(6)) MB" -ForegroundColor Green
        }
        
        Write-Host ""
        Write-Host "📊 Build Statistics:" -ForegroundColor Cyan
        $TotalSize = ($Builds | Measure-Object -Property Length -Sum).Sum / 1MB
        Write-Host "   Total builds: $($Builds.Count)" -ForegroundColor White
        Write-Host "   Total size: $([math]::Round($TotalSize, 1)) MB" -ForegroundColor White
        Write-Host "   Average size: $([math]::Round($TotalSize / $Builds.Count, 1)) MB" -ForegroundColor White
        
        Write-Host ""
        Write-Host "🔍 Tag Analysis:" -ForegroundColor Cyan
        $TagBuilds = $Builds | Where-Object { $_.Name -like "*tags*" }
        Write-Host "   Builds with descriptive tags: $($TagBuilds.Count)" -ForegroundColor Green
        Write-Host "   Framework coverage:" -ForegroundColor White
        $VanillaCount = ($TagBuilds | Where-Object { $_.Name -like "*vanilla*" }).Count
        $GinCount = ($TagBuilds | Where-Object { $_.Name -like "*gin*" }).Count  
        $FiberCount = ($TagBuilds | Where-Object { $_.Name -like "*fiber*" }).Count
        Write-Host "     • Vanilla: $VanillaCount builds" -ForegroundColor Gray
        Write-Host "     • Gin: $GinCount builds" -ForegroundColor Gray
        Write-Host "     • Fiber: $FiberCount builds" -ForegroundColor Gray
        
        Write-Host ""
        Write-Host "💡 Usage Tips:" -ForegroundColor Cyan
        Write-Host "   • Tag-based names clearly show included providers" -ForegroundColor Gray
        Write-Host "   • Choose builds based on your deployment requirements" -ForegroundColor Gray
        Write-Host "   • All builds include mock providers for testing" -ForegroundColor Gray
        
    } else {
        Write-Host "❌ No build directory found. Run some builds first!" -ForegroundColor Red
    }
    
} catch {
    Write-Host "❌ Error: $_" -ForegroundColor Red
} finally {
    Pop-Location
}

Write-Host ""