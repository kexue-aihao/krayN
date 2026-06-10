param(
    [string]$ProjectDir = "app"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path "$PSScriptRoot\.."
$appPath = Join-Path $RepoRoot $ProjectDir
Push-Location $appPath
try {
    flutter create . --project-name krayn --org io.krayn --platforms windows,macos,linux,android,ios
    flutter pub get
    dart run flutter_launcher_icons
}
finally {
    Pop-Location
}
