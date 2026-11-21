# AGENTS.md — DockDuck

Read `doc/` first (start at `doc/README.md`). Phases live in `doc/04-phases.md`; work one phase at a time and tick its checklist.

## What this is
A monorepo: Go backend (`apps/server`) + Vue 3 frontend (`apps/web`), shipped as ONE binary (frontend embedded via `go:embed`). It manages *existing* Compose projects by shelling out to the `docker` CLI. User replies/communication: Persian; code, comments, commits, docs: English.

## Hard rules
1. **Never create/remove/update Docker contexts** (`docker context create/rm/update/import` are forbidden). Listing/inspecting is allowed. *Switching* is allowed and is app-level: the selected context is app state and every command gets `--context <name>`. `docker context use` (changing the CLI's global default) happens only if the user explicitly enables the "sync with Docker CLI" setting.
2. **Never parse Compose files ourselves.** Use `docker compose ... config --format json`, `ps --format json`, `ls -a`. Raw YAML is edited as plain text only.
3. **Every file path from the client is untrusted.** Resolve with `filepath.EvalSymlinks` and verify it is inside a registered root and is a known compose/env file of a discovered project. No path traversal.
4. **No shell strings.** Use `exec.CommandContext(name, args...)`; never `sh -c` with user input.
5. **Secrets never leave the server in plain text**: registry credentials are not stored by us (docker credential store does it) and are never returned; `.env` values are masked by the API unless explicitly revealed (`?reveal=1`, authenticated, audited in the log).
6. **Bind to 127.0.0.1 by default.** Auth is on by default; refusing to start on a non-loopback address without a password set.
7. Docker access == root on the host. Treat every endpoint as privileged.

## Conventions
- Go: stdlib `net/http` (1.22+ ServeMux patterns), `log/slog`, `coder/websocket`, `creack/pty`. `gofmt` + `go vet` + `golangci-lint`. Table-driven tests. No global state; pass dependencies via structs. Docker calls go through the `dockercli` package interface so tests can fake them.
- Frontend: Vue 3 `<script setup lang="ts">`, Pinia, Vue Router, Tailwind v4, CodeMirror 6 (editors), xterm.js (terminal/logs). **VueUse (`@vueuse/core`) is mandatory**: prefer its composables over hand-rolled ones (`useStorage`, `useColorMode`, `useWebSocket`, `useEventListener`, `useClipboard`, `useElementSize`, `useVirtualList`, `onKeyStroke`, `useDebounceFn`, `useEventSource`, `refDebounced`, `useMagicKeys`, `useTimeAgo`…). Check VueUse before writing a util.
- **Icons**: package `@iconify-json/lucide` via `unplugin-icons`. ALL icons are declared in ONE module, `apps/web/src/icons.ts`, as semantic re-exports (`export { default as IconPlay } from "~icons/lucide/play"`); components import only from `@/icons`, never from `~icons/*` directly. Add new concepts there with a semantic name. Use the tree-shaking plugin (`moduleSideEffects: false` for `~icons/*`) as in shiva-connect.
- Style reference = the **newer** projects. erp-front is legacy: do not copy conventions from it. Take ideas (not code) from `~/prj/shiva/shiva-connect` (vite config, `src/icons.js`, prettier sort-imports config, Btn/IconBtn/Badge/Alert/SegmentedControl components, composables) and `~/prj/mnr/zhoon_front`. Baseline: double quotes, 4-space indent, trailing commas, `@trivago/prettier-plugin-sort-imports`, `@/` alias, no auto-import, generic components (slots over baked-in props/buttons), `vue-sonner` for toasts.
- API: JSON under `/api/v1`, WebSocket under `/ws/v1`. Contract in `doc/05-api.md` — update it in the same change as the code.
- Design tokens come from `doc/design/DockDuck.html`; dark is primary, light is complete, log/terminal surfaces are always dark.
- Commits: Conventional Commits. Do not commit on the user's behalf; he commits himself.

## Commands (once scaffolded)
- `make dev` — Go server (air/`go run`) + Vite dev server with proxy
- `make build` — build web, embed, build `bin/dockduck`
- `make test` / `make lint`

## Environment notes
- Go is not yet installed on the dev machine; install it before Phase 0 (`doc/04-phases.md`).
- Network to registries (npm/Go proxy/Docker Hub) is slow; run installs in the background.
- Don't run tests that touch the real Docker daemon against the user's running stacks; integration tests use their own throwaway compose project (`dockduck-test-*`).

## Decision log
- Icons: Lucide (matches the Claude Design mockup) through Iconify (`@iconify-json/lucide` + `unplugin-icons`), all declared in `apps/web/src/icons.ts`. (Hugeicons was considered and rejected.)
