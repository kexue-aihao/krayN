# Profile Format

Profiles are stored in the core config file, usually under the operating system config directory:

- Windows: `%AppData%\krayN\config.json`
- macOS: `~/Library/Application Support/krayN/config.json`
- Linux: `~/.config/krayN/config.json`

Minimal profile:

```json
{
  "name": "demo",
  "transport": "tcp",
  "endpoint": "127.0.0.1:9000",
  "client_id": "krayn-demo",
  "client_secret": "base64url-secret-from-krayn-core-gen-keys",
  "server_public_key": "base64url-public-key-from-server"
}
```

Supported transports in the current core:

- `tcp`: raw KLESS over TCP.
- `tls`: KLESS over TLS 1.3.
- `websocket`: KLESS over WebSocket or WSS.
- `http-upgrade`: HTTP/1.1 upgrade.
- `http-stream`: streaming HTTP POST response.
- `grpc`: gRPC-style HTTP/2 framed stream.
- `xhttp`: split upload/download HTTP streams.

Generate demo key material:

```powershell
go run ./cmd/krayn-core -gen-keys
```

