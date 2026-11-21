# 05 — API contract (draft)

Base: `/api/v1` (JSON). Auth: session cookie + `X-CSRF-Token` on mutating requests. Errors: `{ "error": { "code": "string", "message": "string" } }`.
Context selection: header `X-Docker-Context: <name>` (falls back to the saved selection / default).

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
| `GET /projects` | `[{id,name,dir,files[],env_file?,status,running,total}]` |
| `GET /projects/{id}` | adds services[] |
| `POST /projects/{id}/actions/{up|down|start|stop|restart|build|pull}` | body `{services?[], volumes?, remove_orphans?}` → `{job_id}` |
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
| `GET /ports` | `[{host_ip,host_port,proto,container_port,project,service,container}]` |
| `GET /jobs/{id}` | status |
| `GET /registries` | `[{url,username?,store}]` (never secrets) |
| `POST /registries/login` | `{url,username,secret}` |
| `DELETE /registries/{url}` | logout |
| `GET/PUT /settings` | theme, scan depth, terminal defaults, default context, behaviour |

## WebSocket (`/ws/v1`)
| Path | Direction / frames |
|---|---|
| `/events` | server→client JSON: `{type:"container", action, project, service, ...}` |
| `/projects/{id}/logs?services=a,b&tail=200&timestamps=1` | server→client text lines |
| `/projects/{id}/exec?service=app&user=1000&cmd=...&cols=&rows=` | binary = PTY bytes both ways; text = `{"type":"resize","cols":,"rows":}` |
| `/jobs/{id}` | server→client output lines, then `{type:"done",ok:bool}` |
