# 06 — Security

Access to the Docker socket is root-equivalent. DockDuck therefore ships secure by default.

- **Bind**: `127.0.0.1:8080` default. Non-loopback bind requires a configured password, else refuse to start.
- **App auth**: first-run password setup, argon2id hash in config (0600), HttpOnly+SameSite=Strict session cookie (`Secure` when behind TLS), CSRF token on mutations, login rate-limit/backoff, idle + absolute session expiry. WS upgrade checks `Origin` and the session.
- **TLS**: terminate at a reverse proxy (example in `deploy/`); optional `--tls-cert/--tls-key`.
- **Filesystem**: file API only touches discovered compose/.env files; `EvalSymlinks` + prefix check against registered roots; size limit (2 MB); atomic writes; `.bak`.
- **Process execution**: argument arrays only; project name/service/user validated against `^[A-Za-z0-9][A-Za-z0-9_.-]*$` (user: numeric uid, `uid:gid` or name).
- **Secrets**: `.env` values masked by default (heuristic on key names: PASSWORD, SECRET, TOKEN, KEY…) and only revealed via an explicit, authenticated call; registry secrets go through stdin to `docker login`, are never logged or returned. Logs redact `Authorization`/passwords.
- **Audit**: structured log line for up/down/exec/file write/reveal/login with user + IP.
- **Resource limits**: max concurrent log/exec streams, bounded buffers, process kill on disconnect.

## As built (Phase 8)
- Auth, sessions, CSRF, throttling and the bind rule are described in `04-phases.md` (Phase 8). Recovery of a lost password: `dockduck passwd` on the machine.
- A hard crash (SIGKILL) of the server cannot sweep processes that terminal shells left inside containers; every normal stop (SIGTERM / Ctrl+C) does.
- Config file `auth.password_hash` is a hash only; the file is written 0600 atomically.
