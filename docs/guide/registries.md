# Registries

**Registries** shows where Docker is signed in (Docker Hub, GitHub Container Registry, your own) and lets you sign in or out. It uses `docker login` and `docker logout`, so it changes the same credentials the `docker` command uses.

- The list shows names and usernames only. A stored secret is never read back.
- The password goes to Docker on standard input. It is never in a process list, a log or an error message.
- Prefer an **access token** with the least scope (for example read-only pull) to your account password.
- If Docker keeps credentials unencrypted in `~/.docker/config.json`, DockDuck says so. A credential helper (`docker-credential-secretservice`, `pass`) keeps them in your system keychain instead.
