# 04 — Phases

Each phase ends with something runnable. Tick boxes as you go.

## Phase 0 — Scaffold (½ day)
- [ ] Install Go ≥ 1.23 (not installed yet), pnpm present (12.x)
- [ ] `apps/server`: `go mod init`, `cmd/dockduck/main.go`, `/api/v1/health`, flags (`--addr`, `--config`, `--dev`)
- [ ] `apps/web`: Vite + Vue 3 + TS + Tailwind v4 + Pinia + Router + **@vueuse/core** + unplugin-icons with `@iconify-json/lucide`; ESLint/Prettier/sort-imports config taken from shiva-connect (not erp-front)
- [ ] `src/icons.ts`: single module re-exporting every icon with a semantic name + icons tree-shaking vite plugin
- [ ] `go:embed` of `dist`, SPA fallback, dev proxy (Vite → :8080)
- [ ] Makefile works: `make build` yields one binary serving the SPA
- [ ] CI: lint + test + build

## Phase 1 — Docker core: contexts + discovery (1–2 d)
- [ ] `dockercli` wrapper (timeouts, `--context`, JSON output, fake for tests)
- [ ] Contexts: list, inspect, ping/availability, switch (app-level selection), default from settings; optional opt-in sync via `docker context use`; no create/remove
- [ ] Config file + roots CRUD (`GET/POST/DELETE /roots`), rescan
- [ ] Discovery walk + merge with `compose ls -a`; project model (name, dir, files, env file)
- [ ] Tests with fake docker + temp dirs
- **Done when:** `curl /api/v1/projects` lists real projects from `~/prj`.

## Phase 2 — Project state + actions (2 d)
- [ ] Project/service state from `ps -a` (status, health, image, ports, container names)
- [ ] Actions: up, down (opt. `-v`, `--remove-orphans`), start, stop, restart, build, pull — project and per service
- [ ] Jobs with streamed output; `events` watcher pushing updates
- [ ] Resource usage (`docker stats --no-stream`, throttled)
- **Done when:** can up/down a real project from curl + watch events.

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
