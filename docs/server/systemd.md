# systemd service

The installer can do this for you (`--service`). This is what it sets up, so you can do it by hand or adjust it.

## The unit

`/etc/systemd/system/dockduck.service`:

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

Replace `deploy` with the user that owns your compose projects. That user must be in the `docker` group:

```sh
sudo usermod -aG docker deploy
```

::: warning Run as a normal user
DockDuck reads and writes your compose files, so run it as the user who owns them. Running it as root means every file it saves becomes root-owned.
:::

Then:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now dockduck
sudo systemctl status dockduck
journalctl -u dockduck -f        # logs
```

The config (roots, settings, the password hash) lives in that user's `~/.config/dockduck/config.json`. Set the password as the **same user**: `sudo -u deploy dockduck passwd`.

## Updating

- **From the page:** Settings → Updates. DockDuck replaces its binary and exits; systemd starts the new one (see [Restart behaviour](#restart-behaviour)).
- **From the shell:** run the install command again, or `sudo dockduck update`, then `sudo systemctl restart dockduck`.

::: tip In-app updates need write access
Updating from Settings replaces `/usr/local/bin/dockduck` as the service user, who cannot write there by default. Either update from the shell with `sudo`, or give the user the file: `sudo chown deploy /usr/local/bin/dockduck`.
:::

## Restart behaviour

`Restart=always` matters: when DockDuck updates itself under systemd it exits cleanly and relies on systemd to start the new version. With `Restart=on-failure` a clean exit is not restarted and DockDuck would stay stopped after an in-app update. `systemctl stop dockduck` still stops it for good.
