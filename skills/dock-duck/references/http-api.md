# HTTP API

Base `/api/v1`, JSON. Errors: `{"error":{"code":"...","message":"..."}}`. Once a password exists everything except `/api/v1/health` and `/api/v1/auth/*` needs the session cookie (401 `unauthenticated`); non-GET also needs `X-CSRF-Token` (403 `bad_csrf`) and a same-origin `Origin` or none. Context: header `X-Docker-Context` or `?context=`.

```sh
B=http://127.0.0.1:9473
curl -c jar -H 'Content-Type: application/json' -d '{"password":"..."}' $B/api/v1/auth/login
CSRF=$(curl -s -b jar $B/api/v1/auth/status | jq -r .csrf)
curl -b jar -X POST -H "X-CSRF-Token: $CSRF" $B/api/v1/projects/<id>/actions/restart
```

| Area | Endpoints |
|---|---|
| Health, auth | `GET /health`; `GET /auth/status`, `POST /auth/setup|login|logout|password` |
| Contexts | `GET /contexts`, `PUT /contexts/selected {name}` (no create/delete) |
| Roots | `GET/POST /roots`, `DELETE /roots/{id}`, `POST /roots/rescan` |
| Projects | `GET /projects`, `GET /projects/{id}`, `/stats`, `/down-preview`, `/services/{svc}` |
| Actions | `POST /projects/{id}/actions/{up|down|start|stop|restart|build|pull}` body `{services?, volumes?, remove_orphans?, build?, no_cache?, pull?}`; 202 `{job_id}`; 409 `busy` while one runs; `down` takes no services, `volumes:true` only deliberately |
| Jobs | `GET /jobs`, `GET /jobs/{id}`, WebSocket `/ws/v1/jobs/{id}` |
| Logs | WebSocket `/ws/v1/projects/{id}/logs`, `GET /projects/{id}/logs/download` |
| Files | `GET /projects/{id}/files`, `GET|PUT /files/{fid}` (`If-Match` etag), `POST /files/{fid}/validate`, `GET /projects/{id}/env?reveal=1` |
| Terminal | `POST /projects/{id}/terminals`, `GET /terminals`, `DELETE /terminals/{id}`, WebSocket `/ws/v1/terminals/{id}` |
| Ports | `GET /ports` |
| Registries | `GET /registries`, `POST /registries/login|logout` |
| Settings | `GET|PUT /settings` (partial: `default_context`, `sync_cli`, `scan_depth`) |
| Update | `GET /update[?refresh=1]`, `POST /update/install` |
| Live events | WebSocket `/ws/v1/events` |

Full contract: https://github.com/mnr73/dock-duck/blob/main/doc/05-api.md
