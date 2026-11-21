---
name: dock-duck
description: Use when installing, running, updating or troubleshooting DockDuck, the web GUI for Docker Compose projects.
version: 0.1.0
license: MIT
---

# DockDuck

A web GUI for the Docker Compose projects a person already has: up, down, start, stop, restart, pull, build, live logs, a PTY terminal in any container, compose and `.env` editing, registry login, and every published port. **One Go binary with the Vue UI inside**, listening on `127.0.0.1:9473`. It works on the user's own folders and compose files and keeps no copy. Docker contexts are listed and switched, never created or removed.

Docs: https://mnr73.github.io/dock-duck/ . Source: https://github.com/mnr73/dock-duck . Details for each task are in `references/`.

## When to use

- The user wants to install, run, update or remove DockDuck, or put it on a server.
- Something in DockDuck fails (blank page, "cross-origin", terminal stuck, project missing).
- The user asks how to script it (HTTP API) or where its state lives.

Do not use it to run a single `docker compose` command for the user; just run that command.

## Install and update

```sh
curl -fsSL https://mnr73.github.io/dock-duck/install.sh | sh        # install, and update by running it again
curl -fsSL https://mnr73.github.io/dock-duck/install.sh | sh -s -- --service   # + systemd unit
dockduck                  # http://127.0.0.1:9473
dockduck update           # or: Settings -> Updates
```

Linux and macOS, amd64 and arm64 only. The installer verifies the download's SHA-256 against the release's `checksums.txt` and refuses on mismatch. Updating never happens on its own: only on `dockduck update`, the installer, or the Settings button.

## Rules that surprise people

- **Default bind is loopback.** Any other `--addr` (`0.0.0.0`, `:9473`, a LAN IP) **refuses to start without a password**: run `dockduck passwd` first (as the user that runs the service).
- **No password + loopback = open.** Anyone on that machine can use it; the UI offers "Set password". First-time setup from a browser is only allowed from the machine itself.
- **No built-in TLS.** Put nginx or Caddy in front (`deploy/` has both). The proxy must forward `Host`, set `X-Forwarded-Proto: https` and pass WebSockets, or login works but the terminal hangs and actions say "cross-origin".
- **With a password any Host name is accepted; without one only `localhost` and IPs** (DNS-rebinding guard).
- Under systemd use `Restart=always`: an in-app update exits cleanly and relies on the manager to start the new binary.
- Docker access is root access: tell the user to use a long password and TLS before exposing it.
- `down -v` needs the project name typed. `down` is deliberately not in the command palette.
- A dev build (`dev` or a `-dirty` version) cannot compare or update itself.

## Where things are

| What | Where |
|---|---|
| Settings, roots, password hash | `~/.config/dockduck/config.json` (0600; `--config` overrides) |
| Per-browser prefs (theme, accent, terminal defaults) | the browser's local storage |
| Docker login | `~/.docker/config.json` (DockDuck calls `docker login`) |
| Logs under systemd | `journalctl -u dockduck -f` |

## Commands

`dockduck [--addr HOST:PORT] [--config FILE]`, `dockduck passwd`, `dockduck update [--check]`, `dockduck version`. Env: `DOCKDUCK_TERMINAL` (native terminal command), `DOCKDUCK_UPDATE_API` (mirror, set by whoever starts it). See `references/cli-and-config.md`.

## Troubleshooting

Match the message, then read `references/troubleshooting.md`:

| Message | Cause in one line |
|---|---|
| "No password is set on this server, so it cannot be used from the network yet" | opened remotely with no password: run `dockduck passwd` on the server |
| "refusing to listen on ... without a password" | non-loopback `--addr` before a password exists |
| "this host name is not allowed while no password is set" | open server reached by a domain: set a password, or use localhost/IP |
| "cross-origin request refused" | proxy rewrites `Host` |
| terminal / logs "connecting" forever | proxy does not forward WebSockets |
| "Can't reach the Docker daemon" | selected context is down, or the user is not in the `docker` group |
| Settings -> Updates "not writable" | binary lives where the user cannot write: `sudo dockduck update` |

## Scripting

The UI uses a JSON API under `/api/v1` (WebSockets under `/ws/v1`). With a password: log in, keep the cookie, send `X-CSRF-Token` (from `GET /api/v1/auth/status`) on every non-GET. Without one on loopback it is open. Full contract: `references/http-api.md`.
