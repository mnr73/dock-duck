# 03 — Monorepo structure

```
dock-duck/
├─ AGENTS.md
├─ README.md
├─ Makefile
├─ pnpm-workspace.yaml          # (Phase 0) packages: apps/web, packages/*
├─ doc/                         # plans, API, design (this folder)
│  └─ design/DockDuck.html
├─ apps/
│  ├─ server/                   # Go module: dockduck
│  │  ├─ go.mod
│  │  ├─ cmd/dockduck/          # main.go: flags, config, wiring
│  │  └─ internal/
│  │     ├─ api/                # HTTP handlers, routing, middleware
│  │     ├─ auth/               # app login, sessions, CSRF, rate limit
│  │     ├─ config/             # config file load/save
│  │     ├─ dockercli/          # CLI wrapper + interface + fake
│  │     ├─ contexts/           # list/inspect/ping/switch (no create/remove)
│  │     ├─ discovery/          # root scanning, project model
│  │     ├─ projects/           # state, actions, jobs
│  │     ├─ files/              # safe compose/.env read/write/validate
│  │     ├─ logs/ exec/ events/ # streaming pieces (WS)
│  │     ├─ ports/              # aggregate published ports
│  │     ├─ registry/           # docker login/logout/status
│  │     └─ webui/              # go:embed dist/
│  └─ web/                      # Vue 3 SPA
│     ├─ src/
│     │  ├─ icons.ts            # ALL icons (lucide) re-exported with semantic names
│     │  ├─ api/                # typed client; WS via VueUse useWebSocket
│     │  ├─ components/         # generic UI (Btn, IconBtn, Menu, Dialog, Table, Tabs…)
│     │  ├─ composables/        # app composables built on @vueuse/core
│     │  ├─ features/           # contexts, projects, services, compose, env,
│     │  │                      # logs, terminal, ports, registries, settings
│     │  ├─ stores/             # Pinia
│     │  ├─ styles/             # tokens (from design), tailwind
│     │  └─ router/
│     └─ index.html
├─ packages/
│  └─ api-types/                # TS types for the API (hand-written or generated from doc/05-api.md/OpenAPI)
├─ deploy/                      # systemd unit, example reverse-proxy, install script
└─ .github/workflows/           # CI: lint, test, build matrix (linux/amd64, arm64, darwin)
```

Why this shape: a single Go module + a single web app is enough; `packages/` exists only for shared types so the contract has one source. Release = `make build` → `bin/dockduck`.
