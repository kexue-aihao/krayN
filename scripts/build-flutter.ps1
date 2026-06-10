param(
    [string]$CoreDist = "dist/core",
    [string]$OutDir = "dist/app"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path "$PSScriptRoot\.."
$AppPath = Join-Path $RepoRoot "app"
$OutputRoot = Join-Path $RepoRoot $OutDir
Push-Location $AppPath
try {
    $missingRunner = -not (Test-Path "windows") -or -not (Test-Path "linux") -or -not (Test-Path "macos") -or -not (Test-Path "android") -or -not (Test-Path "ios")
    if ($missingRunner) {
        flutter create . --project-name krayn --org io.krayn --platforms windows,macos,linux,android,ios
    }
    flutter pub get
    flutter build windows --release
    flutter build linux --release
    flutter build macos --release
    flutter build apk --release --split-per-abi
}
finally {
    Pop-Location
}

New-Item -ItemType Directory -Force $OutputRoot | Out-Null
Write-Host "Flutter release outputs are under app/build. Pair desktop bundles with the matching krayn-core binary from $CoreDist."
