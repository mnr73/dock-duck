# 01 — Overview

## Goal
Manage the Compose projects that already exist on a machine from a clean web UI, without redefining them in the tool (the Dockge limitation) and without creating Docker contexts (the Docker Desktop limitation). Runs as a host process, so it works on a laptop and on a server.

## Features (from the requirements)
1. Show the current Docker context name (+ endpoint, availability)
2. List and switch contexts (switch = app-level selection, per-command `--context`; creating/removing contexts is out of scope and forbidden)
3. Add/remove root directories; scan them for Compose projects; manual rescan
4. Show services of a project (status, image, ports, health, usage)
5. Edit Compose file(s) (multi-file projects supported)
6. View/edit `.env` (table + raw, masked secrets, handle missing file)
7. Start / stop / up / down / restart — per project and per service; confirm destructive actions (`down -v`)
8. Logs: live stream, service filter, search, pause, clear, autoscroll, timestamps, copy, download
9. Exec into a container with user choice: image default / `0` / `1000` / custom uid or name; optional command, otherwise interactive shell (PTY)
10. Login/auth: (a) app login to the DockDuck UI, (b) Docker registry login/logout (Docker Hub + private)
11. Global "Ports" view: every published port of every container, with owner project/service, conflict hints
12. Settings: theme (system/light/dark), scan config, terminal defaults, default context, behaviour

## Non-goals (v1)
- Swarm/Kubernetes, image/volume/network management UI (maybe later), multi-host agent, Compose YAML visual builder, parsing YAML ourselves.

## Key decisions
| Decision | Choice | Why |
|---|---|---|
| Backend | Go | Official Docker ecosystem, single static binary, easy cross-compile, good PTY/WS libs |
| Delivery | One binary, UI embedded | Same install on laptop/server; no Docker-in-Docker path-mount issues |
| Docker access | `docker` CLI (+ events via `docker events`) | Identical semantics to what the user runs by hand; Compose parsing stays in Compose |
| Contexts | List + switch; `--context` per call | Switching never mutates the user's Docker config (optional opt-in "sync with Docker CLI" runs `docker context use`); never create/remove |
| Frontend | Vue 3 + Vite + TS + Tailwind v4 + VueUse | User stack; conventions from newer projects (shiva-connect, zhoon_front), not erp-front |
| Icons | `@iconify-json/lucide` via unplugin-icons, all declared in one `src/icons.ts` module | Consistent icon-per-concept, tree-shaken |
| Repo | Monorepo, pnpm workspace + Go module | One place for everything |
| Name | App: DockDuck, repo: dock-duck | — |
