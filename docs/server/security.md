# Security

## What you are protecting

Docker access equals root on the machine. DockDuck can start containers, open shells in them and edit compose files, so anyone who gets in can do what root can do. Everything below follows from that.

## What DockDuck does

| Area | Behaviour |
|---|---|
| **Where it listens** | `127.0.0.1:9473`. Any other address refuses to start without a password |
| **Login** | argon2id password hash (stored `0600`), `HttpOnly` + `SameSite=Strict` session cookie, `Secure` behind HTTPS. Sessions end after 12 h idle or 7 days, and when the password changes |
| **Brute force** | 4 free attempts, then a growing lock (5 s up to 15 min); while locked even the right password is refused; a global cap stops attempts spread over many addresses |
| **CSRF** | Every change needs a per-session token and a same-origin `Origin`. WebSockets check `Origin` and the session |
| **DNS rebinding** | While no password is set, only `localhost` and IP literals are accepted as the host name |
| **Sign out** | Closes the terminals and live streams of that session |
| **Files** | Only the project's own compose files and `.env` can be read or written. Paths come from the server, never from the browser. Writes are atomic with a `.bak` |
| **Secrets** | `.env` secrets are masked on the server until you reveal them. Registry passwords go to `docker login` over stdin and are never stored, logged or returned |
| **Commands** | Always argument lists, never a shell string. Names are validated |
| **Updates** | Only when you ask. The download must match `checksums.txt` of the same release |

## What it does not do

- **No TLS of its own.** Use a [reverse proxy](/server/reverse-proxy) or an SSH tunnel.
- **No multiple users or roles.** One password.
- **No protection from someone who can already run Docker or read the config file on the machine.** They own it already.

## Recommendations

1. Keep it on `127.0.0.1` and reach it through an SSH tunnel when you can.
2. Otherwise: HTTPS reverse proxy, a long password (a few random words), and restrict the proxy by IP or VPN if possible.
3. Run it as an ordinary user in the `docker` group, not as root.
4. `dockduck passwd` on the machine resets a lost password and ends every session.
5. Updates replace the binary: install from the official release only, and keep the in-app update to the official repository (it is not configurable from the page).
