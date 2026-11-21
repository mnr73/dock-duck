# 04 — Phases

Each phase ends with something runnable. Tick boxes as you go.

## Phase 0 — Scaffold (½ day)
- [x] Install Go ≥ 1.23 (installed to ~/.local/go from the aliyun mirror: go.dev/dl and dl.google.com are blocked here; GOPROXY=goproxy.cn), pnpm present (12.x)
- [x] `apps/server`: `go mod init`, `cmd/dockduck/main.go`, `/api/v1/health`, flags (`--addr`, `--config`, `--dev`)
- [x] `apps/web`: Vite + Vue 3 + TS + Tailwind v4 + Pinia + Router + **@vueuse/core** + unplugin-icons with `@iconify-json/lucide`; ESLint/Prettier/sort-imports config taken from shiva-connect (not erp-front)
- [x] `src/icons.ts`: single module re-exporting every icon with a semantic name + icons tree-shaking vite plugin
- [x] `go:embed` of `dist`, SPA fallback, dev proxy (Vite → :8080)
- [x] Makefile works: `make build` yields one binary serving the SPA
- [x] CI: lint + test + build

Notes: TypeScript pinned to ^6 (typescript-eslint/vue-tsc do not support TS 7 yet). Go module name is `dockduck` (local, not published). Pinia 4, vue-router 5, VueUse 15, Vite 8, Tailwind 4.

## Phase 1 — Docker core: contexts + discovery (1–2 d)
- [x] `dockercli` wrapper (timeouts, `--context`, JSON output, fake for tests)
- [x] Contexts: list, inspect, ping/availability, switch (app-level selection), default from settings; optional opt-in sync via `docker context use`; no create/remove
- [x] Config file + roots CRUD (`GET/POST/DELETE /roots`), rescan
- [x] Discovery walk + merge with `compose ls -a`; project model (name, dir, files, env file)
- [x] Tests with fake docker + temp dirs
- **Done when:** `curl /api/v1/projects` lists real projects from `~/prj`.

Notes (as built):
- `dockercli.Client.Output` refuses `docker context create/rm/update/import/use`; only `UseContext` can run `use` (opt-in `sync_cli`). Names/IDs validated by `dockercli.ValidName`.
- Selected context resolution: saved selection → `default_context` setting → Docker CLI current → first.
- Project state in this phase is the coarse `compose ls -a` status (running/partial/stopped + counts); per-service state comes in Phase 2. Projects known only to Docker (outside roots) are listed with `root` empty.
- Discovery skips hidden dirs, `node_modules`, `vendor`, `dist`, `build`, `target`, venvs; stops descending at a project; roots must be absolute existing dirs (symlinks resolved).
- Verified on the real machine: 27 projects found under ~/prj, erp-back 4/4 running.
- Endpoints live now: `GET /contexts`, `PUT /contexts/selected`, `GET|POST /roots`, `DELETE /roots/{id}`, `POST /roots/rescan`, `GET /projects`.

## Phase 2 — Project state + actions (2 d)
- [x] Project/service state from `ps -a` (status, health, image, ports, container names)
- [x] Actions: up, down (opt. `-v`, `--remove-orphans`), start, stop, restart, build, pull — project and per service
- [x] Jobs with streamed output; `events` watcher pushing updates
- [x] Resource usage (`docker stats --no-stream`, throttled)
- **Done when:** can up/down a real project from curl + watch events.

Notes (as built):
- Service status folds container states: restarting > unhealthy > starting > running > paused > error (exited non-zero) > exited > stopped. Declared-but-never-created services show `stopped`; containers of removed services still appear.
- State = `compose config --format json` (declared) merged with `compose ps -a --format json` (existing). If `config` fails (e.g. missing env var), `config_error` is returned and services come from containers only.
- Commands pass `--project-directory`, every `-f` file, and `-p <name>` only when Docker already knows the project (so `name:`/COMPOSE_PROJECT_NAME keep working otherwise).
- One running job per project (409 `busy`); other projects are independent. Job output is kept in memory (5000 lines, last 50 jobs).
- `down -v` is only built when `volumes:true` is sent explicitly; the UI must confirm it (Phase 4). `down` rejects a service list; service names are validated.
- Events: one `docker events` process per context while ≥1 WebSocket listens; killed with the last listener; auto-restart with backoff.
- Stats: `docker stats --no-stream`, cached 3 s per project, running containers only.
- WebSockets (coder/websocket) accept only same-origin (plus the Vite dev origin from loopback); a foreign Origin gets 403 (tested).
- E2E on real Docker with a throwaway project (`dockduck-test-a`): up/stop one service/start/down all succeeded, healthcheck state, exit-code-3 service shown as `error`, published port detected, IPv6 duplicate port collapsed, events stream saw create/start/die/health_status/stop/destroy, job WS streamed output then `done`.

## Phase 3 — UI shell + design system (2 d)
- [ ] Tokens (dark/light) from mockup; fonts (Geist / mono); theme via VueUse `useColorMode`; generic components (ideas from shiva-connect): Btn, IconBtn, Menu, Dialog/Confirm, Toast, Tabs, Table, Tooltip, Badge/StatusDot, Select, Input
- [ ] Layout: collapsible sidebar, status bar, project header (boards 11, 12, 13)
- [ ] Context selector popover (board 06), project list with status + expandable services
- [ ] Roots dialog (board 07), rescan, empty/loading/disconnected states
- **Done when:** UI lists contexts and projects live.

## Phase 4 — Services, ports, actions UI (1–2 d)
- [ ] Services table (board 01) with row context menu, status indicators, health, usage
- [ ] Project actions with confirmations (down, down -v), toasts, job output drawer
- [ ] Global **Ports** view: all published ports, filter, conflict detection, open link
- [ ] Inspect drawer for a service/container

## Phase 5 — Logs (1–2 d)
- [ ] WS streaming, service filter, timestamps, search/highlight, pause/resume, clear, autoscroll, copy, download
- [ ] Virtualised rendering, ANSI colours, bounded buffer
- [ ] Always-dark surface (board 04)

## Phase 6 — Editors: compose + env (2–3 d)
- [ ] CodeMirror 6: YAML highlight, line numbers, search, indent, dirty indicator, save/discard (board 02)
- [ ] Multi-file tabs; server-side validation via `compose config -q`; ETag conflict handling; `.bak`
- [ ] Env editor: table + raw toggle, masked secrets with per-row reveal, add/remove, missing-file state (board 03)
- [ ] "Apply changes" hint (recreate with `up -d`)

## Phase 7 — Exec terminal (1–2 d)
- [ ] PTY over WS, xterm.js, resize, reconnect handling
- [ ] Target service/container, user: default / 0 / 1000 / custom; shell autodetect (`bash`→`sh`); optional command (board 05)
- [ ] Multiple terminal tabs; default prefs from settings

## Phase 8 — Auth (2 d)
- [ ] App auth: first-run setup (password), argon2id, session cookie, CSRF, rate-limit, logout; refuse non-loopback bind without password
- [ ] Registries (board 08): status from `~/.docker/config.json` (names only), `docker login --password-stdin`, `logout`, Docker Hub preset, error feedback

## Phase 9 — Settings + polish (1–2 d)
- [ ] Settings (board 09): theme, scan config, terminal defaults, default context, behaviour
- [ ] Keyboard shortcuts / command palette, a11y pass, error & disconnected states, i18n-ready strings

## Phase 10 — Packaging & release (1 d)
- [ ] Cross-compiled binaries (linux amd64/arm64, darwin), `--version`
- [ ] systemd unit, install script, reverse-proxy example (TLS), README usage
- [ ] Integration tests against a throwaway compose project

## Later / ideas
Multi-host agent mode, image/volume/network pages, compose templates, project groups/tags, notifications on unhealthy, Podman support.
