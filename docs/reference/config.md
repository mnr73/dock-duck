# Configuration

DockDuck keeps its state in one JSON file: `~/.config/dockduck/config.json` (override with `--config`). It is written `0600`. You normally never edit it; the UI and `dockduck passwd` do.

```json
{
  "roots": [{ "id": "a1b2c3d4e5", "path": "/home/me/projects" }],
  "settings": {
    "default_context": "",
    "sync_cli": false,
    "scan_depth": 4
  },
  "selected_context": "default",
  "auth": { "password_hash": "$argon2id$..." }
}
```

| Key | Meaning |
|---|---|
| `roots` | Folders scanned for compose projects |
| `settings.default_context` | Context used on launch when none is remembered (`""` = last used) |
| `settings.sync_cli` | Also run `docker context use` when you switch context |
| `settings.scan_depth` | Folder levels below a root, 1 to 8 |
| `selected_context` | The context chosen last |
| `auth.password_hash` | argon2id hash. Never the password |

Changes made to the file while DockDuck runs are picked up within a second.

Per-browser preferences (theme, accent, density, terminal defaults, toggles) are not here; they live in the browser's local storage.
