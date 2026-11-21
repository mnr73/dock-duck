# Command line and configuration

```
dockduck [--addr HOST:PORT] [--config FILE] [--dev]
dockduck passwd [--config FILE]
dockduck update [--check]
dockduck version
```

| Flag / command | Meaning |
|---|---|
| `--addr` | default `127.0.0.1:9473`. Non-loopback needs a password |
| `--config` | default `~/.config/dockduck/config.json` |
| `--dev` | debug logs, for developing DockDuck |
| `passwd` | sets or resets the password; reads the terminal, or the first stdin line when piped. Does not ask for the old one. A running server ends all sessions within a second |
| `update` | installs the latest release over this binary (checksum verified); `--check` only reports |
| `version` / `--version` | prints the version |

Exit codes: 0 normal, 1 error, 2 bad arguments.

Environment: `DOCKDUCK_TERMINAL` (e.g. `konsole -e`), `DOCKDUCK_UPDATE_API` (GitHub-compatible API base), `DOCKER_HOST`, `DOCKER_CONFIG`.

## config.json

```json
{
  "roots": [{ "id": "a1b2c3d4e5", "path": "/home/me/projects" }],
  "settings": { "default_context": "", "sync_cli": false, "scan_depth": 4 },
  "selected_context": "default",
  "auth": { "password_hash": "$argon2id$..." }
}
```

`scan_depth` is 1 to 8. `sync_cli` makes a context switch also run `docker context use` (off by default). `password_hash` is a hash, never the password. The file is written 0600 and edits are picked up within a second.

## systemd unit (what `--service` writes)

```ini
[Unit]
Description=DockDuck - web GUI for Docker Compose projects
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=deploy
ExecStart=/usr/local/bin/dockduck --addr 127.0.0.1:9473
Restart=always
RestartSec=2
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
```

The user must be in the `docker` group and must own the compose projects (files DockDuck saves keep that owner). Set the password as the same user: `sudo -u deploy dockduck passwd`. For in-app updates the user also needs to own `/usr/local/bin/dockduck` (or update from a shell with sudo).
