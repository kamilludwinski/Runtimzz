# E2E tests for rtz on Windows.
# Requires: RTZ_APP_DIR set to a temp dir, rtz.exe in current directory.
# Run from repo root after building: go build -o rtz.exe .

$ErrorActionPreference = "Stop"
# Prefer rtz.exe in current directory (e.g. when run from repo root in CI)
$rtz = Join-Path (Get-Location) "rtz.exe"
if (-not (Test-Path $rtz)) {
    $rtz = Join-Path $PSScriptRoot "..\..\rtz.exe"
}
if (-not (Test-Path $rtz)) {
    $rtz = "rtz.exe"
}
if (-not (Test-Path $rtz)) {
    Write-Error "rtz.exe not found. Build with: go build -o rtz.exe ."
}

$appDir = $env:RTZ_APP_DIR
if (-not $appDir) {
    $appDir = Join-Path $env:TEMP "rtz-e2e"
    New-Item -ItemType Directory -Force -Path $appDir | Out-Null
}
$env:RTZ_APP_DIR = $appDir

function Assert-ExitZero {
    param([string]$Caption, [scriptblock]$Block)
    & $Block
    if ($LASTEXITCODE -ne 0) {
        Write-Error "$Caption failed with exit code $LASTEXITCODE"
    }
}

Write-Host "E2E: rtz version"
$out = & $rtz version 2>&1
if ($LASTEXITCODE -ne 0) { throw "rtz version failed" }
if (-not ($out -match "[\d]+\.[\d]+\.[\d]+")) { throw "version output should contain x.y.z: $out" }

Write-Host "E2E: rtz (help)"
$out = & $rtz 2>&1
if ($LASTEXITCODE -ne 0) { throw "rtz (no args) failed" }
if (-not ($out -match "version")) { throw "help should mention version: $out" }
if (-not ($out -match "purge")) { throw "help should mention purge: $out" }
if (-not ($out -match "update")) { throw "help should mention update: $out" }

Write-Host "E2E: rtz purge (no app dir)"
$out = & $rtz purge 2>&1
if ($LASTEXITCODE -ne 0) { throw "rtz purge failed" }
if (-not ($out -match "Nothing to purge|purged|logs")) { throw "purge output unexpected: $out" }

Write-Host "E2E: rtz go purge (no versions)"
$out = & $rtz go purge 2>&1
if ($LASTEXITCODE -ne 0) { throw "rtz go purge failed" }
if (-not ($out -match "nothing to purge|No.*versions")) { throw "go purge output unexpected: $out" }

Write-Host "E2E: rtz go ls (smoke)"
$out = & $rtz go ls 2>&1
if ($LASTEXITCODE -ne 0) { throw "rtz go ls failed: $out" }
if (-not ($out -match "Go|available|installed")) { throw "go ls should mention Go/versions: $out" }

Write-Host "E2E: all passed"