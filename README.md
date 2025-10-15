# Modbus Web

Modbus Web is a lightweight Go application that serves a single-page web UI for working with Modbus/TCP devices.  
It wraps a `gin` HTTP server, manages per-user connections to a Modbus target, and ships a browser experience for
reading register values and doing quick binary/decimal/hex conversions.

## Features
- Launches a local web UI (`/home`) that can connect to a Modbus/TCP server and keep the session alive per browser.
- Reads values from coils, discrete inputs, input registers, and holding registers by address.
- Provides at-a-glance conversions between signed/unsigned integers, binary, and hexadecimal representations.
- Optionally serves prebuilt binaries from the `downloads` directory so the UI can proxy file downloads.
- Exposes JSON APIs (`/set-server`, `/get-value`, `/version-info`, `/allow-download`, `/resource-list`) that the UI uses and that other clients can reuse.

## Getting Started

### Prerequisites
- Go 1.21 or newer
- A Modbus/TCP server you can reach from the machine running this project

### Run in place
```bash
go run ./main.go
```

By default the server listens on port `80` and opens `http://127.0.0.1:80/home` in your default browser.

### Build

Install the dependencies and build a local binary:
```bash
go build -o modbus-web ./main.go
```

The provided `Makefile` cross-compiles ready-to-download binaries for common targets and drops them in `./downloads/`:
```bash
make build-all
```

## Command-line flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `-listenPort` | `80` | Port the HTTP server listens on. |
| `-proxyDownload` | `false` | If set, exposes `/downloads` and `/resource-list` so the UI can serve local files. |
| `-downloadFolder` | `downloads` | Folder to serve when `proxyDownload` is enabled. |
| `-v` | `false` | Print build metadata and exit. |

## Using the Web UI

1. Open the app in your browser (the server will usually launch it automatically).
2. Enter the Modbus host, port, and slave ID, then click **Connect** to establish the session.
3. Add addresses with the **Add Address** button, pick the register type, and optionally label each address.
4. Click **Read Values** to query the selected addresses. Results include raw bytes for quick inspection.
5. Use the conversion card at the top to translate between decimal, binary, and hex, or double-click the values in the results table to auto-fill the converter.

## API Overview

All endpoints respond with JSON.

| Method | Path | Description |
| ------ | ---- | ----------- |
| `POST` | `/set-server` | Bind the current user session to a Modbus host/port/slave ID. |
| `POST` | `/get-value` | Read values for a list of addresses (`register_type`, `address`). |
| `GET` | `/version-info` | Return build time and git commit (populated via linker flags). |
| `GET` | `/allow-download` | Indicate whether download proxying is enabled. |
| `GET` | `/resource-list` | List files in the configured download folder (only when proxying). |
| `GET` | `/downloads/*` | Serve files from the download folder (only when proxying). |

## Project Layout

- `main.go` — HTTP server bootstrap, argument parsing, route registration.
- `internal/` — Modbus connection management, register helpers, and utilities.
- `static/index.html` — The single-page interface loaded at `/home`.
- `downloads/` — Optional artifacts exposed when `-proxyDownload` is enabled.

## License

This project is distributed under the [Apache 2.0 License](LICENSE).
