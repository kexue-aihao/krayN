param(
    [string]$InstallDir = "",
    [switch]$AllUsers,
    [switch]$KeepProfiles,
    [switch]$SkipInstallDir,
    [switch]$Silent
)

$ErrorActionPreference = "Stop"
$isChinese = [System.Globalization.CultureInfo]::CurrentUICulture.Name -like "zh*"

function U([int[]]$CodePoints) {
    $chars = foreach ($point in $CodePoints) {
        [char]$point
    }
    -join $chars
}

$messages = if ($isChinese) {
    @{
        Title = (U @(0x006B, 0x0072, 0x0061, 0x0079, 0x004E, 0x0020, 0x5B8C, 0x6574, 0x5378, 0x8F7D))
        Stop = (U @(0x6B63, 0x5728, 0x505C, 0x6B62, 0x0020, 0x006B, 0x0072, 0x0061, 0x0079, 0x004E, 0x0020, 0x8FDB, 0x7A0B, 0x002E, 0x002E, 0x002E))
        Shortcuts = (U @(0x6B63, 0x5728, 0x5220, 0x9664, 0x5FEB, 0x6377, 0x65B9, 0x5F0F, 0x002E, 0x002E, 0x002E))
        Data = (U @(0x6B63, 0x5728, 0x5220, 0x9664, 0x7528, 0x6237, 0x914D, 0x7F6E, 0x548C, 0x7F13, 0x5B58, 0x002E, 0x002E, 0x002E))
        InstallFiles = (U @(0x6B63, 0x5728, 0x6E05, 0x7406, 0x5B89, 0x88C5, 0x6587, 0x4EF6, 0x002E, 0x002E, 0x002E))
        Install = (U @(0x6B63, 0x5728, 0x5220, 0x9664, 0x5B89, 0x88C5, 0x76EE, 0x5F55, 0x002E, 0x002E, 0x002E))
        Keep = (U @(0x5DF2, 0x6309, 0x8981, 0x6C42, 0x4FDD, 0x7559, 0x7528, 0x6237, 0x914D, 0x7F6E, 0x3002))
        Done = (U @(0x006B, 0x0072, 0x0061, 0x0079, 0x004E, 0x0020, 0x5DF2, 0x5B8C, 0x6574, 0x5378, 0x8F7D, 0x3002))
        Skip = (U @(0x8DF3, 0x8FC7, 0x8DEF, 0x5F84, 0xFF1A, 0x007B, 0x0030, 0x007D))
        Remove = (U @(0x5220, 0x9664, 0xFF1A, 0x007B, 0x0030, 0x007D))
        Warn = (U @(0x8B66, 0x544A, 0xFF1A, 0x007B, 0x0030, 0x007D))
    }
}
else {
    @{
        Title = "krayN complete uninstall"
        Stop = "Stopping krayN processes..."
        Shortcuts = "Removing shortcuts..."
        Data = "Removing user configuration and cache..."
        InstallFiles = "Removing installed files..."
        Install = "Removing install directory..."
        Keep = "User profiles were kept as requested."
        Done = "krayN has been completely uninstalled."
        Skip = "Skipped path: {0}"
        Remove = "Removed: {0}"
        Warn = "Warning: {0}"
    }
}

function Write-Localized([string]$Text) {
    if (-not $Silent) {
        Write-Host $Text
    }
}

function Remove-PathIfExists([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path)) {
        return
    }
    try {
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction Stop
        Write-Localized ($messages.Remove -f $Path)
    }
    catch {
        Write-Localized ($messages.Warn -f $_.Exception.Message)
    }
}

function Get-ProfileRoots {
    if (-not $AllUsers) {
        return @($env:USERPROFILE)
    }
    $usersRoot = Join-Path $env:SystemDrive "Users"
    if (-not (Test-Path -LiteralPath $usersRoot)) {
        return @($env:USERPROFILE)
    }
    Get-ChildItem -LiteralPath $usersRoot -Directory -Force |
        Where-Object { $_.Name -notin @("All Users", "Default", "Default User", "Public") } |
        ForEach-Object { $_.FullName }
}

function Test-IsSafeInstallDir([string]$Path) {
    if ([string]::IsNullOrWhiteSpace($Path) -or -not (Test-Path -LiteralPath $Path)) {
        return $false
    }
    $resolved = (Resolve-Path -LiteralPath $Path).Path
    $leaf = Split-Path -Path $resolved -Leaf
    if ($leaf -notmatch "krayn|krayN") {
        return $false
    }
    $markers = @(
        (Join-Path $resolved "krayn.exe"),
        (Join-Path $resolved "krayn-core.exe"),
        (Join-Path $resolved "unins000.exe"),
        (Join-Path $resolved "uninstall-krayN.ps1")
    )
    foreach ($marker in $markers) {
        if (Test-Path -LiteralPath $marker) {
            return $true
        }
    }
    return $false
}

function Get-CandidateInstallDirs {
    $candidateDirs = @()
    if (-not [string]::IsNullOrWhiteSpace($InstallDir)) {
        $candidateDirs += $InstallDir
    }
    if ($PSScriptRoot -and (Test-Path -LiteralPath $PSScriptRoot)) {
        $candidateDirs += $PSScriptRoot
    }
    if ($env:ProgramFiles) {
        $candidateDirs += (Join-Path $env:ProgramFiles "krayN")
    }
    $programFilesX86 = [Environment]::GetEnvironmentVariable("ProgramFiles(x86)")
    if ($programFilesX86) {
        $candidateDirs += (Join-Path $programFilesX86 "krayN")
    }
    $candidateDirs | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique
}

function Clear-InstallDirContents([string]$Path) {
    if (-not (Test-IsSafeInstallDir $Path)) {
        if (-not [string]::IsNullOrWhiteSpace($Path)) {
            Write-Localized ($messages.Skip -f $Path)
        }
        return
    }
    $items = @(
        "krayn.exe",
        "krayn-core.exe",
        "flutter_windows.dll",
        "screen_retriever_windows_plugin.dll",
        "tray_manager_plugin.dll",
        "window_manager_plugin.dll",
        "uninstall-krayN.ps1",
        "data"
    )
    foreach ($item in $items) {
        Remove-PathIfExists (Join-Path $Path $item)
    }
}

Write-Localized $messages.Title
Write-Localized $messages.Stop
Get-Process -Name "krayn", "krayn-core" -ErrorAction SilentlyContinue |
    Stop-Process -Force -ErrorAction SilentlyContinue

Write-Localized $messages.Shortcuts
$shortcutRoots = @(
    (Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\krayN"),
    (Join-Path $env:ProgramData "Microsoft\Windows\Start Menu\Programs\krayN"),
    (Join-Path ([Environment]::GetFolderPath("Desktop")) "krayN.lnk"),
    (Join-Path $env:PUBLIC "Desktop\krayN.lnk")
)
foreach ($path in $shortcutRoots) {
    Remove-PathIfExists $path
}

if ($KeepProfiles) {
    Write-Localized $messages.Keep
}
else {
    Write-Localized $messages.Data
    foreach ($profileRoot in Get-ProfileRoots) {
        if ([string]::IsNullOrWhiteSpace($profileRoot)) {
            continue
        }
        $paths = @(
            (Join-Path $profileRoot "AppData\Roaming\krayN"),
            (Join-Path $profileRoot "AppData\Local\krayN"),
            (Join-Path $profileRoot "AppData\Local\krayn"),
            (Join-Path $profileRoot "AppData\Local\io.krayn.krayn")
        )
        foreach ($path in $paths) {
            Remove-PathIfExists $path
        }
    }
}

Write-Localized $messages.InstallFiles
foreach ($dir in Get-CandidateInstallDirs) {
    Clear-InstallDirContents $dir
}

if (-not $SkipInstallDir) {
    Write-Localized $messages.Install
    foreach ($dir in Get-CandidateInstallDirs) {
        if (Test-IsSafeInstallDir $dir -or (Test-Path -LiteralPath $dir)) {
            Remove-PathIfExists $dir
        }
        elseif (-not [string]::IsNullOrWhiteSpace($dir)) {
            Write-Localized ($messages.Skip -f $dir)
        }
    }
}

Write-Localized $messages.Done
