# 完整卸载 / Complete Uninstall

## 中文

krayN 的完整卸载包含三部分：停止运行中的 `krayn` / `krayn-core` 进程、删除安装文件和快捷方式、删除用户配置与缓存。

卸载脚本会根据系统语言自动显示中文或英文：

- Windows 使用当前 Windows UI 语言。
- macOS / Linux 使用 `LC_ALL`、`LC_MESSAGES` 或 `LANG`。

如果你需要保留节点配置，请在运行脚本时加 `--keep-profiles` 或 `-KeepProfiles`。

### Windows

使用 `setup.exe` 安装的版本：

1. 打开“设置” -> “应用” -> “已安装的应用”。
2. 找到 `krayN` 并点击“卸载”。
3. 卸载器会按 Windows 系统语言显示界面，并自动清理安装目录、快捷方式和 krayN 用户数据。

使用 zip 解压的版本，在解压目录运行：

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-krayN.ps1 -InstallDir "$PWD" -AllUsers
```

会删除：

- 安装目录中的 `krayn.exe`、`krayn-core.exe` 和随附文件。
- 开始菜单和桌面快捷方式。
- `%APPDATA%\krayN`
- `%LOCALAPPDATA%\krayN`
- `%LOCALAPPDATA%\krayn`
- `%LOCALAPPDATA%\io.krayn.krayn`

只卸载程序但保留节点配置：

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-krayN.ps1 -InstallDir "$PWD" -KeepProfiles
```

### macOS

DMG 内会附带 `uninstall-krayN.sh`。也可以从应用包内运行：

```bash
bash "/Applications/krayn.app/Contents/Resources/uninstall-krayN.sh"
```

会删除：

- `/Applications/krayn.app` 或 `/Applications/krayN.app`
- `~/Library/Application Support/krayN`
- `~/Library/Caches/krayN`
- `~/Library/Preferences/io.krayn.krayn.plist`
- `~/Library/Saved Application State/io.krayn.krayn.savedState`

保留节点配置：

```bash
bash "/Applications/krayn.app/Contents/Resources/uninstall-krayN.sh" --keep-profiles
```

### Linux

deb / rpm 安装后可以运行：

```bash
sudo /opt/krayN/uninstall-krayN.sh --all-users
```

脚本会先尝试通过 `dpkg` 或 `rpm` 删除系统包，然后清理：

- `/opt/krayN`
- `${XDG_CONFIG_HOME:-~/.config}/krayN`
- `${XDG_CACHE_HOME:-~/.cache}/krayN`
- `${XDG_DATA_HOME:-~/.local/share}/krayN`
- `${XDG_DATA_HOME:-~/.local/share}/krayn`
- 本地桌面入口和图标缓存路径

AppImage 版本可以从挂载后的 AppImage 内执行同名脚本，或手动删除 AppImage 文件后清理上述用户目录。

保留节点配置：

```bash
/opt/krayN/uninstall-krayN.sh --keep-profiles
```

### Android

通过系统设置卸载：

1. 打开“设置” -> “应用”。
2. 找到 `krayN`。
3. 点击“卸载”。

使用 adb：

```bash
adb uninstall io.krayn.krayn
```

如果需要先清理应用数据：

```bash
adb shell pm clear io.krayn.krayn
adb uninstall io.krayn.krayn
```

---

## English

A complete krayN uninstall has three parts: stop running `krayn` / `krayn-core` processes, remove installed files and shortcuts, and remove user configuration and cache data.

The uninstall scripts display Chinese or English automatically:

- Windows uses the current Windows UI language.
- macOS / Linux use `LC_ALL`, `LC_MESSAGES`, or `LANG`.

Use `--keep-profiles` or `-KeepProfiles` if you want to keep node profiles.

### Windows

For the `setup.exe` package:

1. Open Settings -> Apps -> Installed apps.
2. Find `krayN` and choose Uninstall.
3. The uninstaller follows the Windows UI language and cleans the install directory, shortcuts, and krayN user data.

For the zip package, run this from the extracted directory:

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-krayN.ps1 -InstallDir "$PWD" -AllUsers
```

It removes:

- `krayn.exe`, `krayn-core.exe`, and bundled files in the install directory.
- Start Menu and desktop shortcuts.
- `%APPDATA%\krayN`
- `%LOCALAPPDATA%\krayN`
- `%LOCALAPPDATA%\krayn`
- `%LOCALAPPDATA%\io.krayn.krayn`

Remove the app but keep node profiles:

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-krayN.ps1 -InstallDir "$PWD" -KeepProfiles
```

### macOS

The DMG includes `uninstall-krayN.sh`. You can also run the script from the app bundle:

```bash
bash "/Applications/krayn.app/Contents/Resources/uninstall-krayN.sh"
```

It removes:

- `/Applications/krayn.app` or `/Applications/krayN.app`
- `~/Library/Application Support/krayN`
- `~/Library/Caches/krayN`
- `~/Library/Preferences/io.krayn.krayn.plist`
- `~/Library/Saved Application State/io.krayn.krayn.savedState`

Keep node profiles:

```bash
bash "/Applications/krayn.app/Contents/Resources/uninstall-krayN.sh" --keep-profiles
```

### Linux

For deb / rpm packages:

```bash
sudo /opt/krayN/uninstall-krayN.sh --all-users
```

The script first tries to remove the package through `dpkg` or `rpm`, then cleans:

- `/opt/krayN`
- `${XDG_CONFIG_HOME:-~/.config}/krayN`
- `${XDG_CACHE_HOME:-~/.cache}/krayN`
- `${XDG_DATA_HOME:-~/.local/share}/krayN`
- `${XDG_DATA_HOME:-~/.local/share}/krayn`
- local desktop entries and icon paths

For AppImage builds, run the bundled script from the mounted AppImage, or delete the AppImage file and then remove the user directories listed above.

Keep node profiles:

```bash
/opt/krayN/uninstall-krayN.sh --keep-profiles
```

### Android

Through system Settings:

1. Open Settings -> Apps.
2. Find `krayN`.
3. Tap Uninstall.

Using adb:

```bash
adb uninstall io.krayn.krayn
```

Clear app data first when needed:

```bash
adb shell pm clear io.krayn.krayn
adb uninstall io.krayn.krayn
```
