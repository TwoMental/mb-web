# Modbus Web

Modbus Web is a lightweight Go application that serves a single-page web UI for working with Modbus/TCP devices.  
It wraps a `gin` HTTP server, manages per-user connections to a Modbus target, and ships a browser experience for
reading register values and doing quick binary/decimal/hex conversions.

## Features
- Connect to Modbus targets over TCP or RTU (serial) with per-browser session management.
- Launch a local web UI (`/home`) that guides you through TCP host/port or RTU serial settings, including detected port suggestions.
- Read values from coils, discrete inputs, input registers, and holding registers by address.
- Convert results between signed/unsigned integers, binary, and hexadecimal representations at a glance.
- Optionally serve prebuilt binaries from the `downloads` directory so the UI can proxy file downloads.
- Expose JSON APIs (`/set-server`, `/get-value`, `/version-info`, `/allow-download`, `/serial-ports`, `/resource-list`) that the UI uses and that other clients can reuse.

## Getting Started

### Prerequisites
- Go 1.21 or newer
- A Modbus/TCP server you can reach from the machine running this project
- (Optional) Access to a serial adapter/port when working with Modbus RTU devices

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
2. Choose the **TCP** or **RTU** tab, fill in the relevant connection fields (host/port or serial parameters plus slave ID), then click **Connect**.
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
| `GET` | `/serial-ports` | Enumerate available serial ports for RTU connections. |
| `GET` | `/resource-list` | List files in the configured download folder (only when proxying). |
| `GET` | `/downloads/*` | Serve files from the download folder (only when proxying). |

## Project Layout

- `main.go` — HTTP server bootstrap, argument parsing, route registration.
- `internal/` — Modbus connection management, register helpers, and utilities.
- `static/index.html` — The single-page interface loaded at `/home`.
- `downloads/` — Optional artifacts exposed when `-proxyDownload` is enabled.

## TODOs
- [ ] Write
- [ ] Auto reconnect
- [ ] Read/Write block (multiple register)
- [ ] Setting upload/download
- [ ] Ignore register
- [ ] Auto refresh (only for read)
- [ ] Multi connection
- [x] Modbus RTU

## License

This project is distributed under the [Apache 2.0 License](LICENSE).
