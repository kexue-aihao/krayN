# krayN

[中文](#中文) | [English](#english) | [功能](#功能) | [订阅格式](#订阅格式) | [构建发布](#构建发布) | [卸载](#卸载) | [许可证](#许可证) | [Releases](https://github.com/kexue-aihao/krayN/releases) | [Actions](https://github.com/kexue-aihao/krayN/actions)

## 中文

krayN 是一个面向 KLESS / kray 节点的图形化代理客户端。当前桌面端基于 [2dust/v2rayN](https://github.com/2dust/v2rayN) 的成熟客户端代码改造，使用 Avalonia 作为主要跨平台桌面界面，并通过本仓库的 Go 侧车核心 `krayn-core` 对接 [kexue-aihao/kray](https://github.com/kexue-aihao/kray)。

本项目当前重点不是继续保留完整 v2rayN 多核心生态，而是把客户端收敛到 kray / KLESS 场景：导入订阅、启动本地代理、管理 kray 核心、测试节点质量、打包桌面发行版。

## 功能

- KLESS 节点管理：支持新增、编辑、分享、订阅导入和订阅更新。
- kray 核心集成：客户端启动时会生成 krayN 原生配置，并启动内置 `krayn-core`。
- 本地代理：由 `krayn-core` 提供本地 SOCKS 代理和控制 API。
- 系统代理：保留 v2rayN 桌面客户端中的系统代理管理能力。
- 订阅导入：支持 `kray://` URI，以及 KLESS / kray JSON 订阅。
- Kboard 兼容字段：支持 `client_id`、`client_secret`、`server_public_key`、`server_signing_key`、`kless_client_id`、`kless_client_secret` 等字段。
- 节点诊断：支持真实 HTTPS 延迟、TCP RTT、10 次 RTT 平均值、最大 RTT、RTT 标准差/抖动率、失败率/丢包率。
- 节点 IP 信息：通过代理出口查询 IPv4 IP 信息，避免优先展示 IPv6。
- 测速：保留下载测速入口，可使用默认测速 URL 或自定义测速 URL。
- UDP 类型：当前 krayN 本地链路以 TCP 代理为主，真实节点出口 UDP NAT 类型检测暂显示为不支持，等待核心 UDP relay / STUN 链路完善后接入。
- 版本显示：客户端版本由 Release tag 注入，例如 `v0.1.29` 会显示为 `V0.1.29`。
- 官网入口：客户端帮助菜单指向 [krayN Releases](https://github.com/kexue-aihao/krayN/releases)。
- 默认中文界面：当前默认界面语言为简体中文，同时保留上游已有多语言资源。
- 完整卸载：Windows 发行包内置 `krayn-uninstall.exe`，用于清理程序文件、配置、缓存和内置 kray 核心。

## 项目结构

- `v2rayN/`：当前主要桌面客户端，来自 v2rayN 源码改造；Release 主路径使用 `v2rayN.Desktop` Avalonia 项目。
- `core/`：Go 编写的 `krayn-core`，负责本地代理、KLESS 出站、控制 API、诊断和 Windows 卸载器。
- `third_party/kray/`：kray / KLESS 协议核心子模块。
- `packaging/`：安装、卸载和跨平台打包相关资源。
- `scripts/`：构建和辅助脚本。
- `docs/`：架构、配置格式、卸载说明等文档。
- `app/`：早期 Flutter 客户端代码，当前不作为 Release 主客户端。

## 订阅格式

### kray URI

```text
kray://client-id@example.com:8443?transport=tcp&client_secret=CLIENT_SECRET&server_public_key=SERVER_PUBLIC_KEY&server_name=edge.example#JP-01
```

常用参数：

- `transport`：`tcp`、`tls`、`websocket`、`http-upgrade`、`http-stream`、`grpc`、`xhttp`。
- `client_secret` / `kless_client_secret`：客户端密钥。
- `server_public_key` / `server_signing_key`：服务端公钥。
- `server_name` / `sni`：TLS SNI 或服务端名称。
- `skip_tls_verify`：是否跳过 TLS 证书校验。
- `headers`：WebSocket / HTTP 类传输使用的请求头 JSON。
- `padding_min`、`padding_max`：KLESS padding 参数。

### JSON 订阅

支持数组、单节点对象、`profiles`、`nodes` 和 `profile` 包装格式。

```json
{
  "profiles": [
    {
      "name": "JP-01",
      "endpoint": "example.com:8443",
      "transport": "tcp",
      "client_id": "kboard-node-3",
      "client_secret": "client-secret",
      "server_public_key": "server-public-key",
      "server_name": "example.com",
      "skip_tls_verify": false
    }
  ]
}
```

也支持 Kboard 风格字段：

```json
{
  "nodes": [
    {
      "name": "JP-01",
      "endpoint": "example.com:8443",
      "transport": "tcp",
      "kless_client_id": "kboard-node-3",
      "kless_client_secret": "client-secret",
      "server_signing_key": "server-public-key"
    }
  ]
}
```

## Knode 接入提示

如果节点由 Knode 管理，krayN 客户端必须连接对公网开放的 KLESS 服务端入站，也就是模式为 `kless-server` 的入站，常见名称是 `public-kless`。

`tcp` / `local-tcp` 这类入站通常只是内部转发端口，不能作为 krayN 客户端直连入口。如果日志出现 `server closed the connection before handshake completed`，请优先检查客户端连到的端口是否为公网 KLESS 入站，并确认 `client_id`、`client_secret`、`server_public_key` / `server_signing_key` 与服务端下发配置一致。

## 构建发布

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

构建 Windows 桌面端示例：

```powershell
dotnet publish v2rayN\v2rayN.Desktop\v2rayN.Desktop.csproj -c Release -r win-x64 --self-contained true -o dist\publish\windows-amd64
```

构建 Windows krayN 核心示例：

```powershell
cd core
go build -o ..\dist\core\krayn-core.exe .\cmd\krayn-core
go build -o ..\dist\core\krayn-uninstall.exe .\cmd\krayn-uninstall
```

桌面端运行时会在程序目录下查找核心：

```text
bin/kray/krayn-core.exe
```

Linux / macOS 对应为：

```text
bin/kray/krayn-core
```

推送 `v*` tag 会触发 [.github/workflows/release.yml](.github/workflows/release.yml)，自动构建并发布：

- Windows `amd64` / `arm64` zip。
- Windows `amd64` setup 安装包。
- Linux `amd64` / `arm64` tar.gz。
- Linux `amd64` / `arm64` deb。
- macOS `amd64` / `arm64` dmg。
- 独立 `krayn-core-*` 核心二进制。
- Windows 独立 `krayn-uninstall-*` 卸载器。
- `SHA256SUMS` 校验文件。

当前 Release 主目标是 Windows、Linux、macOS 桌面端。Android / iOS 需要移动端 VPN/TUN 壳层和系统权限适配，当前工作流不会直接产出移动端安装包。

## 卸载

Windows setup 安装版可通过系统“设置 -> 应用 -> 已安装的应用”卸载。卸载流程会调用 `krayn-uninstall.exe`，尽量清理安装目录、快捷方式、配置、缓存和内置 kray 核心。

Windows zip 解压版可在解压目录运行：

```powershell
.\krayn-uninstall.exe --install-dir "$PWD" --all-users
```

保留节点配置：

```powershell
.\krayn-uninstall.exe --install-dir "$PWD" --keep-profiles
```

macOS / Linux 包内包含 `uninstall-krayN.sh`，可从安装目录运行：

```bash
./uninstall-krayN.sh --all-users
```

保留节点配置：

```bash
./uninstall-krayN.sh --keep-profiles
```

卸载器会根据系统语言显示中文或英文提示。

## 许可证

本仓库包含并修改了 v2rayN 相关源码，整体按 [GPL-3.0](LICENSE) 开源和分发。

第三方组件遵循其各自许可证：

- `third_party/kray/`：以 [kexue-aihao/kray](https://github.com/kexue-aihao/kray) 上游许可证为准。
- `v2rayN/GlobalHotKeys/`：保留上游组件许可证声明。

---

## English

krayN is a graphical proxy client focused on KLESS / kray nodes. The current desktop client is rebuilt from the mature [2dust/v2rayN](https://github.com/2dust/v2rayN) codebase, uses Avalonia as the primary cross-platform desktop UI, and integrates the Go sidecar core `krayn-core` with [kexue-aihao/kray](https://github.com/kexue-aihao/kray).

The project is intentionally narrowed to the kray / KLESS use case instead of keeping the full v2rayN multi-core ecosystem: subscription import, local proxy startup, kray core management, node diagnostics, and desktop release packaging.

## Features

- KLESS node management: add, edit, share, import, and update subscriptions.
- kray core integration: the client generates native krayN config and launches the bundled `krayn-core`.
- Local proxy: `krayn-core` provides the local SOCKS proxy and control API.
- System proxy: keeps the desktop system proxy controls from v2rayN.
- Subscription import: supports `kray://` URI and KLESS / kray JSON subscriptions.
- Kboard-compatible fields: supports `client_id`, `client_secret`, `server_public_key`, `server_signing_key`, `kless_client_id`, `kless_client_secret`, and related names.
- Node diagnostics: real HTTPS latency, TCP RTT, 10-sample RTT average, max RTT, RTT standard deviation / jitter, and failure rate / packet loss.
- IP information: queries IPv4 egress IP information through the selected proxy.
- Speed test: keeps the download speed test entry with default or custom test URLs.
- UDP type: the current local path is mainly TCP proxying, so real egress UDP NAT type detection is shown as unsupported until UDP relay / STUN support lands in the core.
- Version display: release tags are injected into the client version, for example `v0.1.29` becomes `V0.1.29`.
- Project link: the Help menu points to [krayN Releases](https://github.com/kexue-aihao/krayN/releases).
- Default Chinese UI: Simplified Chinese is the default UI language, while existing upstream localization resources are kept.
- Complete uninstall: Windows packages include `krayn-uninstall.exe` for cleaning app files, config, cache, and bundled kray core files.

## Repository Layout

- `v2rayN/`: the main desktop client, rebuilt from v2rayN sources. Current releases use the `v2rayN.Desktop` Avalonia project.
- `core/`: Go `krayn-core`, including the local proxy, KLESS outbound, control API, diagnostics, and Windows uninstaller.
- `third_party/kray/`: kray / KLESS protocol core submodule.
- `packaging/`: installer, uninstaller, and packaging resources.
- `scripts/`: build and helper scripts.
- `docs/`: architecture, profile format, uninstall notes, and related docs.
- `app/`: earlier Flutter client code, kept for reference and not used as the current release client.

## Subscription Format

### kray URI

```text
kray://client-id@example.com:8443?transport=tcp&client_secret=CLIENT_SECRET&server_public_key=SERVER_PUBLIC_KEY&server_name=edge.example#JP-01
```

Common parameters:

- `transport`: `tcp`, `tls`, `websocket`, `http-upgrade`, `http-stream`, `grpc`, `xhttp`.
- `client_secret` / `kless_client_secret`: client secret.
- `server_public_key` / `server_signing_key`: server public key.
- `server_name` / `sni`: TLS SNI or server name.
- `skip_tls_verify`: skip TLS certificate verification.
- `headers`: JSON headers for WebSocket / HTTP transports.
- `padding_min`, `padding_max`: KLESS padding parameters.

### JSON Subscription

Array, single object, `profiles`, `nodes`, and `profile` wrappers are supported.

```json
{
  "profiles": [
    {
      "name": "JP-01",
      "endpoint": "example.com:8443",
      "transport": "tcp",
      "client_id": "kboard-node-3",
      "client_secret": "client-secret",
      "server_public_key": "server-public-key",
      "server_name": "example.com",
      "skip_tls_verify": false
    }
  ]
}
```

Kboard-style field names are also supported:

```json
{
  "nodes": [
    {
      "name": "JP-01",
      "endpoint": "example.com:8443",
      "transport": "tcp",
      "kless_client_id": "kboard-node-3",
      "kless_client_secret": "client-secret",
      "server_signing_key": "server-public-key"
    }
  ]
}
```

## Knode Notes

When a node is managed by Knode, krayN must connect to the public KLESS server inbound, usually an inbound in `kless-server` mode and often named `public-kless`.

`tcp` / `local-tcp` inbounds are usually internal forwarding ports and cannot be used as direct krayN client endpoints. If the log says `server closed the connection before handshake completed`, first verify that the endpoint is a public KLESS inbound and that `client_id`, `client_secret`, and `server_public_key` / `server_signing_key` match the server-issued values.

## Build And Release

Initialize submodules:

```powershell
git submodule update --init --recursive
```

Test the Go core:

```powershell
cd core
go test ./...
```

Test the .NET client library:

```powershell
dotnet restore v2rayN\ServiceLib.Tests\ServiceLib.Tests.csproj
dotnet test v2rayN\ServiceLib.Tests\ServiceLib.Tests.csproj -c Debug
```

Build the Windows desktop client:

```powershell
dotnet publish v2rayN\v2rayN.Desktop\v2rayN.Desktop.csproj -c Release -r win-x64 --self-contained true -o dist\publish\windows-amd64
```

Build the Windows krayN core:

```powershell
cd core
go build -o ..\dist\core\krayn-core.exe .\cmd\krayn-core
go build -o ..\dist\core\krayn-uninstall.exe .\cmd\krayn-uninstall
```

At runtime, the desktop client looks for the core at:

```text
bin/kray/krayn-core.exe
```

On Linux / macOS:

```text
bin/kray/krayn-core
```

Pushing a `v*` tag triggers [.github/workflows/release.yml](.github/workflows/release.yml), which publishes:

- Windows `amd64` / `arm64` zip packages.
- Windows `amd64` setup installer.
- Linux `amd64` / `arm64` tar.gz packages.
- Linux `amd64` / `arm64` deb packages.
- macOS `amd64` / `arm64` dmg packages.
- Standalone `krayn-core-*` binaries.
- Standalone Windows `krayn-uninstall-*` binaries.
- `SHA256SUMS`.

Current releases target Windows, Linux, and macOS desktop. Android / iOS require mobile VPN/TUN shells and platform permission handling, so the current workflow does not produce mobile installers.

## Uninstall

For the Windows setup installer, uninstall krayN from Settings -> Apps -> Installed apps. The uninstall flow calls `krayn-uninstall.exe` to clean the install directory, shortcuts, config, cache, and bundled kray core.

For the Windows zip package, run from the extracted directory:

```powershell
.\krayn-uninstall.exe --install-dir "$PWD" --all-users
```

Keep node profiles:

```powershell
.\krayn-uninstall.exe --install-dir "$PWD" --keep-profiles
```

macOS / Linux packages include `uninstall-krayN.sh`:

```bash
./uninstall-krayN.sh --all-users
```

Keep node profiles:

```bash
./uninstall-krayN.sh --keep-profiles
```

The uninstaller displays Chinese or English prompts according to the operating system language.

## License

This repository includes and modifies v2rayN source code, so the project is distributed under [GPL-3.0](LICENSE).

Third-party components keep their own license terms:

- `third_party/kray/`: follows the upstream [kexue-aihao/kray](https://github.com/kexue-aihao/kray) license.
- `v2rayN/GlobalHotKeys/`: keeps the upstream component license notice.
