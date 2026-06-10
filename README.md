# krayN

## [中文](#中文) | [English](#english) | [完整卸载](#完整卸载) | [Complete Uninstall](#complete-uninstall) | [开源协议](#开源协议--license) | [GitHub Releases](https://github.com/kexue-aihao/krayN/releases) | [GitHub Actions](https://github.com/kexue-aihao/krayN/actions)

## 中文

krayN 是一个基于 [kexue-aihao/kray](https://github.com/kexue-aihao/kray) KLESS 核心构建的高性能、跨平台图形化代理节点客户端。

当前项目由 Flutter 图形界面、Go 运行时核心和 GitHub Actions 自动发布流水线组成，目标是提供接近 [FlClash v0.8.93](https://github.com/chen08209/FlClash/releases/tag/v0.8.93) 的平台覆盖能力。

### 功能概览

- 使用 `kray/pkg/kless` 作为 KLESS 协议核心。
- Go sidecar 提供本地 SOCKS5 代理、KLESS 出站连接、节点配置、流量统计和 HTTP 控制 API。
- Flutter Material 3 图形界面支持节点管理、启动/停止、状态查看和流量展示。
- 桌面端会尝试自动启动同目录、当前目录、`core/` 或 `KRAYN_CORE` 指定的 `krayn-core`。
- GitHub Actions 支持自动构建多平台包、生成 `SHA256SUMS` 并发布 GitHub Release。

### 仓库结构

```text
app/                 Flutter GUI
core/                Go runtime sidecar and control API
docs/                Architecture and profile documentation
scripts/             Build helpers
third_party/kray     kray core submodule
```

### 快速开始

```powershell
git submodule update --init --recursive
cd core
go test ./...
go run ./cmd/krayn-core -gen-keys
go run ./cmd/krayn-core
```

在安装 Flutter SDK 的机器上启动图形界面：

```powershell
.\scripts\scaffold-flutter.ps1
cd app
flutter pub get
flutter run -d windows
```

### 控制 API

默认控制 API 地址为 `127.0.0.1:9727`。

- `GET /health`
- `GET /state`
- `GET /profiles`
- `POST /profiles`
- `PUT /profiles/{id}`
- `DELETE /profiles/{id}`
- `POST /profiles/{id}/activate`
- `POST /start`
- `POST /stop`

启动后，本地 SOCKS5 默认监听 `127.0.0.1:7890`。

### 发布目标

发布包覆盖矩阵参考 FlClash v0.8.93：

- Windows `amd64` / `arm64`：zip 和 setup.exe。
- macOS `amd64` / `arm64`：DMG。
- Linux `amd64`：AppImage、deb、rpm。
- Linux `arm64`：deb。
- Android `arm64-v8a` / `armeabi-v7a` / `x86_64`：APK。
- iOS / iPadOS：Flutter UI 已准备，正式分发需要 Apple 签名和 Network Extension。

构建 Go core 矩阵：

```powershell
.\scripts\build-core.ps1 -Version 0.1.0 -Commit $(git rev-parse --short HEAD)
```

Android Go sidecar 构建需要 Android NDK/cgo；仅在 Android 工具链环境中使用 `-IncludeAndroid`。移动端正式代理能力通常需要 APK/IPA 加平台原生 VPN 集成。

Flutter 打包需要本机 Flutter SDK：

```powershell
.\scripts\build-flutter.ps1
```

### GitHub 自动发布

推送 `v*` 标签会触发 [.github/workflows/release.yml](.github/workflows/release.yml)，自动构建发布矩阵、生成 `SHA256SUMS`，并发布到 [GitHub Releases](https://github.com/kexue-aihao/krayN/releases)。

示例：

```powershell
git tag -a v0.1.1 -m "krayN v0.1.1"
git push origin master
git push origin v0.1.1
```

Linux `arm64` 当前发布 core-sidecar deb，因为 GitHub 托管 runner 上的 stable Flutter Linux arm64 工具链无法通过当前 Flutter setup action 稳定获取。

### 平台集成说明

第一版运行时提供 SOCKS5 到 KLESS 的代理能力。全局 VPN/TUN 捕获需要平台原生接入：

- Android `VpnService`
- iOS Network Extension packet tunnel
- Windows Wintun / 系统代理集成
- macOS Network Extension / 系统代理集成
- Linux tun2socks / 系统代理集成

更多说明见 [docs/architecture.md](docs/architecture.md) 和 [docs/profile-format.md](docs/profile-format.md)。

### 完整卸载

发布包会附带按系统语言自动显示中文/英文的卸载脚本。Windows `setup.exe` 卸载器会跟随 Windows UI 语言，并在卸载时清理安装目录、快捷方式、`%APPDATA%\krayN`、`%LOCALAPPDATA%\krayN` 等用户数据。

zip 版本可在解压目录运行：

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-krayN.ps1 -InstallDir "$PWD" -AllUsers
```

macOS / Linux 可运行随包附带的 `uninstall-krayN.sh`：

```bash
sudo /opt/krayN/uninstall-krayN.sh --all-users
```

Android 可通过系统设置卸载，或使用 `adb uninstall io.krayn.krayn`。完整路径和保留节点配置的命令见 [docs/uninstall.md](docs/uninstall.md)。

### 开源协议 / License

本仓库新增代码类型包括 Flutter/Dart GUI、Go runtime sidecar、PowerShell 构建脚本和 GitHub Actions 工作流。基于这类客户端应用和工具链代码的分发需求，krayN 采用宽松的 [MIT License](LICENSE) 开源。

`third_party/kray` 是上游 Git 子模块，版权和许可边界以其上游仓库声明为准。

---

## English

krayN is a high-performance cross-platform graphical client for KLESS nodes, built on top of [kexue-aihao/kray](https://github.com/kexue-aihao/kray).

The project combines a Flutter GUI, a Go runtime sidecar, and GitHub Actions release automation. Its package matrix is designed to follow the platform coverage style of [FlClash v0.8.93](https://github.com/chen08209/FlClash/releases/tag/v0.8.93).

### Features

- Uses `kray/pkg/kless` as the KLESS protocol core.
- Provides a Go sidecar for local SOCKS5 proxying, KLESS outbound connections, profile storage, traffic stats, and an HTTP control API.
- Provides a Flutter Material 3 GUI for profile management, start/stop controls, runtime status, and traffic display.
- Attempts to auto-start `krayn-core` from the app directory, current directory, `core/`, or the `KRAYN_CORE` environment variable on desktop platforms.
- Uses GitHub Actions to build multi-platform packages, generate `SHA256SUMS`, and publish GitHub Releases.

### Repository Layout

```text
app/                 Flutter GUI
core/                Go runtime sidecar and control API
docs/                Architecture and profile documentation
scripts/             Build helpers
third_party/kray     kray core submodule
```

### Quick Start

```powershell
git submodule update --init --recursive
cd core
go test ./...
go run ./cmd/krayn-core -gen-keys
go run ./cmd/krayn-core
```

Run the GUI on a machine with Flutter SDK installed:

```powershell
.\scripts\scaffold-flutter.ps1
cd app
flutter pub get
flutter run -d windows
```

### Control API

The control API listens on `127.0.0.1:9727` by default.

- `GET /health`
- `GET /state`
- `GET /profiles`
- `POST /profiles`
- `PUT /profiles/{id}`
- `DELETE /profiles/{id}`
- `POST /profiles/{id}/activate`
- `POST /start`
- `POST /stop`

After startup, the local SOCKS5 proxy listens on `127.0.0.1:7890` by default.

### Release Targets

The release package matrix follows FlClash v0.8.93:

- Windows `amd64` / `arm64`: zip and setup.exe.
- macOS `amd64` / `arm64`: DMG.
- Linux `amd64`: AppImage, deb, rpm.
- Linux `arm64`: deb.
- Android `arm64-v8a` / `armeabi-v7a` / `x86_64`: APK.
- iOS / iPadOS: Flutter UI is ready; production distribution requires Apple signing and Network Extension work.

Build the Go core matrix:

```powershell
.\scripts\build-core.ps1 -Version 0.1.0 -Commit $(git rev-parse --short HEAD)
```

Android Go sidecar builds require Android NDK/cgo; use `-IncludeAndroid` only in an Android toolchain environment. Mobile proxy support normally requires APK/IPA packaging plus native VPN integration.

Flutter packaging requires a local Flutter SDK:

```powershell
.\scripts\build-flutter.ps1
```

### GitHub Release Automation

Pushing a `v*` tag triggers [.github/workflows/release.yml](.github/workflows/release.yml), builds the release matrix, generates `SHA256SUMS`, and publishes to [GitHub Releases](https://github.com/kexue-aihao/krayN/releases).

Example:

```powershell
git tag -a v0.1.1 -m "krayN v0.1.1"
git push origin master
git push origin v0.1.1
```

Linux `arm64` currently publishes a core-sidecar deb because GitHub's hosted stable Flutter Linux arm64 toolchain is not reliably available through the current Flutter setup action.

### Platform Integration Notes

The first runtime provides SOCKS5-to-KLESS proxying. Full-device VPN/TUN capture requires platform-native integration:

- Android `VpnService`
- iOS Network Extension packet tunnel
- Windows Wintun / system proxy integration
- macOS Network Extension / system proxy integration
- Linux tun2socks / system proxy integration

See [docs/architecture.md](docs/architecture.md) and [docs/profile-format.md](docs/profile-format.md) for details.

### Complete Uninstall

Release packages include uninstall scripts that automatically display Chinese or English based on the operating system language. The Windows `setup.exe` uninstaller follows the Windows UI language and removes the install directory, shortcuts, `%APPDATA%\krayN`, `%LOCALAPPDATA%\krayN`, and related user data.

For zip builds, run this from the extracted directory:

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall-krayN.ps1 -InstallDir "$PWD" -AllUsers
```

For macOS / Linux, run the bundled `uninstall-krayN.sh`:

```bash
sudo /opt/krayN/uninstall-krayN.sh --all-users
```

Android can be removed from system Settings or with `adb uninstall io.krayn.krayn`. Full paths and keep-profile commands are documented in [docs/uninstall.md](docs/uninstall.md).

### License

This repository's own code includes Flutter/Dart GUI code, a Go runtime sidecar, PowerShell build scripts, and GitHub Actions workflows. For this type of client application and tooling code, krayN is released under the permissive [MIT License](LICENSE).

`third_party/kray` is an upstream Git submodule. Its copyright and licensing boundary follows the upstream repository.
