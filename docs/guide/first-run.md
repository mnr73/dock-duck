# First run

```sh
dockduck
```

It listens on `127.0.0.1:9473` (this machine only) and prints the address.

## 1. Add a folder

Open **Roots** in the left rail and add the folder that holds your compose projects. DockDuck scans it (4 folder levels deep by default; change it in [Settings](/guide/settings)) and lists every project with a `compose.yaml`, `compose.yml`, `docker-compose.yaml` or `docker-compose.yml`.

Projects that Docker already runs but that live outside your folders appear under **Outside your roots**.

## 2. Pick a Docker context

The status bar shows the engine in use. **Contexts** lists the contexts Docker already has and lets you switch. DockDuck never creates or deletes a context.

## 3. Set a password (recommended)

On `127.0.0.1` with no password, anyone who can open the page on this machine controls your containers. **Settings → Login → Set password** (at least 10 characters), or:

```sh
dockduck passwd
```

A password is **required** to listen on any other address; see [Server setup](/server/).

Forgot it? Run `dockduck passwd` on the machine again. It does not ask for the old one, because whoever can run it already owns the box.

## Ports and addresses

| You want | Run |
|---|---|
| This machine only (default) | `dockduck` |
| Another port | `dockduck --addr 127.0.0.1:7000` |
| From your network | `dockduck --addr 0.0.0.0:9473` (needs a password first, and put TLS in front, see [Reverse proxy](/server/reverse-proxy)) |
