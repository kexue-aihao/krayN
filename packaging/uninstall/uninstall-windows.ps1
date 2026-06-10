param(
    [string]$InstallDir = "",
    [switch]$AllUsers,
    [switch]$KeepProfiles,
    [switch]$SkipInstallDir,
    [switch]$Silent
)

$ErrorActionPreference = "Stop"
$isChinese = [System.Globalization.CultureInfo]::CurrentUICulture.Name -like "zh*"

$messages = if ($isChinese) {
    @{
        Title = "krayN 完整卸载"
        Stop = "正在停止 krayN 进程..."
        Shortcuts = "正在删除快捷方式..."
        Data = "正在删除用户配置和缓存..."
        Install = "正在删除安装目录..."
        Keep = "已按要求保留用户配置。"
        Done = "krayN 已完整卸载。"
        Skip = "跳过路径：{0}"
        Remove = "删除：{0}"
        Warn = "警告：{0}"
    }
}
else {
    @{
        Title = "krayN complete uninstall"
        Stop = "Stopping krayN processes..."
        Shortcuts = "Removing shortcuts..."
        Data = "Removing user configuration and cache..."
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
        (Join-Path $resolved "unins000.exe")
    )
    foreach ($marker in $markers) {
        if (Test-Path -LiteralPath $marker) {
            return $true
        }
    }
    return $false
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

if (-not $SkipInstallDir) {
    Write-Localized $messages.Install
    $candidateDirs = @()
    if (-not [string]::IsNullOrWhiteSpace($InstallDir)) {
        $candidateDirs += $InstallDir
    }
    $candidateDirs += @(
        (Join-Path $env:ProgramFiles "krayN"),
        (Join-Path ${env:ProgramFiles(x86)} "krayN")
    )
    foreach ($dir in $candidateDirs | Select-Object -Unique) {
        if (Test-IsSafeInstallDir $dir) {
            Remove-PathIfExists $dir
        }
        elseif (-not [string]::IsNullOrWhiteSpace($dir)) {
            Write-Localized ($messages.Skip -f $dir)
        }
    }
}

Write-Localized $messages.Done
