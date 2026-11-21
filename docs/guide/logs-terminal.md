# Logs and terminal

## Logs

Live logs for the whole project or chosen services, with search (`Ctrl+F`), level filters, "since" ranges, pause, copy selected lines and **Download**. If the connection drops you see *Disconnected* and a **Reconnect** button.

## Terminal

Open a shell inside any running service: **Terminal** tab, or *Execute command…* in a service's row menu.

- **Run as**: the container's own user, `root` (UID 0), UID 1000, or any name or UID you type.
- **Shell**: bash, falling back to sh; or a path you choose.
- **A real terminal**: arrow keys, history, `vi`, Ctrl+C and resizing work. It is not a command box.
- **Sessions survive**: close the tab or reload and the shell is still there with its scrollback; reopen the terminal and you are back in it. Only one window controls a session at a time; opening it in another window takes it over.
- **Cleanup**: closing a session also stops what it started in the container.

### Open in your own terminal

*Open in konsole* (or your system's terminal) runs the same `docker exec` in a native window. It works when DockDuck runs on your desktop; set `DOCKDUCK_TERMINAL` to choose a terminal program.

::: warning
A shell as root inside a container is a powerful thing. That is why logging in matters when DockDuck is reachable from a network.
:::
