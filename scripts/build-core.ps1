param(
    [string]$Version = "dev",
    [string]$Commit = "local",
    [string]$OutDir = "dist/core",
    [switch]$IncludeAndroid
)

$ErrorActionPreference = "Stop"
$RepoRoot = Resolve-Path "$PSScriptRoot\.."
$OutputRoot = Join-Path $RepoRoot $OutDir
$targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Ext = ".exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; Ext = ".exe" },
    @{ GOOS = "linux"; GOARCH = "amd64"; Ext = "" },
    @{ GOOS = "linux"; GOARCH = "arm64"; Ext = "" },
    @{ GOOS = "darwin"; GOARCH = "amd64"; Ext = "" },
    @{ GOOS = "darwin"; GOARCH = "arm64"; Ext = "" }
)
$androidTargets = @(
    @{ GOOS = "android"; GOARCH = "arm64"; Ext = "" },
    @{ GOOS = "android"; GOARCH = "arm"; Ext = "" },
    @{ GOOS = "android"; GOARCH = "amd64"; Ext = "" }
)
if ($IncludeAndroid) {
    $targets += $androidTargets
    if (-not $env:ANDROID_NDK_HOME -and -not $env:ANDROID_HOME) {
        Write-Warning "Android Go targets usually require Android NDK/cgo. Set ANDROID_NDK_HOME before using -IncludeAndroid."
    }
}

New-Item -ItemType Directory -Force $OutputRoot | Out-Null
Push-Location (Join-Path $RepoRoot "core")
try {
    foreach ($target in $targets) {
        $env:GOOS = $target.GOOS
        $env:GOARCH = $target.GOARCH
        if ($target.GOOS -eq "android" -and $target.GOARCH -eq "arm") {
            $env:GOARM = "7"
        } else {
            Remove-Item Env:\GOARM -ErrorAction SilentlyContinue
        }
        $name = "krayn-core-$($target.GOOS)-$($target.GOARCH)$($target.Ext)"
        $out = Join-Path $OutputRoot $name
        go build -trimpath -ldflags "-s -w -X main.version=$Version -X main.commit=$Commit" -o $out ./cmd/krayn-core
        Write-Host "built $out"
    }
}
finally {
    Pop-Location
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARM -ErrorAction SilentlyContinue
}
