# Reverse proxy and TLS

DockDuck speaks plain HTTP. To reach it from outside the machine, put a reverse proxy in front that provides HTTPS. DockDuck keeps listening on `127.0.0.1:9473`.

What the proxy **must** do:

1. **Forward `Host`** unchanged (`proxy_set_header Host $host`). DockDuck compares it with the browser's `Origin` to refuse cross-site requests.
2. **Set `X-Forwarded-Proto: https`.** DockDuck then marks the session cookie `Secure`.
3. **Allow WebSockets** (`Upgrade` / `Connection` headers). Logs, terminals and live updates use them.
4. **Not time out idle connections quickly.** A terminal can sit idle; use a long read timeout.

A domain name (`duck.example.com`) is accepted as long as a password is set. Without a password only `localhost` and IP addresses are accepted, to stop DNS-rebinding attacks on an open server.

## Caddy

Automatic HTTPS and WebSockets, no extra options:

```text
duck.example.com {
    reverse_proxy 127.0.0.1:9473
}
```

## nginx

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ""      close;
}

server {
    listen 80;
    server_name duck.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    http2 on;
    server_name duck.example.com;

    ssl_certificate     /etc/letsencrypt/live/duck.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/duck.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:9473;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 1h;
        proxy_send_timeout 1h;
        proxy_buffering off;
    }
}
```

Both files are in [`deploy/`](https://github.com/mnr73/dock-duck/tree/main/deploy). This nginx setup was tested with a non-localhost host name: login, the `Secure` cookie, the CSRF/Origin checks, and WebSockets all work through it, and a WebSocket from a foreign origin is refused.

## Another login in front

If your proxy already adds its own password (basic auth, an SSO gateway), DockDuck still has its own login, which is what protects the terminal and the Docker socket if the proxy is ever bypassed. Keep both.
