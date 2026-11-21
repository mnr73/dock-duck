# Command line

```
dockduck [--addr HOST:PORT] [--config FILE] [--dev]
dockduck passwd [--config FILE]
dockduck update [--check]
dockduck version
dockduck --version
```

## `dockduck`

Starts the server.

| Flag | Default | Meaning |
|---|---|---|
| `--addr` | `127.0.0.1:9473` | Address to listen on. Anything other than loopback needs a password |
| `--config` | `~/.config/dockduck/config.json` | Settings file |
| `--dev` | off | Debug logging (for developing DockDuck itself) |
| `--version` | | Print the version and exit |

Stop it with Ctrl+C or `SIGTERM`. It closes open terminals and stops what they left running in containers before exiting.

## `dockduck passwd`

Sets or resets the login password. It reads from the terminal, or the first line of standard input when piped. It does not ask for the old password: whoever can run it on the machine already owns it. A running server notices the change within a second and ends every session.

## `dockduck update`

Installs the latest release over this binary (checksum verified). `--check` only reports. A running server keeps its old code until restarted; the Settings page restarts it for you.

## Environment variables

| Variable | Meaning |
|---|---|
| `DOCKDUCK_TERMINAL` | Command for "open in a native terminal", e.g. `konsole -e` |
| `DOCKER_HOST`, `DOCKER_CONFIG` | Passed to `docker` as usual |
| `DOCKDUCK_UPDATE_API` | Use another GitHub-compatible API for updates (a mirror). Only the person who starts DockDuck can set it |

## Exit codes

`0` normal stop, `1` error (for example *refusing to listen without a password*), `2` bad arguments.
