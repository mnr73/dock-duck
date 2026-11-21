# Troubleshooting

## "Can't reach the DockDuck server"
The page lost its connection. Is `dockduck` still running (`systemctl status dockduck`)? The banner clears by itself when it is back.

## "Can't reach the Docker daemon"
The selected context does not answer. Check `docker ps` works for the same user, that the user is in the `docker` group, and switch context on the **Contexts** page. Projects stay listed; actions fail until Docker is back.

## "No password is set on this server, so it cannot be used from the network yet"
You opened it from another machine (or through a proxy) and no password exists. Setting it from the browser is only allowed on the machine itself. Run `dockduck passwd` on the server, then reload.

## "refusing to listen on … without a password"
You asked for a network address. Run `dockduck passwd` first, or keep the default `127.0.0.1`.

## "this host name is not allowed while no password is set"
You reached an open server by a domain name. Set a password (any host name is then accepted), or use `localhost` / the IP.

## Behind a proxy: login works but the terminal or logs say "connecting" forever
WebSockets are not being forwarded. Add the `Upgrade`/`Connection` headers (see [Reverse proxy](/server/reverse-proxy)).

## Behind a proxy: every action says "cross-origin request refused"
The proxy changes the `Host` header. Forward the original (`proxy_set_header Host $host`).

## A project is missing
It must be inside a root (**Roots** page) and within the scan depth (Settings), and contain `compose.yaml`, `compose.yml`, `docker-compose.yaml` or `docker-compose.yml`. Press the rescan button.

## The Environment tab says a variable is "hidden"
That is a secret. Press **Reveal secrets** to see it; a hidden value is never saved over the real one.

## Settings → Updates: "not writable"
The binary is in a folder your user cannot write. Update from a shell with `sudo` (run the install command again), or `sudo chown $USER /usr/local/bin/dockduck`.

## After updating from Settings DockDuck did not come back
Run it again (`dockduck`). Under systemd make sure the unit has `Restart=always` ([systemd](/server/systemd)).

## Reporting a problem
Open an issue at <https://github.com/mnr73/dock-duck/issues> with the output of `dockduck version`, your OS, and the log (`journalctl -u dockduck` or the terminal output).
