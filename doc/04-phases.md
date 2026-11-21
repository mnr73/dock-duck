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
- [x] Tokens (dark/light) from mockup; fonts (Geist / mono); theme via VueUse `useColorMode`; generic components (ideas from shiva-connect): Btn, IconBtn, Menu, Dialog, Toast (vue-sonner), Tabs, Tooltip, Badge/StatusDot, Alert, EmptyState, SegmentedControl, Popover
- [x] Layout: collapsible sidebar, status bar, project header (boards 11, 12, 13)
- [x] Context selector page (board 06) + status-bar chip, project list with status + expandable services (project → service → container)
- [x] Roots page (board 07) with add/remove(confirm)/rescan, empty/loading/disconnected states
- **Done when:** UI lists contexts and projects live.

Notes (as built):
- Dark tokens are extracted from the mockup (`#0A0A0A` app, `#0D0D0D` sidebar, `#1F1F1F` lines, accent `#F2B94B`, green/amber/red/blue status colours); the mockup is dark-only, so the **light palette is derived** (`styles/tokens.css`). Tailwind names (`bg-app`, `text-muted`, `border-line`, `text-ok-fg`…) map to the tokens in `main.css`; components never hard-code colours. Log/terminal surfaces use `--dd-term-*` (always dark).
- Fonts: Geist + Geist Mono via `@fontsource-variable` (bundled, works offline).
- Deviation from the mockup: the **"+ New context" button is intentionally absent** (contexts are list/switch only). Contexts and Roots are pages reached from the rail, not popovers; the status-bar chip links to Contexts.
- Added beyond the board list: `/ports` and `/registries` rail entries (placeholder pages until Phases 4/8), `Outside your roots` group in the tree for projects Docker knows but no root contains, path hint for duplicate project names (two `megabin`s).
- Backend additions: `/health` returns the user's `home` (UI shortens `~`), and the WS/HTTP context can be chosen with `?context=` (browsers can't set headers on WebSockets).
- Live updates: `useLiveEvents` (VueUse `useWebSocket`, infinite reconnect) refreshes the project list and loaded details on container events, debounced 400 ms; reopened on context switch.
- State: Pinia stores `system`, `contexts`, `projects`, `ui` (sidebar/tree expansion persisted with `useStorage`). Theme: `useColorMode` (System/Light/Dark) in Settings.
- Header actions Up/Start/Stop/Restart already call the Phase 2 API (toast on start); **Down… is disabled** until the confirmation dialog lands in Phase 4. Overview and Services tabs show the same basic table for now; Compose/Env/Logs/Terminal are placeholders.
- Verified in a real browser against the embedded binary: dark + light themes, tree expansion (also restored after reload), context list/selected state, roots add (relative path → server error shown)/remove dialog (Cancel, ✕, Esc, backdrop all close), project page with live data from erp-back.
- Docker-unreachable state verified with `DOCKER_HOST` pointing at a dead socket: red banner ("Can’t reach the Docker daemon…" + Switch context / Retry), status-bar chip turns red, projects still listed; the daemon error text is shortened server-side (`friendlyError`).
- Known polish left for later: services table columns/typography, per-row service actions, Down confirmation lands in Phase 4.

## Phase 4 — Services, ports, actions UI (1–2 d)
- [x] Services table (board 01) with row context menu, status indicators, health, usage
- [x] Project actions with confirmations (down, down -v), toasts, job output drawer
- [x] Global **Ports** view: all published ports, filter, conflict detection, open link
- [x] Inspect drawer for a service/container

Notes (as built):
- Backend: `GET /ports` (all published ports of running containers; IPv4+IPv6 merged into `host_ips`, unpublished entries dropped, sorted by host port), `GET /projects/{id}/services/{service}` (trimmed `docker inspect`; **environment variables are deliberately never returned**), `GET /projects/{id}/down-preview` (containers, project-labelled networks and volumes that `down`/`down -v` would remove), and per-container `mem_bytes` in stats so the UI can sum per service. `docker inspect` ids are validated as hex.
- Services table: state with health badge, image, ports as `host→container` links (open `http://localhost:<port>`, `https` for 443/8443), CPU/memory per service (polled every 5 s with VueUse `useIntervalFn`, only while something runs), spinner on rows with an action in flight, row menu (Start/Stop/Restart/View logs/Execute command/Inspect; entries disable by state). `?service=` highlights a row (sidebar links use it). Overview and Services tabs share the page; Logs/Execute menu entries navigate to the Phase 5/7 routes with `?service=`.
- Down dialog: previews what is removed; "remove volumes" is only enabled when the project has named volumes, lists them, and requires typing the project name; separate orphan toggle. The header Down… button is enabled only when the project has containers and no job is running.
- Jobs: `stores/jobs` posts the action, follows `/ws/v1/jobs/{id}`, shows a bottom drawer with the live output (copy button, always-dark surface), toasts success ("<project> was taken down") or failure (last error-looking line + "Show output"), and reconnects/asks `/jobs/{id}` if the socket drops. A floating pill reopens the last output.
- Ports page: filter box, "reachable from the network only" toggle (persisted with `useStorage`), `duplicate` badge when one host port/protocol appears twice, project links, open-in-browser; refreshes on live container events and on context change.
- Verified end to end in a real browser on a throwaway project (`dockduck-test-a`): up with live CPU/mem, row Stop (only that container stopped, checked with `docker ps`), inspect drawer, Down dialog gates, **real `down -v` removed container+network+volume (checked in Docker)**, and a real failure (`Bind for 0.0.0.0:3306 failed: port is already allocated`) produced the error toast with the daemon message. The user's running erp-back containers were never touched.
- Pitfall hit twice: running `prettier --write` between scripted edits reflows lines so a later `str.replace` silently misses. Re-read the file or use the patch tool, then grep to confirm the change landed.
- Still open: log/exec targets of the row menu (Phases 5/7), `Pull`/`Build` buttons, Services tab is identical to Overview.

## Phase 5 — Logs (1–2 d)
- [x] WS streaming, service filter, timestamps, search/highlight, pause/resume, clear, autoscroll, copy, download
- [x] Virtualised rendering, ANSI colours, bounded buffer
- [x] Always-dark surface (board 04)

Notes (as built):
- Backend: `GET /ws/v1/projects/{id}/logs?services=a,b&tail=500|all&since=15m|<RFC3339>` runs `docker compose logs -f --timestamps --no-color` and sends **batched JSON arrays** `[{s,t,m}]` every 60 ms (max 500 per frame); a final `{"type":"end","error"?}` frame follows when docker's process exits. `GET /api/v1/projects/{id}/logs/download` returns a plain-text attachment (`<service> | <time> <message>`). `tail` is capped at 20 000, `services`/`since` are validated (argv only), foreign Origins get 403. Compose's line prefix (container name, or `proj-svc-N`) is mapped back to the service name using the project detail.
- Frontend: `useLogStream` (VueUse `useWebSocket`) keeps at most 20 000 lines; **pause** keeps collecting into a side buffer ("Paused · N new lines buffered") and resume flushes; any option change (services/since/context) restarts the stream with an empty buffer; an `end` frame shows `ended` + a Reconnect button. `LogView` uses VueUse `useVirtualList` (fixed 20 px rows, ~64 DOM rows for thousands of lines) on an always-dark surface (`--dd-term-*`, also in light theme).
- Toolbar: live/paused/ended/not-running pill, service multi-select (per-service colours), level (all / warn+ / error), since, search (debounced, escaped plain text or regex, match counter, invalid regex shown inline, Ctrl+F focuses it), timestamps, wrap, follow, pause/resume, clear, download. Click selects a line, shift-click a range, Copy puts `service | time message` lines on the clipboard, Esc clears the selection.
- Follow: pinned to the newest line; scrolling up (wheel or scrollbar) turns it off and a "Jump to latest" button appears; scrolling back to the bottom turns it on again. Filtering/clearing clamps the scroll offset (otherwise the virtual window rendered nothing until the next scroll event).
- Severity is heuristic (`utils/logs.ts`): `error`/`fatal`/`panic`/`Traceback`, JSON/logfmt `level=`, Apache `[mod:error]`, PHP; warn also covers `deprecated`, `failed`, `refused`, `timeout` and Apache `AH00558`. 28 vitest cases cover levels, search/highlight, formatting and the stream composable (mocked socket).
- Verified in a real browser on a throwaway project and on real (read-only) erp-back history: live streaming with follow, 64 DOM rows, scroll-up/jump, pause/resume buffering, level filter, plain+regex search, invalid regex, service filter (+URL `?service=`), selection + copy (clipboard contents checked), clear (stream continues), download endpoint, `end` after `down`, stream opened on a stopped project picks up lines once a container starts, light-theme empty state.
- Known limits: log timestamps are shown in local time without the date (the date is in the downloaded file and the copy); search is client-side over the buffered lines only (not the whole history on the server); one WebSocket + one `compose logs` process per open viewer.

Follow-up (after Phase 5):
- **Pull / Build buttons**, for the whole project and per service. Header: `Pull` (project-wide `compose pull --ignore-buildable`, so locally built images don't fail with "denied") and `Build…` (dialog: no-cache, pull newer base images). Row menu: `Pull image` (disabled with a hint for buildable services) and `Build image` (disabled with a hint for registry-only services), with a separator above the image entries. `compose config` tells which services have a `build:` section (`ComposeService.buildable`). Both run as normal jobs (output drawer, toast, one job per project). Verified in a real browser on a throwaway project with one built and one registry service: project build with `--no-cache` (no `CACHED` layers), per-service build (cache used), project pull (skipped the buildable image, pulled the other), per-service pull.
- **Log viewer stuck on "connecting"** (reported in dev mode): an older backend binary that lacks `/ws/v1/.../logs` answered the WebSocket handshake with the SPA's `index.html` (200), which never upgrades, so the socket hung. Fixes: unknown `/ws/*` is now a real JSON 404 (tested), and the toolbar shows `disconnected` with an error alert and a Reconnect button once the socket gives up. In dev, a backend started before a code change keeps running the old code: restart `go run` after pulling backend changes.


## Phase 6 — Editors: compose + env (2–3 d) — done
- [x] CodeMirror 6 (YAML, line numbers, search, history, indent), colours from the app tokens so light/dark follow the theme; dirty badge ("N lines changed"), Save (Ctrl+S) / Discard / Validate
- [x] Multi-file: one tab per Compose file (`?file=N`); validation with `docker compose config -q` where the edited file goes through **stdin** (nothing is written before Save; other files, `.env` interpolation and relative paths still resolve). Errors show a line (squiggle + gutter + list), warnings (an unset `${VAR}`) do not block. Save runs validation first; "Save anyway" for the rare case
- [x] Safe writes: file ids are server-derived (`compose-N`, `env`), never paths; ETag (content hash) via `If-Match`, 409 on a change on disk (UI offers "Reload from disk"); atomic temp+rename, mode preserved (a new `.env` is 0600), symlinks written through, one `.bak` next to the file (same mode), 2 MiB / UTF-8 text only
- [x] Env editor: Table / Raw toggle, add/remove rows, key validation (empty, malformed, duplicate), quotes/comments/`export`/multi-line values preserved, missing-file state ("not created yet"), `.env.example` banner ("Add them"; secret-looking example values are not copied), which services load it via `env_file`, which `${VARS}` Compose uses that are not defined
- [x] Secrets: values of secret-looking keys (PASSWORD, TOKEN, KEY, SECRET, DSN, ...) are masked **on the server**, so they are not in the browser until "Reveal secrets" (`?reveal=1`, logged). A masked secret serializes as the mask, and the server and the UI both refuse to save text that contains it, so a hidden value can never overwrite the real one
- [x] "Apply" hint after a save while containers are running (`up -d` recreates only what changed), verified: old value stays until Apply, then the container shows the new one
- Unsaved-changes guard on route change and tab close.

## Phase 7 — Exec terminal (1–2 d) — done
- [x] Real PTY (`creack/pty`) per session, TERM=xterm-256color, binary WebSocket (raw bytes both ways, JSON text frames only for control: resize / attached / exit / taken / gone). xterm.js in the browser with fit, unicode 11, links; arrow keys, history, Home/End, Ctrl+keys, application-cursor mode (vim/less), bracketed paste, resize (SIGWINCH) verified in a real browser
- [x] Sessions live on the server: reload, page change or a closed tab do not kill the shell; re-attach replays the last 256 KiB (terminal *queries* inside the replay are not answered again). A second viewer takes the session over ("taken"). Limits: 16 sessions, finished ones kept 10 min, unviewed ones closed after 6 h
- [x] Target service/replica, user (container default / root / UID 1000 / custom `name[:group]`), shell (auto bash -> sh, or custom), working directory, optional command instead of a shell. Everything user-supplied is validated and passed as separate argv entries (never through a local shell); the container comes from the project's own state
- [x] No leaked processes: docker does not forward the hangup, so every session tags its processes with `DOCKDUCK_TERM=<id>` and a cleanup (`/proc` scan, HUP then KILL, run as the session's own user because `/proc/PID/environ` of another user needs CAP_SYS_PTRACE) runs once on exit, close, reaper and graceful shutdown. Limit: a hard crash (SIGKILL) of DockDuck cannot sweep
- [x] Tabs per project (persist across pages), new-terminal dialog with a live `docker exec` preview, font size, copy, Ctrl+Shift+C/V, banners for ended (red on a non-zero exit) / taken / lost / gone; reconnect with backoff, and "gone" when the server restarted
- [x] Native terminal: "Open in <terminal>" runs the same `docker exec` in the system terminal (x-terminal-emulator, konsole, gnome-terminal, kitty, ...; override with `DOCKDUCK_TERMINAL`), loopback clients only, display env discovered from the session's sockets when started by a service; errors from a terminal that dies at once are reported, docker errors stay visible in the window
- [x] Browser guard (needed once an exec endpoint exists, before real auth): state-changing requests with a foreign `Origin` / `Sec-Fetch-Site` are refused (CSRF via `text/plain`), and the `Host` must be localhost / an IP literal (DNS rebinding)
- Defaults for the dialog (user, shell) are kept in the browser for now; Phase 9 moves them into Settings.

## Phase 8 — Auth + registries (2 d) — done
- [x] App login: argon2id (m=64 MiB, t=3, p=2; verify reads the parameters from the stored PHC hash and bounds them, so a crafted config cannot exhaust memory), min 10 characters. Sessions are in memory only (a restart logs everyone out), tokens stored hashed, HttpOnly + SameSite=Strict cookie (`Secure` behind TLS / `X-Forwarded-Proto: https`), idle 12 h + absolute 7 d, bound to the password: changing it ends every session
- [x] CSRF: every state-changing request needs `X-CSRF-Token` (per session) on top of the Origin/Host guard; logout and password change too. WebSocket upgrades need the cookie and a same-origin `Origin`, and **are closed when the session ends** (logout, password change, `dockduck passwd`), so an open terminal does not outlive the login
- [x] Throttling: 4 free attempts per client, then 5 s doubling to 15 min (IPv6 grouped by /64); a global cap of 100 failures / 10 min stops address rotation; while locked even the right password is refused; at most 4 argon2 checks at once. UI shows the lock countdown from `Retry-After`
- [x] Modes: loopback without a password stays open (as before) with a warning and a "Set password" button; **a non-loopback bind (`0.0.0.0`, `:port`, a LAN IP) refuses to start without a password**, and a networked server never runs open even if the password vanishes from the config. First-run setup is local-only (a remote browser cannot claim a fresh server). `dockduck passwd` (prompt or piped stdin) sets/resets the password and is the recovery path; a running server notices the config change within a second
- [x] Login screen with first-run setup, no flash of the app before the server says who you are, auto switch to the login screen when the session ends (events socket closes), Settings: set / change password, sign out
- [x] Registries: list from `~/.docker/config.json` (+ credential helpers' `list`), **names and usernames only: the stored secret is never read into the response**; sign in via `docker login --username U --password-stdin` (secret on stdin only: not in argv, not in logs, scrubbed from error text when a registry echoes it back); Docker Hub preset, re-authenticate, sign out; warns when docker keeps credentials unencrypted; docker's own error (401 ...) is shown. Verified against a real throwaway `registry:2` with basic auth and an isolated `DOCKER_CONFIG` (the real login file was untouched)
- Not done on purpose: TLS in the binary (use a reverse proxy; `Secure` cookies are handled), multi-user accounts.

## Phase 9 — Settings + polish (1–2 d) — done
- [x] Brand: the supplied duck (`assets/logo.svg`, favicon, `BrandMark` in the rail and on the login screen); the Lucide bird is gone
- [x] Settings page = board 09 in five cards, saved as you go. **Server-side** (shared by every browser, `GET/PUT /api/v1/settings`, partial updates, validated: scan depth 1–8, default context must exist in Docker, `sync_cli`; a changed depth rescans at once; the response carries the detected Compose version and never auth data). **Per browser** (localStorage): theme, accent colour (5 swatches; the readable text variant adapts to light/dark), density, terminal defaults (shell, user, font size; the new-terminal dialog reads the same store), "confirm before compose down" (volume removal always asks and still needs the typed name), "toasts for finished operations" (failures always show)
- [x] Command palette (Ctrl/Cmd+K): pages, every project, the open project's tabs and safe actions (up/restart/stop/pull), rescan, theme; fuzzy search (exact > prefix > word initials > substring > subsequence). `down` is deliberately not in it. Shortcuts: `?` help, `[` sidebar, `g` then `p/o/r/s` to navigate; single keys never fire while typing in a field, the code editor or a terminal
- [x] a11y: no unnamed buttons/inputs/links on any of the 11 routes, one `h1` per page, landmarks, contrast checked (muted/dim/status colours ≥ 4.5:1 in both themes); dialogs now trap Tab/Shift+Tab and return focus to the opener; the palette is a combobox/listbox with `aria-activedescendant`
- [x] Error and disconnected states: the server going away raises the banner within a second (no Docker-context button then), and a reconnect clears it and reloads contexts/projects, since events were missed meanwhile
- Not done on purpose: "Watch roots for changes" and "Keep running in the menu bar" from the mockup (a server has no menu bar; a filesystem watcher is a separate feature); i18n is string-ready only (all UI text is English literals in templates, no catalogue yet)

## Phase 10 — Packaging & release (1 d)
- [ ] Cross-compiled binaries (linux amd64/arm64, darwin), `--version`
- [ ] systemd unit, install script, reverse-proxy example (TLS), README usage
- [ ] Integration tests against a throwaway compose project

## Later / ideas
Multi-host agent mode, image/volume/network pages, compose templates, project groups/tags, notifications on unhealthy, Podman support.
