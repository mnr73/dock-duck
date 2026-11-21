# Compose and .env

## Compose file

**Compose** opens the project's file in an editor with syntax highlighting. **Validate** runs `docker compose config` on your unsaved text and shows the errors and warnings (such as an unset `${VAR}`) with line numbers, without touching the file. `Ctrl+S` saves.

- Saving writes atomically and keeps one `.bak` copy of the previous version.
- If the file changed on disk after you opened it, saving is refused and you choose whether to reload.
- If the project is running, a hint offers to apply the change with `up`.

## .env

A table of variables, or **Raw** text.

- **Secrets are hidden by default.** Names that look like secrets (PASSWORD, SECRET, TOKEN, KEY…) show as dots, and the value is not even sent to your browser until you press **Reveal secrets**.
- A hidden value can never be saved over the real one: reveal first.
- Keys that your `.env.example` has and `.env` lacks are offered with *Add them*.
- A new `.env` is created readable by you only (mode 0600).

Only the project's own compose and `.env` files can be opened. There is no way to browse other files.
