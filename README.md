# krayN

krayN is a high-performance cross-platform graphical client for KLESS nodes, built on top of [kexue-aihao/kray](https://github.com/kexue-aihao/kray).

The current implementation contains:

- Go core sidecar using `kray/pkg/kless`.
- Local SOCKS5 proxy and encrypted KLESS outbound relay.
- HTTP control API for GUI/native integrations.
- Flutter Material 3 client for Windows, macOS, Linux, Android, and iOS UI work.
- Build matrix aligned with FlClash-style desktop/mobile packaging.

## Repository Layout

```text
app/                 Flutter GUI
core/                Go runtime sidecar and control API
docs/                Architecture and profile documentation
scripts/             Build helpers
third_party/kray     kray core submodule
```

## Quick Start

```powershell
git submodule update --init --recursive
cd core
go test ./...
go run ./cmd/krayn-core -gen-keys
go run ./cmd/krayn-core
```

Then open the Flutter app from `app/` on a machine with Flutter installed:

```powershell
.\scripts\scaffold-flutter.ps1
cd app
flutter pub get
flutter run -d windows
```

The GUI attempts to auto-start a desktop `krayn-core` binary from the same directory, the current directory, `core/`, or the `KRAYN_CORE` environment variable. During development you can also run the core manually.

## Control API

The core listens on `127.0.0.1:9727` by default.

- `GET /health`
- `GET /state`
- `GET /profiles`
- `POST /profiles`
- `PUT /profiles/{id}`
- `DELETE /profiles/{id}`
- `POST /profiles/{id}/activate`
- `POST /start`
- `POST /stop`

Local SOCKS5 listens on `127.0.0.1:7890` by default once started.

## Release Targets

Reference coverage follows FlClash v0.8.93:

- Windows `amd64`, `arm64`: installer/zip capable.
- macOS `amd64`, `arm64`: DMG capable.
- Linux `amd64`, `arm64`: AppImage/deb/rpm capable.
- Android `arm64-v8a`, `armeabi-v7a`, `x86_64`: APK capable.
- iOS / iPadOS: Flutter UI is ready; production distribution requires Apple signing and Network Extension work.

Build the core matrix:

```powershell
.\scripts\build-core.ps1 -Version 0.1.0 -Commit $(git rev-parse --short HEAD)
```

Android Go sidecar builds require Android NDK/cgo; use `-IncludeAndroid` only in an Android toolchain environment. The normal mobile deliverable is the Flutter APK/IPA plus native VPN integration.

Flutter packaging requires a local Flutter SDK:

```powershell
.\scripts\build-flutter.ps1
```

## GitHub Release Automation

Pushing a tag such as `v0.1.0` runs `.github/workflows/release.yml`, builds the release matrix, uploads `SHA256SUMS`, and publishes a GitHub Release automatically.

The workflow targets the same package family as FlClash v0.8.93: Android APKs for `arm64-v8a`, `armeabi-v7a`, and `x86_64`; Windows `amd64`/`arm64` zip and setup packages; macOS `amd64`/`arm64` DMG packages; Linux `amd64` AppImage/deb/rpm and Linux `arm64` deb.

Linux `arm64` currently publishes a core-sidecar deb because GitHub's stable Flutter Linux arm64 toolchain is not available through the hosted Flutter setup action.

## Notes

The first runtime provides SOCKS5 proxying through KLESS. Full-device VPN/TUN capture needs platform-native glue:

- Android `VpnService`
- iOS Network Extension packet tunnel
- Windows Wintun / system proxy integration
- macOS Network Extension / system proxy integration
- Linux tun2socks / system proxy integration

See [docs/architecture.md](docs/architecture.md) and [docs/profile-format.md](docs/profile-format.md).
