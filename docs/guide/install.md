# Install and update

DockDuck is a single file for Linux or macOS (Intel and Apple/ARM). It needs the `docker` command (with the Compose plugin) on the same machine, and a user that may use Docker.

## One command

```sh
curl -fsSL https://mnr73.github.io/dock-duck/install.sh | sh
```

The script:

1. detects your system and CPU,
2. downloads the latest release from GitHub,
3. **checks its SHA-256 against the release's `checksums.txt`** and stops if they differ,
4. installs `dockduck` into `/usr/local/bin` (using `sudo` if needed) or `~/.local/bin`.

Then start it:

```sh
dockduck
```

and open <http://127.0.0.1:9473>.

The same script is also served straight from the repository, which works even if this site is down:

```sh
curl -fsSL https://raw.githubusercontent.com/mnr73/dock-duck/main/install.sh | sh
```

::: tip Read it first
Piping a script into `sh` means trusting it. [install.sh](https://github.com/mnr73/dock-duck/blob/main/install.sh) is short; read it before you run it.
:::

### Options

Pass them after `sh -s --`:

```sh
curl -fsSL https://mnr73.github.io/dock-duck/install.sh | sh -s -- --version v0.1.0
```

| Option | Meaning |
|---|---|
| `--version vX.Y.Z` | Install that release instead of the latest |
| `--dir PATH` | Install into `PATH` |
| `--service` | Also install and start a systemd service (Linux). See [systemd](/server/systemd) |
| `--user NAME` | The user the service runs as (must be in the `docker` group) |

## Update

Any of these:

- **Run the install command again.** It installs only when a newer version exists.
- **`dockduck update`** (or `dockduck update --check` to only look).
- **Settings → Updates → Check for updates → Update.** DockDuck downloads the release, verifies it, replaces itself and restarts. Terminals that are open are closed by the restart.

DockDuck never contacts GitHub on its own. A check happens only when you ask.

If the binary sits in a folder your user cannot write (for example `/usr/local/bin` without root), updating from Settings explains that; run the install command with `sudo`, or `sudo dockduck update`.

## Manual download

Every [release](https://github.com/mnr73/dock-duck/releases) has `dockduck_<version>_<os>_<arch>.tar.gz` and `checksums.txt`.

```sh
sha256sum -c --ignore-missing checksums.txt
tar -xzf dockduck_*_linux_amd64.tar.gz dockduck
sudo install -m 755 dockduck /usr/local/bin/
```

## Uninstall

```sh
sudo systemctl disable --now dockduck 2>/dev/null; sudo rm -f /etc/systemd/system/dockduck.service
sudo rm -f /usr/local/bin/dockduck ~/.local/bin/dockduck
rm -rf ~/.config/dockduck      # roots, settings and the password hash
```
