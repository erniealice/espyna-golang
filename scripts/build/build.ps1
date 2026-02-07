[CmdletBinding()]
param(
    # Profile-based build system
    [ValidateSet("dev-minimal", "dev-postgres", "gcp-full", "gcp-hybrid", 
                 "aws-standard", "azure-standard", "hybrid-postgres", 
                 "hybrid-firestore", "testing-integration", "minimal-api")]
    [string]$Profile = "dev-minimal",
    
    # Framework selection (overrides profile default if specified)
    [ValidateSet("vanilla", "gin", "fiber", "fiber_v3")]
    [string]$Framework = "",
    
    # Individual parameter overrides (optional - will use profile defaults if not specified)
    [ValidateSet("postgres", "firestore", "mock_db")]
    [string]$Database = "",
    
    [ValidateSet("firebase", "jwt_auth", "mock_auth", "none")]
    [string]$Auth = "",
    
    [ValidateSet("gcp_storage", "s3", "local_storage", "mock_storage")]
    [string]$Storage = "",
    
    [ValidateSet("gmail", "microsoftgraph", "mock_email")]
    [string]$Email = "",
    
    [ValidateSet("google", "aws", "microsoft", "none")]
            [string]$CloudProvider = "",
    
    [ValidateSet("google_uuidv7", "local_uuid", "mock_id")]
    [string]$IdGenerator = "",
    
    # Build configuration
    [string]$OutputFileBase = "espyna-server",
    [string]$OutputDir = "build/",
    [string]$GoOS = $env:GOOS,
    [string]$GoArch = $env:GOARCH,
    [switch]$DryRun
)

# Profile Configuration Definitions
$ProfileConfigurations = @{
    "dev-minimal" = @{
        Framework = "vanilla"
        Database = "mock_db"
        Auth = "mock_auth"
        Storage = "local_storage"
        Email = "mock_email"
                CloudProvider = "none"
        idgenerator = "google_uuidv7"
        Description = "Local development with mock database and local file storage"
    }
    "dev-postgres" = @{
        Framework = "vanilla"
        Database = "postgres"
        Auth = "jwt_auth"
        Storage = "local_storage"
        Email = "mock_email"
        CloudProvider = "none"
        idgenerator = "google_uuidv7"
        Description = "Local development with real database"
    }
    "gcp-full" = @{
        Framework = "vanilla"
        Database = "firestore"
        Auth = "firebase"
        Storage = "gcp_storage"
        Email = "gmail"
                CloudProvider = "google"
        idgenerator = "google_uuidv7"
        Description = "Complete Google Cloud Platform deployment"
    }
    "gcp-hybrid" = @{
        Framework = "vanilla"
        Database = "firestore"
        Auth = "firebase"
        Storage = "gcp_storage"
        Email = "microsoftgraph"
        CloudProvider = "microsoft"
        IdGenerator = "google_uuidv7"
        Description = "GCP with Microsoft Graph email integration"
    }
    "aws-standard" = @{
        Framework = "vanilla"
        Database = "postgres"
        Auth = "jwt_auth"
        Storage = "s3"
        Email = "mock_email"
        CloudProvider = "aws"
        IdGenerator = "google_uuidv7"
        Description = "Standard AWS deployment with PostgreSQL and S3"
    }
    "azure-standard" = @{
        Framework = "vanilla"
        Database = "postgres"
        Auth = "jwt_auth"
        Storage = "local_storage"
        Email = "microsoftgraph"
        CloudProvider = "microsoft"
        IdGenerator = "google_uuidv7"
        Description = "Microsoft Azure deployment with Microsoft Graph"
    }
    "hybrid-postgres" = @{
        Framework = "vanilla"
        Database = "postgres"
        Auth = "jwt_auth"
        Storage = "local_storage"
        Email = "mock_email"
        CloudProvider = "none"
        IdGenerator = "google_uuidv7"
        Description = "Database-focused deployment without cloud dependencies"
    }
    "hybrid-firestore" = @{
        Framework = "vanilla"
        Database = "firestore"
        Auth = "none"
        Storage = "local_storage"
        Email = "mock_email"
        CloudProvider = "none"
        IdGenerator = "google_uuidv7"
        Description = "Firestore with local services"
    }
    "testing-integration" = @{
        Framework = "gin"
        Database = "postgres"
        Auth = "jwt_auth"
        Storage = "local_storage"
        Email = "mock_email"
        CloudProvider = "none"
        IdGenerator = "google_uuidv7"
        Description = "Integration testing with external services"
    }
    "minimal-api" = @{
        Framework = "gin"
        Database = "mock_db"
        Auth = "none"
        Storage = "mock_storage"
        Email = "mock_email"
        CloudProvider = "none"
        IdGenerator = "google_uuidv7"
        Description = "Lightweight API server"
    }
}

# Tag dependency map for auto-correction
$TagDependencies = @{
    "gmail" = "google"
    "gcp_storage" = "google"
    "s3" = "aws"
    "microsoftgraph" = "microsoft"
    "firebase" = "google"
}

function Write-ProfileInfo {
    param([string]$ProfileName, [hashtable]$Config)
    
    Write-Host ""
    Write-Host "Profile: $ProfileName" -ForegroundColor Cyan
    Write-Host "Description: $($Config.Description)" -ForegroundColor Gray
    Write-Host "Configuration:" -ForegroundColor Yellow
    Write-Host "  Framework: $($Config.Framework)" -ForegroundColor White
    Write-Host "  Database: $($Config.Database)" -ForegroundColor White
    Write-Host "  Auth: $($Config.Auth)" -ForegroundColor White
    Write-Host "  Storage: $($Config.Storage)" -ForegroundColor White
    Write-Host "  Email: $($Config.Email)" -ForegroundColor White
    Write-Host "  CloudProvider: $($Config.CloudProvider)" -ForegroundColor White
    Write-Host "  IdGenerator: $($Config.IdGenerator)" -ForegroundColor White
    Write-Host ""
}

function Get-EffectiveConfiguration {
    param([string]$ProfileName, [hashtable]$Overrides)
    
    # Start with profile defaults
    $profileConfig = $ProfileConfigurations[$ProfileName].Clone()
    
    # Apply parameter overrides
    if ($Overrides.Framework) { $profileConfig.Framework = $Overrides.Framework }
    if ($Overrides.Database) { $profileConfig.Database = $Overrides.Database }
    if ($Overrides.Auth) { $profileConfig.Auth = $Overrides.Auth }
    if ($Overrides.Storage) { $profileConfig.Storage = $Overrides.Storage }
    if ($Overrides.Email) { $profileConfig.Email = $Overrides.Email }
    if ($Overrides.CloudProvider) { $profileConfig.CloudProvider = $Overrides.CloudProvider }
    if ($Overrides.IdGenerator) { $profileConfig.IdGenerator = $Overrides.IdGenerator }
    
    return $profileConfig
}

function Resolve-Dependencies {
    param([hashtable]$Config)
    
    $resolvedConfig = $Config.Clone()
    $changed = $false
    
    # Auto-correct cloud provider based on service dependencies
    foreach ($service in @($Config.Storage, $Config.Email, $Config.Auth)) {
        if ($TagDependencies.ContainsKey($service)) {
            $requiredProvider = $TagDependencies[$service]
            if ($Config.CloudProvider -ne $requiredProvider) {
                Write-Warning "Auto-correcting CloudProvider from '$($Config.CloudProvider)' to '$requiredProvider' due to $service dependency"
                $resolvedConfig.CloudProvider = $requiredProvider
                $changed = $true
            }
        }
    }
    
    return @{
        Config = $resolvedConfig
        Changed = $changed
    }
}

function Build-TagString {
    param([hashtable]$Config)
    
    $tags = @()
    
    # Add core tags
    $tags += $Config.Framework
    $tags += $Config.Database
    
    if ($Config.Auth -ne 'none') {
        $tags += $Config.Auth
    }
    
    $tags += $Config.Storage
    $tags += $Config.Email
    
    if ($Config.CloudProvider -ne 'none') {
        $tags += $Config.CloudProvider
    }
    
    # Add ID generator tag
    if ($Config.IdGenerator -ne 'none' -and $Config.IdGenerator -ne '') {
        $tags += $Config.IdGenerator
    }
    
    # No longer need bootstrap tag - factory pattern handles provider selection
    # The presence of provider-specific tags (postgres, firestore, mock_db, etc.) 
    # determines which factory implementation is compiled in
    
    # Add noop tag for auth when using mock_auth
    if ($Config.Auth -eq "mock_auth") {
        $tags += "noop"
    }
    
    # Apply dependency resolution
    $initialTags = @($tags)
    foreach ($tag in $initialTags) {
        if ($TagDependencies.ContainsKey($tag)) {
            $dependentTag = $TagDependencies[$tag]
            if ($tags -notcontains $dependentTag) {
                $tags += $dependentTag
            }
        }
    }
    
    # Create unique, sorted list
    $uniqueTags = $tags | Select-Object -Unique | Sort-Object
    return $uniqueTags -join ','
}

function Build-OutputFileName {
    param([string]$Profile, [string]$Framework, [string]$BaseDir, [string]$GoOS)
    
    # Use profile-based naming: espyna-server-{profile}-{framework}
    $fileName = "$OutputFileBase-$Profile-$Framework"
    
    # Add .exe extension for Windows builds
    if ($GoOS -eq 'windows' -or ($GoOS -eq '' -and $env:OS -match 'Windows')) {
        $fileName += '.exe'
    }
    
    return $fileName
}

# Main execution starts here
Write-Host "Espyna Profile-Based Build System" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Green

# Validate profile exists
if (-not $ProfileConfigurations.ContainsKey($Profile)) {
    Write-Error "Unknown profile: $Profile"
    Write-Host "Available profiles:" -ForegroundColor Yellow
    foreach ($p in $ProfileConfigurations.Keys | Sort-Object) {
        Write-Host "  $p - $($ProfileConfigurations[$p].Description)" -ForegroundColor White
    }
    exit 1
}

# Build override hashtable
$overrides = @{
    Framework = $Framework
    Database = $Database
    Auth = $Auth
    Storage = $Storage
    Email = $Email
    CloudProvider = $CloudProvider
    IdGenerator = $IdGenerator
}

# Get effective configuration with overrides
$effectiveConfig = Get-EffectiveConfiguration -ProfileName $Profile -Overrides $overrides

# Display configuration info
Write-ProfileInfo -ProfileName $Profile -Config $effectiveConfig

# Resolve dependencies and show any auto-corrections
$resolutionResult = Resolve-Dependencies -Config $effectiveConfig
$finalConfig = $resolutionResult.Config

if ($resolutionResult.Changed) {
    Write-Host "Final configuration after dependency resolution:" -ForegroundColor Yellow
    foreach ($key in $finalConfig.Keys | Where-Object { $_ -ne 'Description' } | Sort-Object) {
        $value = $finalConfig[$key]
        Write-Host "  ${key}: $value" -ForegroundColor White
    }
    Write-Host ""
}

# Build tags string
$tagString = Build-TagString -Config $finalConfig

# Build output file name using profile-based naming
$outputFileName = Build-OutputFileName -Profile $Profile -Framework $finalConfig.Framework -BaseDir $OutputDir -GoOS $GoOS

# Construct full output path
$packageRoot = (Get-Item -Path $PSScriptRoot).Parent.Parent.FullName
$outputFileFullPath = Join-Path -Path $packageRoot -ChildPath "$OutputDir$outputFileName"

# Ensure output directory exists
$outputDirFullPath = Split-Path -Path $outputFileFullPath -Parent
if (-not (Test-Path -Path $outputDirFullPath)) {
    New-Item -ItemType Directory -Path $outputDirFullPath -Force | Out-Null
}

# Set environment variables for cross-compilation
if ($GoOS) { $env:GOOS = $GoOS }
if ($GoArch) { $env:GOARCH = $GoArch }

# Construct the build command
$buildCmd = "go build -o `"$outputFileFullPath`" -ldflags=`"-s -w`" -tags=`"$tagString`" ./cmd/server"

# Display build information
Write-Host "Build Configuration:" -ForegroundColor Cyan
Write-Host "  Profile: $Profile" -ForegroundColor White
Write-Host "  Framework: $($finalConfig.Framework)" -ForegroundColor White
Write-Host "  Tags: $tagString" -ForegroundColor White
Write-Host "  Output: $outputFileName" -ForegroundColor White
Write-Host "  Full Path: $outputFileFullPath" -ForegroundColor Gray
Write-Host ""

if ($VerbosePreference -ne 'SilentlyContinue' -or $DryRun) {
    Write-Host "Build command:" -ForegroundColor Yellow
    Write-Host "  Working Directory: $packageRoot" -ForegroundColor Gray
    Write-Host "  Command: $buildCmd" -ForegroundColor Gray
    Write-Host ""
}

if ($DryRun) {
    Write-Host "Dry run completed. No build executed." -ForegroundColor Magenta
    exit 0
}

# For demonstration purposes, create mock binary if demo mode is enabled
if ($env:ESPYNA_DEMO_MODE -eq 'true') {
    Write-Host ""
    Write-Host "Demo Mode: Creating mock binary to demonstrate profile naming..." -ForegroundColor Yellow
    
    # Create a mock binary file to demonstrate the profile naming system
    $demoContent = @"
Espyna Server - Profile Build Demo
==================================
Profile: $Profile
Framework: $($finalConfig.Framework)
Build Tags: $tagString
Generated: $(Get-Date)
Binary Name: $outputFileName
"@
    
    Set-Content -Path $outputFileFullPath -Value $demoContent -Encoding UTF8
    Write-Host "Mock binary created for demonstration." -ForegroundColor Green
    $exitCode = 0
} else {

# Execute the build
Write-Host "Building..." -ForegroundColor Green
Push-Location $packageRoot
try {
    Invoke-Expression $buildCmd
    $exitCode = $LASTEXITCODE
} catch {
    Write-Error "Build command failed: $_"
    exit 1
} finally {
    Pop-Location
}

}

# Report results
if ($exitCode -eq 0) {
    Write-Host ""
    Write-Host "Build successful!" -ForegroundColor Green
    Write-Host "  Binary: $outputFileFullPath" -ForegroundColor White
    
    # Show file info if it exists
    if (Test-Path $outputFileFullPath) {
        $fileInfo = Get-Item $outputFileFullPath
        $fileSizeMB = [math]::Round($fileInfo.Length / 1MB, 1)
        Write-Host "  Size: $fileSizeMB MB" -ForegroundColor Gray
        Write-Host "  Created: $($fileInfo.CreationTime)" -ForegroundColor Gray
    }
} else {
    Write-Host ""
    Write-Host "Build failed!" -ForegroundColor Red
    Write-Host "  Exit code: $exitCode" -ForegroundColor Red
    exit $exitCode
}