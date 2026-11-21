# HTTP API

The web UI uses a JSON API under `/api/v1` and WebSockets under `/ws/v1`. You can script against it.

- **Errors:** `{ "error": { "code": "...", "message": "..." } }`
- **Auth:** once a password exists every route except `/api/v1/health` and `/api/v1/auth/*` needs the session cookie. Mutating requests also need the `X-CSRF-Token` from `GET /api/v1/auth/status` (only returned to a logged-in session) and a same-origin `Origin` (or none, for curl).
- **Context:** `X-Docker-Context: <name>` (or `?context=` for WebSockets) overrides the selected one.

A script example:

```sh
B=http://127.0.0.1:9473
curl -c jar -H 'Content-Type: application/json' -d '{"password":"…"}' $B/api/v1/auth/login
CSRF=$(curl -s -b jar $B/api/v1/auth/status | jq -r .csrf)
curl -b jar -X POST -H "X-CSRF-Token: $CSRF" $B/api/v1/projects/<id>/actions/restart
```

The full list of endpoints, request and response shapes is in the repository:
[doc/05-api.md](https://github.com/mnr73/dock-duck/blob/main/doc/05-api.md).

Without a password and on loopback the API is open (no cookie needed), as before.
