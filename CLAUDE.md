# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**0E7** is an AWD (Attack With Defense) CTF/security-competition toolbox. It is a single Go binary (`0e7.go`, module name `0E7`) that can run as a **server**, a **client**, or both at once, controlled by `config.ini`. A Vue 3 SPA is embedded into the binary at compile time and served by the server. There are also desktop-app wrappers (Wails, Electron) that bundle the backend binary.

User-facing strings, comments, and logs are predominantly in Chinese — match this convention when editing.

## Build & Run Commands

### Prerequisites
- Go 1.25, Node.js 24+, and (because of CGO) a C toolchain.
- Linux: `libpcap-dev` required (`sudo apt-get install -y libpcap-dev`).
- The `0e7.go` binary does `//go:embed dist`, so **the frontend must be built into `dist/` before `go build`** — a raw `go build` fails if `dist/` does not exist.

### Frontend (Vue 3 + Vite)
```bash
cd frontend
npm install
npm run build         # type-check (vue-tsc) + vite build → outputs to ../dist
npm run build-only    # vite build only, skips type-check (used by build.sh)
npm run type-check    # vue-tsc --noEmit
npm run dev           # dev server; proxies /api and /webui to http://localhost:6102
```
`npm` is the package manager used by all build scripts, even though a `pnpm-lock.yaml` exists.

### Go backend
```bash
go build .                       # requires dist/ to exist; needs CGO (sqlite + gopacket/libpcap)
go build -ldflags="-s -w" -trimpath -o 0e7_local .
```

### Full builds (orchestrate frontend + backend)
- `./build.sh` — current platform only (the canonical local build; also runs UPX if present).
- `./build-advanced.sh` — cross-platform / clean / compress / release modes (`-a` all platforms, `-p windows|linux|darwin`, `-r` release). Requires a cross-compile toolchain for non-host targets.
- `./build-wails.sh` — Wails desktop app. Embeds the backend binary as `wails/bin/backend.bin` (so a backend build must exist first unless `--skip-backend`).
- `./build-electron.sh` — Electron desktop app; backend binary goes in `electron/resources/bin/`.

### Run
The server listens on **two ports**: `server.port` (default 6102, the C/S protocol clients connect to) and `server.admin_port` (default 6103, the admin web UI). On first `--server` start, `config.ini` is generated and `cs_token` / `admin_password` are **randomly generated, written back to `config.ini`, and printed in the log**.
```bash
./0e7_<os>_<arch> --server              # server mode; generates config.ini + secrets if absent
./0e7_<os>_<arch> -config config.ini    # run with a specific config (server/client behavior per [server]/[client] sections)
./0e7_<os>_<arch> --server -p 6200      # override C/S port (server.port); also via OE7_SERVER_PORT env
./0e7_<os>_<arch> --server --admin-port 6300   # override admin port (server.admin_port); also via OE7_ADMIN_PORT env
```
`--cpu-profile` / `--mem-profile` produce pprof files; `--log-file` redirects the rotating log.

Go tests exist for the auth layer — run `go test ./service/mw/ ./service/config/ ./utils/`. There is no linter config. CI (`.github/workflows/build.yml`) builds all platforms on `push` to `main` (artifacts only) and on `v*` tags (full release); the `release` job fires only on tags.

## Architecture

### One binary, two roles (`0e7.go` `main`)
- **Server mode** (`config.Server_mode`, set via `--server` flag): boots a Gin HTTP server, serves the embedded SPA from `dist/`, registers all API route groups, starts the pcap queue + watcher, launches the flag detector, and UDP-broadcasts its presence (`service/udpcast`) so clients can auto-discover it.
- **Client mode** (`config.Client_mode`): runs goroutines that heartbeat to the server, poll for exploit tasks, download & execute them, and report output/flags/traffic back. Optionally also runs a standalone local proxy when not in server mode.
- The default generated config enables **both** server and client, so a single instance acts as controller + executor. They are independent flags, not mutually exclusive.

### Two HTTP ports, two engines, with auth (`0e7.go` `main`, `service/mw`)
The server runs **two separate Gin engines on two ports**, each with its own auth — they are NOT one engine:
- **C/S engine (`rCS`, `server.port` 6102)** — the client↔server protocol. Routes: `/api/heartbeat`, `/api/exploit`, `/api/exploit_download`, `/api/exploit_output`, `/api/flag`, `/api/monitor` (`service/route/register.go`), `/api/update` (`service/update`), and the cross-over `/api/pcap_upload` (`service/webui/register_cs.go` re-registers the webui pcap handler under `/api/` so clients can upload pcaps). Protected by `mw.CSWhitelist()` (IP allow-list from `cs_whitelist`) **then** `mw.CSToken()` (`X-CS-Token` header vs shared `cs_token`). No etag/gzip here (they would buffer the large `/api/exploit_download` / `/api/update` bodies into memory).
- **Admin engine (`rAdmin`, `server.admin_port` 6103)** — the Vue SPA + management API. Routes: `/webui/*` (`service/webui/register.go`, incl. `/webui/log/ws` WebSocket), `/git/*` (git smart HTTP), `/proxy/*`, `/static/*`, `/`, plus `/api/admin/{login,logout,status}`. Protected by `mw.AdminAuth()` — cookie session first (login at `/api/admin/login` with `admin_password`), falling back to HTTP Basic for `/git/*` (git CLI). Has etag + gzip.
- **Both engines call `SetTrustedProxies(nil)`** — mandatory; otherwise `c.ClientIP()` trusts `X-Forwarded-For` and the IP allow-list is bypassable by header forgery.
- The client injects `X-CS-Token` on every request via `utils.NewCSRequest` (replaces `http.NewRequest` at the call sites in `service/client/*` and `service/update/replace.go`).
- UDP broadcast (`service/udpcast`) still advertises the C/S port (6102); `config.Server_url` that clients use points at the C/S port.

### Configuration (`service/config/config.go`)
INI file (`gopkg.in/ini.v1`) with sections `[global]`, `[client]`, `[server]`, `[search]`. Loaded by `config.Init_conf`. All settings are **package-level global variables** in the `config` package (e.g. `config.Server_port`, `config.Server_mode`, `config.Db`, `config.Global_debug`). Edit a setting → read/write the global; there is no config struct passed around. `--server` with a missing config writes a default template via `ensureServerConfig`.

Auth-related fields (read in `Init_conf`, server section; `cs_token` is also mirrored into `[client]`):
- `server.admin_port` (default `6103`), `server.admin_password`, `server.cs_token`, `server.cs_whitelist` (comma-separated IP/CIDR; empty = no IP restriction, token-only).
- If `cs_token` / `admin_password` are empty at server start, `config.ensureSecrets` (`service/config/secure.go`) generates random values, writes them back to `config.ini` (mirroring `cs_token` into `[client]`), and logs them. The token is effectively mandatory — every client must send the same `cs_token`.

### Storage (`service/database`)
GORM, switchable between SQLite (default, file `sqlite.db` with aggressive WAL tuning) and MySQL. Every model is in `service/database/const.go` and all tables are prefixed `0e7_` (Client, Exploit, Flag, ExploitOutput, Action, PcapFile, Monitor, Pcap). Access the DB everywhere through the global `config.Db`.

### Full-text search (`service/search`)
Pluggable engine selected by `[search] search_engine`: **bleve** (embedded, default) or **elasticsearch**. Used to index and query pcap traffic content (`client_content`/`server_content` columns).

### Pcap / traffic analysis (`service/pcap`)
Uses `gopacket` + libpcap (the reason CGO is mandatory). Parses HTTP/H2/WebSocket/TCP/UDP, reassembles flows, and scans for flags via the regex in `[server] flag` (the `service/flag` detector). The server watches a local `pcap/` directory and ingests dropped capture files.

### Actions (`service/server`, `service/webui/action.go`)
Server-side Go scripts stored as text and executed at runtime by **yaegi** (the Go interpreter), driven by a scheduler (`StartActionScheduler`) using `next_run`/`interval` columns. Distinct from "exploits" (which are client-executed external scripts in Python/Go/etc.).

### Desktop wrappers (`wails/`, `electron/`)
Both shells embed the regular backend binary (`wails/bin/backend.bin`, `electron/resources/bin/...`) and at runtime launch it as a subprocess on a random port, then point their webview at `http://127.0.0.1:<port>`. The `wails/main.go` uses `//go:embed bin/backend.bin` — same embed-before-build constraint as the main binary. The Wails GUI build uses the `wails` build tag (`app.go` has `//go:build wails`).

## Key Conventions
- DB access is via the global `config.Db`; config values are globals in package `config`. Do not introduce DI containers — follow the existing global-state pattern.
- **Two engines, not one**: C/S routes go on `rCS` (`/api/*`), admin routes on `rAdmin` (`/webui/*`, `/git/*`, `/proxy/*`, SPA). New client-facing endpoint → handler under `service/route` + register in `service/route/register.go`; new admin endpoint → `service/webui` + `service/webui/register.go`. A route needed on **both** ports (like `pcap_upload`) gets a re-register helper in `service/webui/register_cs.go`.
- **Auth middleware** lives in `service/mw` (`cs.go` = C/S token + IP whitelist, `admin.go` = admin cookie/Basic, `session.go` = go-cache sessions). New C/S routes are auto-covered by `mw.CSWhitelist()+mw.CSToken()` on `rCS`; new admin routes by `mw.AdminAuth()` on `rAdmin`.
- **Client→server requests** must use `utils.NewCSRequest` (not `http.NewRequest`) so the `X-CS-Token` header is injected.
- New persistent entities: add the GORM model to `service/database/const.go` with a `0e7_` table name and an index-heavy tag style matching existing models.
- `0e7.go` keeps a `registerCleanup`/`runCleanup` registry; resource owners register teardown functions that run in reverse order on exit. The two engines use `gin.Engine.Run/RunTLS` (which self-build the `http.Server`), so shutdown is `os.Exit` + `runCleanup`, not a graceful `http.Server.Shutdown(ctx)`.
- Keep new user-facing strings/logs in Chinese to match the codebase.
