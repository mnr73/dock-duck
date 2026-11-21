# Settings and shortcuts

Changes save as you make them.

| Group | Setting | Stored |
|---|---|---|
| Appearance | Theme, accent colour (the logo follows it), density | this browser |
| Project discovery | Scan depth (1 to 8; changing it rescans), roots | on the server |
| Docker | Context on launch, *also switch the docker CLI*, Compose version | on the server |
| Terminal | Default shell, user, font size | this browser |
| Behavior | Confirm before compose down, toasts for finished operations | this browser |
| Login | Set or change the password, sign out | on the server |
| Updates | Check and install a new version | |

*Also switch the docker CLI* is off by default: DockDuck uses its own selected context and does not change what `docker context show` says in your terminal unless you turn it on.

## Keyboard

| Keys | Action |
|---|---|
| `Ctrl/Cmd + K` | Command palette: pages, projects, the open project's tabs and safe actions |
| `?` | Show the shortcut list |
| `[` | Collapse or expand the sidebar |
| `g` then `p` / `o` / `r` / `s` | Projects / Ports / Registries / Settings |
| `Ctrl/Cmd + F` | Search in the log |
| `Ctrl/Cmd + S` | Save the compose file or .env |

Single keys are ignored while you type in a field, the editor or a terminal. `down` is deliberately not in the palette.
