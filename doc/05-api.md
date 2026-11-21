# 05 — API contract (draft)

Base: `/api/v1` (JSON). Auth: session cookie + `X-CSRF-Token` on mutating requests. Errors: `{ "error": { "code": "string", "message": "string" } }`.
Context selection: header `X-Docker-Context: <name>` or query `?context=<name>` (for WebSockets), falling back to the saved selection / default. `GET /health` → `{ok, version, home}`.

## Auth
| | |
|---|---|
| `GET /auth/status` | `{setup_required, authenticated, user}` |
| `POST /auth/setup` | first-run password |
| `POST /auth/login` / `POST /auth/logout` | |

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
| `GET/PUT /settings` | theme, scan depth, terminal defaults, default context, behaviour |

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
