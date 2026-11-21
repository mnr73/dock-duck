# 02 — Architecture

```
Browser (Vue SPA) ──HTTP /api/v1──────────┐
        │  └─WS /ws/v1 (logs, events, exec)│
        ▼                                  ▼
   dockduck (Go binary, embeds SPA)
     ├─ auth      sessions, argon2id, CSRF
     ├─ config    ~/.config/dockduck/config.toml (roots, settings, users)
     ├─ discovery scan roots → projects (+ compose ls -a, container labels)
     ├─ dockercli wrapper around `docker` (context-aware, fakeable)
     ├─ ops       up/down/start/stop/restart/build jobs (streamed output)
     ├─ logs      `compose logs -f --timestamps` → WS fan-out
     ├─ exec      `docker exec -it -u` under creack/pty ⇄ WS ⇄ xterm.js
     ├─ files     safe read/write of compose/.env with backup + validation
     └─ events    `docker events` → SSE/WS push to UI
            │
            ▼  docker --context <ctx> ...
        Docker daemon (local socket / ssh / tcp, per context)
```

## Context handling
- `docker context ls --format json` lists contexts; `docker context inspect` gives endpoint. We never `create/rm/update/import`.
- Switching = changing the selected context in app config (per user/session, default from settings). Every command is `docker --context <name> ...`, so the user's Docker CLI default is untouched.
- Optional setting "sync with Docker CLI" (off by default): on switch also runs `docker context use <name>`. This is the only place `use` is ever called.
- Availability check: `docker --context X info --format '{{json .ServerVersion}}'` with a 3 s timeout; the result is cached and shown (active / unavailable + error).
- Compose files live on the *host running DockDuck*. For remote contexts (ssh/tcp) `compose up` still reads local files and talks to the remote engine; bind mounts then refer to the remote filesystem — surface a warning in the UI.

## Project discovery
1. Walk each root (depth limit, default 4; skip `node_modules`, `.git`, `vendor`, `dist`, hidden dirs by default).
2. A folder is a project if it contains `compose.yaml|compose.yml|docker-compose.yaml|docker-compose.yml` (+ `*.override.*`).
3. Name = `COMPOSE_PROJECT_NAME`/`name:` from `docker compose config`, else directory name.
4. Merge with `docker compose ls -a --format json` so projects started elsewhere also appear (matched by `working_dir` label).
5. Runtime state = `docker compose -p <name> ps -a --format json` per project (batched with one `docker ps -a --filter label=com.docker.compose.project --format json`).

## Realtime
- `docker events --filter type=container --format json` kept as one long-lived process per context; fan-out to browser clients; UI invalidates the affected project.
- Logs: one `compose logs -f --tail N --timestamps [svc]` process per viewer; killed on WS close. Pause is client-side buffering.
- Exec: WS binary frames = PTY bytes, text frames = control JSON (`resize`). PTY process killed on close.
- Long operations (`up`, `build`) are *jobs*: `POST` returns a job id; output streams on `/ws/v1/jobs/{id}`; final status stored in memory for the toast/history.

## File editing
- Only files belonging to discovered projects (compose files from `docker compose config --format json` metadata / the discovery walk, plus the project `.env`).
- Save = write temp + fsync + rename, keep one `.bak` copy, preserve mode. ETag (mtime+hash) to detect external changes → 409 on conflict.
- Validation = `docker compose -f <tmp> config -q` on the proposed content; errors returned with line info when available.
- `.env` parsing for the table view is client-side, line-preserving (comments/blank lines kept when saving from the table).

## Persistence
Small TOML/JSON config file only; no database. Registry credentials are delegated to Docker's credential store (`docker login`), never stored by DockDuck.
