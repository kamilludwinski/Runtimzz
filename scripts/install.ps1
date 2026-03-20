param(
    [string]$InstallDir = "$env:USERPROFILE\bin"
)

$ErrorActionPreference = 'Stop'

$repo = 'kamilludwinski/runtimzzz'
$apiUrl = "https://api.github.com/repos/$repo/releases/latest"
$headers = @{ 'User-Agent' = 'runtimz-installer' }

Write-Host "Fetching latest Runtimz release information..."
$release = Invoke-RestMethod -Uri $apiUrl -Headers $headers

if (-not $release -or -not $release.assets) {
    throw "Failed to fetch release information or no assets found."
}

# Prefer a bare .exe asset (new format), fall back to .zip (older releases)
$asset = $release.assets | Where-Object { $_.name -match 'rtz_.*_windows_amd64\.exe$' } | Select-Object -First 1
$useZip = $false

if (-not $asset) {
    $asset = $release.assets | Where-Object { $_.name -match 'rtz_.*_windows_amd64\.zip$' } | Select-Object -First 1
    if (-not $asset) {
        throw "No suitable Windows asset (.exe or .zip) found in the latest release."
    }
    $useZip = $true
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$destPath = Join-Path $InstallDir 'rtz.exe'

if (Test-Path $destPath) {
    Write-Host "Removing existing rtz.exe from $InstallDir ..."
    Remove-Item $destPath -Force -ErrorAction SilentlyContinue
}

$downloadUrl = $asset.browser_download_url
$tempPath = Join-Path $env:TEMP $asset.name

Write-Host "Downloading $($asset.name) ..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $tempPath

if ($useZip) {
    Write-Host "Extracting archive to $InstallDir ..."
    Expand-Archive -Path $tempPath -DestinationPath $InstallDir -Force
} else {
    Write-Host "Placing rtz.exe in $InstallDir ..."
    Move-Item -Force -Path $tempPath -Destination $destPath
}

if (Test-Path $tempPath) {
    Remove-Item $tempPath -Force -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "Runtimz has been installed to: $InstallDir"
Write-Host "Make sure this directory is on your PATH, then run: rtz"

