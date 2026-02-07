[CmdletBinding()]
param(
    [ValidateSet('vanilla', 'gin', 'fiber')]
    [string]$Framework = 'vanilla',

    [ValidateSet('postgres', 'firestore', 'mock_db')]
    [string]$Database = 'mock_db',

    [ValidateSet('firebase', 'jwt_auth', 'none')]
    [string]$Auth = 'none',

    [ValidateSet('gcp_storage', 's3', 'local_storage', 'mock_storage')]
    [string]$Storage = 'mock_storage',

    [ValidateSet('gmail', 'microsoftgraph', 'mock_email')]
    [string]$Email = 'mock_email',

    [ValidateSet('google', 'aws', 'microsoft', 'none')]
    [string]$CloudProvider = 'none',

    [string]$OutputFileBase = 'espyna-server',
    [string]$OutputDir = 'bin',
    [string]$GoOS = $env:GOOS,
    [string]$GoArch = $env:GOARCH,
    [switch]$Verbose,
    [switch]$Bootstrap = $true
)

# Tag dependency map
$tagDependencies = @{
    "gmail" = "google";
    "gcp_storage" = "google";
    "s3" = "aws";
    "microsoftgraph" = "microsoft";
    "firestore" = "firebase";
}

$tags = @()

# Add tags from parameters
$tags += $Framework
$tags += $Database
if ($Auth -ne 'none') {
    $tags += $Auth
}
$tags += $Storage
$tags += $Email
if ($CloudProvider -ne 'none') {
    $tags += $CloudProvider
}

# Add dependent tags
$initialTags = @($tags) # Copy of initial tags
foreach ($tag in $initialTags) {
    if ($tagDependencies.ContainsKey($tag)) {
        $tags += $tagDependencies[$tag]
    }
}

# Add bootstrap/dev tag
if ($Bootstrap) {
    $tags += "providers_bootstrap"
} else {
    $tags += "dev"
}

# Create unique, sorted list of tags
$uniqueTags = $tags | Select-Object -Unique | Sort-Object
$tagString = $uniqueTags -join ','

# Construct dynamic output file name
$fileNameTagString = $uniqueTags -join '-'
$outputFileName = "$OutputFileBase-$fileNameTagString"
if ($GoOS -eq 'windows') {
    $outputFileName += '.exe'
}

# Construct full output path
$packageRoot = (Get-Item -Path $PSScriptRoot).Parent.FullName
$outputFileFullPath = Join-Path -Path $packageRoot -ChildPath "$OutputDir\$outputFileName"

# Ensure output directory exists
$outputDirFullPath = Split-Path -Path $outputFileFullPath -Parent
if (-not (Test-Path -Path $outputDirFullPath)) {
    New-Item -ItemType Directory -Path $outputDirFullPath | Out-Null
}

# Set environment variables for cross-compilation
$env:GOOS = $GoOS
$env:GOARCH = $GoArch

# Construct the build command
$buildCmd = "go build -o `"$outputFileFullPath`" -ldflags=`"-s -w`" -tags=`"$tagString`" ..\cmd\server"

if ($Verbose) {
    Write-Host "Executing command in $packageRoot:"
    Write-Host $buildCmd
}

# Execute the build
Push-Location $packageRoot
Invoke-Expression $buildCmd
$exitCode = $LASTEXITCODE
Pop-Location

if ($exitCode -eq 0) {
    Write-Host "Build successful! Binary created at $outputFileFullPath"
} else {
    Write-Error "Build failed!"
}