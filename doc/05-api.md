# 05 — API contract (draft)

Base: `/api/v1` (JSON). Auth: session cookie + `X-CSRF-Token` on mutating requests. Errors: `{ "error": { "code": "string", "message": "string" } }`.
Context selection: header `X-Docker-Context: <name>` or query `?context=<name>` (for WebSockets), falling back to the saved selection / default. `GET /health` → `{ok, version, home}`.

## Auth
| | |
|---|---|
| `GET /auth/status` | `{configured, required, authenticated, setup_here, csrf?}` (public; `csrf` only for a logged-in browser) |
| `POST /auth/setup {password}` | first password, loopback clients only; 409 if one exists; 400 `weak_password` |
| `POST /auth/login {password}` | 401 `bad_credentials`, 429 `locked` / `busy` with `Retry-After` |
| `POST /auth/logout` | needs the CSRF token |
| `POST /auth/password {current, next}` | ends all other sessions, returns a fresh session |

Everything under `/api/` and `/ws/` except `/api/v1/health` and `/api/v1/auth/*` needs the session cookie once a password exists (401 `unauthenticated`); state-changing requests also need `X-CSRF-Token` (403 `bad_csrf`). Without a password and on loopback the API stays open.

## Registries
| | |
|---|---|
| `GET /registries` | `{config_path, store, plaintext, entries[{server,name,hub,username?,store}]}`: never a secret |
| `POST /registries/login {server?, username, password}` | `server` empty = Docker Hub; 422 `registry_refused` carries docker's message |
| `POST /registries/logout {server?}` | 204 |

## Contexts
| | |
|---|---|
| `GET /contexts` | `[{name, description, endpoint, current, available, error?}]` |
| `PUT /contexts/selected` | `{name}` — switch (app-level; `sync_cli:true` in settings also runs `docker context use`). No create/delete endpoints by design. |

## Roots & projects
| | |
|---|---|
| `GET /roots`, `POST /roots {path}`, `DELETE /roots/{id}` | |
| `POST /roots/rescan` | returns discovered count |
| `GET /projects` | `[{id,name,dir,files[],env_file?,root?,status,running,total}]` (status: running/partial/stopped/unknown; `root` empty = known only to Docker). `POST /roots` → 201 `{root,projects}`, 409 duplicate, 400 non-absolute/nonexistent path. |
| `GET /projects/{id}` | `{project, services[{name,image,status,health?,buildable,ports[],containers[{id,name,state,status,health?,exit_code,image,ports[]}]}], config_error, active_job?}` |
| `GET /projects/{id}/down-preview` | `{project, containers[], networks[], volumes[]}` removed by `down` / `down -v` |
| `GET /projects/{id}/services/{svc}` | trimmed inspect of the service's containers (no env vars) |
| `GET /projects/{id}/stats` | `{<container id>: {cpu_percent, mem_usage, mem_percent}}`, running containers, cached 3 s |
| `POST /projects/{id}/actions/{up|down|start|stop|restart|build|pull}` | body `{services?[], volumes?, remove_orphans?, build?, no_cache?, pull?}` → 202 `{job_id, job}`; 409 `busy` if the project has a running job; 400 on unknown action/bad names; `down` takes no services; project-wide `pull` adds `--ignore-buildable`; `build` honours `no_cache` / `pull` |
| `GET /projects/{id}/services/{svc}` | inspect |

## Files
| | |
|---|---|
| `GET /projects/{id}/files` | compose files + env |
| `GET /projects/{id}/files/{fid}` | `{content, etag}` |
| `PUT /projects/{id}/files/{fid}` | `{content}` + `If-Match: etag`; 409 on conflict |
| `POST /projects/{id}/files/{fid}/validate` | `{ok, errors[]}` |
| `GET /projects/{id}/env?reveal=0|1` | structured env, secrets masked unless `reveal` |

## Ports, jobs, registries, settings
| | |
|---|---|
| `GET /projects/{id}/logs/download?services=&since=&tail=` | plain-text attachment of the log history |
| `GET /ports` | `[{host_ips[],host_port,container_port,proto,project?,service?,container,container_id,image}]`, running containers only |
| `GET /jobs`, `GET /jobs/{id}` | `{id,project,action,services,status: running|succeeded|failed,error,started_at,ended_at}` |
| `GET /registries` | `[{url,username?,store}]` (never secrets) |
| `POST /registries/login` | `{url,username,secret}` |
| `DELETE /registries/{url}` | logout |
| `GET /settings` | `{default_context, sync_cli, scan_depth, max_scan_depth, compose_version, config_path}` |
| `PUT /settings` | any subset of `default_context` ("" = last used; must exist), `sync_cli`, `scan_depth` (1–8; a change rescans); 400 `invalid_scan_depth` / `invalid_context`. Theme, accent, density and terminal defaults are per browser and not part of the API |

## WebSocket (`/ws/v1`)
| Path | Direction / frames |
|---|---|
| `/events` | server→client JSON: `{type:"container", action, project, service, ...}` |
| `/projects/{id}/logs?services=a,b&tail=500\|all&since=15m` | server→client JSON arrays `[{s,t,m}]` (batched), then `{"type":"end","error"?}` |
| `/projects/{id}/exec?service=app&user=1000&cmd=...&cols=&rows=` | binary = PTY bytes both ways; text = `{"type":"resize","cols":,"rows":}` |
| `/jobs/{id}` | server→client output lines, then `{type:"done",ok:bool}` |

## Files (Phase 6)

| Method | Path | Notes |
|---|---|---|
| GET | `/projects/{id}/files` | `[{id,kind,name,path,exists}]`: `compose-0..N` in merge order, then `env` |
| GET | `/projects/{id}/files/{fid}` | `{content,etag,mode,...}`; `env` is masked unless `?reveal=1` (`X-Content-Masked: 1`); `Cache-Control: no-store` |
| PUT | `/projects/{id}/files/{fid}` | body `{content}`, header `If-Match: <etag>` (empty = create a missing `.env`). 409 `conflict`, 413, 422 `not_text`, 400 `masked_value` |
| POST | `/projects/{id}/files/{fid}/validate` | body `{content}` -> `{ok,problems:[{severity,line?,message}]}`; Compose files only |
| GET | `/projects/{id}/env` | structured view: `lines` (masked unless `?reveal=1`), `example` (missing keys), `usage` (`env_file` services, interpolated vars) |

## Terminals (Phase 7)

| Method | Path | Notes |
|---|---|---|
| POST | `/projects/{id}/terminals` | body `{service, container?, user?, shell?, command?, workdir?, cols?, rows?}` -> 201 session info. 409 `not_running`, 429 `too_many` |
| GET | `/terminals` | all sessions |
| DELETE | `/terminals/{id}` | kills the shell and sweeps what it left in the container |
| GET | `/terminals/system` | `{available, name?, reason?}` native terminal |
| POST | `/projects/{id}/terminals/system` | same body; opens the system terminal (loopback only) |
| WS | `/ws/v1/terminals/{id}` | server: TEXT `{"type":"attached","replay":N}`, then one BINARY replay frame, then BINARY output; TEXT `{"type":"exit","code"}` / `taken` / `slow` / `gone`. client: BINARY keyboard bytes, TEXT `{"type":"resize","cols","rows"}` |

## Update
| | |
|---|---|
| `GET /update[?refresh=1]` | `{current, latest, available, url, notes, date, can_install, reason?, managed}`; cached 2 min; once an update is installed: `{installed, current}`; 502 `update_check_failed` if GitHub cannot be reached |
| `POST /update/install` | downloads, verifies the checksum and replaces the binary, answers `{installed, restart: "self"|"service"}`, then the process restarts; 422 `update_failed` (dev build, not writable, checksum mismatch, already latest), 409 if one was already installed |
