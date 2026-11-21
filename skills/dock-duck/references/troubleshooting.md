# Troubleshooting

Work from the message. Check the cheap things first.

## Can't reach DockDuck at all
`systemctl status dockduck` / `ss -ltn | grep 9473`. The page shows a banner "Can't reach the DockDuck server" and clears it by itself when it is back.

## "Can't reach the Docker daemon"
The selected context does not answer. Run `docker ps` as the same user. Is the user in the `docker` group? Switch context on the Contexts page. Projects stay listed; actions fail until Docker is back.

## No password / remote access
- Remote browser says no password is set: setup is only allowed on the machine. `dockduck passwd` there, reload.
- `refusing to listen on ... without a password`: same fix, before starting with a non-loopback `--addr`.
- `this host name is not allowed while no password is set`: set a password, or use localhost / an IP.

## Behind a reverse proxy
- Actions say "cross-origin request refused": the proxy changed `Host`. Forward the original (`proxy_set_header Host $host`).
- Terminal or logs stay "connecting": forward WebSockets (`Upgrade` and `Connection` headers), long read timeout.
- Cookie not kept: the proxy must send `X-Forwarded-Proto: https` so the cookie is `Secure` on an HTTPS page.

## A project is missing
It must be inside a root (Roots page), within the scan depth (Settings), and contain `compose.yaml`, `compose.yml`, `docker-compose.yaml` or `docker-compose.yml`. Rescan. Projects Docker runs from elsewhere appear under "Outside your roots".

## Environment tab shows "hidden"
That is a secret; press Reveal. A hidden value is never saved over the real one.

## Updates
- `no release found`: the repository has no stable release yet (prereleases are ignored).
- `not writable`: run the install command or `dockduck update` with sudo, or `chown` the binary to the service user.
- Did not come back after an in-app update: start `dockduck` again; under systemd the unit needs `Restart=always`.
- `checksum mismatch`: nothing was installed; retry, and report it if it persists.

## Reporting
Issues: https://github.com/mnr73/dock-duck/issues with `dockduck version`, the OS and the log (`journalctl -u dockduck`).
