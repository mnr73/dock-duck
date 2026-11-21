# Run it on a server

DockDuck on a server gives you the same GUI for the compose projects there. Three things to set up: the binary, a service that keeps it running, and a safe way in.

::: danger Docker access is root access
Whoever can use Docker on a machine can read any file on it and run anything as root (a container can mount `/`). DockDuck's login is therefore the lock on the front door of the whole server. Use a long password, put TLS in front, and do not expose it to the internet without need.
:::

## The quick way

```sh
curl -fsSL https://mnr73.github.io/dock-duck/install.sh | sh -s -- --service
dockduck passwd
```

`--service` installs the binary to `/usr/local/bin` and a systemd unit that runs DockDuck as **you** on `127.0.0.1:9473`, enabled at boot. You then reach it through an SSH tunnel or a [reverse proxy](/server/reverse-proxy).

## Choosing how to reach it

| Way | When | How |
|---|---|---|
| **SSH tunnel** | Just you, no TLS to set up | `ssh -L 9473:127.0.0.1:9473 you@server`, then open <http://127.0.0.1:9473> |
| **Reverse proxy with TLS** | A team, or you want a domain name | [Reverse proxy and TLS](/server/reverse-proxy) |
| **Listen on the network directly** | A trusted private network only | `--addr 0.0.0.0:9473` plus a password. Plain HTTP: the password and the session cookie cross the network unencrypted |

## Rules DockDuck enforces

- Listening on anything other than loopback (`0.0.0.0`, `:9473`, a LAN address) **refuses to start without a password**.
- A server that is reachable from the network never runs open, even if the password is removed from the config.
- First-time password setup in the browser is only allowed from the machine itself, so nobody on the network can claim a fresh server. Use `dockduck passwd` on the server.

More in [Security](/server/security).
