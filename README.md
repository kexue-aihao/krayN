# krayN

## [中文](#中文) | [English](#english) | [构建与发布](#构建与发布) | [卸载](#卸载--uninstall) | [许可证](#许可证--license) | [GitHub Releases](https://github.com/kexue-aihao/krayN/releases) | [GitHub Actions](https://github.com/kexue-aihao/krayN/actions)

## 中文

krayN 是一个基于 [2dust/v2rayN](https://github.com/2dust/v2rayN) 桌面客户端源码改造、使用 [kexue-aihao/kray](https://github.com/kexue-aihao/kray) / KLESS 作为核心出站能力的图形化代理节点客户端。

本仓库已经从早期自研 Flutter/Go 客户端架构迁移为“v2rayN 成熟桌面外壳 + krayN Go 核心”的架构，目标是优先解决节点导入、系统代理、本地代理端口、核心进程管理和多平台桌面打包的可用性问题。

### 当前架构

- `v2rayN/`: 主客户端，来自 v2rayN 的 WPF / Avalonia 桌面源码，当前发布主路径使用 Avalonia 跨平台桌面端。
- `core/`: krayN Go 核心，提供本地混合代理端口、KLESS 出站连接、诊断能力和完整卸载器。
- `third_party/kray`: kray / KLESS 核心协议代码。
- `app/`: 早期 Flutter 客户端代码，暂时保留作历史参考，不再作为当前 Release 主客户端。

### 已接入能力

- 支持 `kray://` 节点 URI 导入和导出。
- 支持 KLESS / kray JSON 订阅导入，包括 `profiles`、`nodes`、`profile` 和单节点对象。
- v2rayN 节点模型中新增 `KLESS` 协议和 `kray` 核心类型。
- 启动代理时会生成 krayN 原生配置，并自动启动 `krayn-core`。
- 发布包会内置 `bin/kray/krayn-core(.exe)`，符合 v2rayN 的核心查找规则。
- 默认界面语言为中文。
- Windows 发布包包含 `krayn-uninstall.exe`，卸载时会清理应用、配置、缓存和内置 kray 核心。

### KLESS 订阅格式

URI 示例：

```text
kray://client-id@example.com:8443?transport=websocket&client_secret=...&server_public_key=...&server_name=edge.example#HK-01
```

JSON 订阅示例：

```json
{
  "profiles": [
    {
      "name": "HK-01",
      "endpoint": "example.com:8443",
      "client_id": "client-id",
      "client_secret": "client-secret",
      "server_public_key": "server-public-key",
      "transport": "websocket",
      "server_name": "edge.example",
      "headers": {
        "Host": "edge.example"
      },
      "padding_min": 8,
      "padding_max": 32
    }
  ]
}
```

### 本地开发

初始化子模块：

```powershell
git submodule update --init --recursive
```

测试 Go 核心：

```powershell
cd core
go test ./...
```

测试 .NET 客户端核心库：

```powershell
dotnet restore v2rayN\ServiceLib.Tests\ServiceLib.Tests.csproj
dotnet test v2rayN\ServiceLib.Tests\ServiceLib.Tests.csproj -c Debug
```

构建 Avalonia 桌面端：

```powershell
dotnet publish v2rayN\v2rayN.Desktop\v2rayN.Desktop.csproj -c Release -r win-x64 --self-contained true -o dist\publish\windows-amd64
```

构建 krayN 核心：

```powershell
cd core
go build -o ..\dist\core\krayn-core.exe .\cmd\krayn-core
```

本地运行时，请把核心放到桌面程序目录下的：

```text
bin/kray/krayn-core.exe
```

### 构建与发布

推送 `v*` tag 会触发 [.github/workflows/release.yml](.github/workflows/release.yml)，自动构建：

- Windows `amd64` / `arm64`: zip，Windows `amd64` 额外生成 setup.exe。
- Linux `amd64` / `arm64`: tar.gz 和 deb。
- macOS `amd64` / `arm64`: dmg。
- 独立 `krayn-core-*` 核心文件和 Windows `krayn-uninstall-*` 卸载器。
- `SHA256SUMS` 校验文件。

说明：当前迁移基于 v2rayN 桌面源码，Release 主目标是 Windows / macOS / Linux 桌面端。Android / iOS 需要单独实现移动端 VPN/TUN 壳，不能直接由 v2rayN 桌面源码产出。

### 卸载 / Uninstall

Windows 安装版可通过系统“应用和功能”卸载。zip 版可在解压目录运行：

```powershell
.\krayn-uninstall.exe --install-dir "$PWD" --all-users
```

卸载器会按系统 UI 语言显示中文或英文，并尽量清理：

- `krayN.exe`
- `bin/kray/krayn-core.exe`
- `krayn-uninstall.exe`
- `binConfigs`、`guiConfigs`、`guiLogs`、`guiTemps`
- `%APPDATA%\krayN`、`%LOCALAPPDATA%\krayN`
- 旧版本可能留下的 `%APPDATA%\v2rayN`、`%LOCALAPPDATA%\v2rayN`

macOS / Linux 包内会带 `uninstall-krayN.sh`，可按安装位置执行清理。

### 许可证 / License

本仓库当前包含并修改了 v2rayN 源码，整体按 [GPL-3.0](LICENSE) 开源和分发。`v2rayN/GlobalHotKeys` 保留其上游许可证声明；`third_party/kray` 以其上游仓库许可证声明为准。

---

## English

krayN is a graphical proxy client rebuilt on top of [2dust/v2rayN](https://github.com/2dust/v2rayN), using [kexue-aihao/kray](https://github.com/kexue-aihao/kray) / KLESS as the core outbound engine.

The project has moved from the earlier custom Flutter/Go client into a “v2rayN desktop shell + krayN Go core” architecture. The immediate goal is practical usability: reliable subscription import, system proxy integration, local proxy ports, core process management, and desktop release packaging.

### Architecture

- `v2rayN/`: main client based on v2rayN WPF / Avalonia sources. Current releases use the Avalonia cross-platform desktop app.
- `core/`: krayN Go core for the local mixed proxy, KLESS outbound transport, diagnostics, and Windows uninstaller.
- `third_party/kray`: kray / KLESS protocol core.
- `app/`: earlier Flutter client code, kept for reference but no longer used as the main release client.

### Highlights

- Imports and exports `kray://` node URIs.
- Imports KLESS / kray JSON subscriptions from `profiles`, `nodes`, `profile`, or a single object.
- Adds the `KLESS` protocol and `kray` core type to the v2rayN model.
- Generates native krayN core config and starts `krayn-core` automatically.
- Bundles the core at `bin/kray/krayn-core(.exe)`, matching v2rayN core discovery.
- Defaults the UI language to Chinese.
- Includes `krayn-uninstall.exe` in Windows packages for complete cleanup.

### Build

```powershell
git submodule update --init --recursive
cd core
go test ./...
cd ..
dotnet test v2rayN\ServiceLib.Tests\ServiceLib.Tests.csproj -c Debug
dotnet publish v2rayN\v2rayN.Desktop\v2rayN.Desktop.csproj -c Release -r win-x64 --self-contained true -o dist\publish\windows-amd64
```

### Releases

Pushing a `v*` tag triggers [.github/workflows/release.yml](.github/workflows/release.yml). It publishes Windows, Linux, and macOS desktop packages, standalone core binaries, the Windows uninstaller, and `SHA256SUMS`.

Android and iOS are not produced from this v2rayN desktop migration. Mobile support needs a separate VPN/TUN application shell.

### License

Because this repository now includes and modifies v2rayN source code, the combined project is distributed under [GPL-3.0](LICENSE).
