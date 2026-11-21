# AI agent skill

The folder [`skills/dock-duck/`](https://github.com/mnr73/dock-duck/tree/main/skills/dock-duck) in the repository is a skill (`SKILL.md` plus a few references) that tells coding agents such as Claude Code, Hermes and Codex how DockDuck works: how to install and update it, the rules that surprise people (loopback bind, password, proxy headers), where its files live, and how to fix the common errors.

| File | Content |
|---|---|
| `SKILL.md` | install and update, the rules, where things are, an error-to-cause table |
| `references/cli-and-config.md` | every command, flag, environment variable, the config file and the systemd unit |
| `references/troubleshooting.md` | symptoms and fixes: proxy, password, Docker, updates |
| `references/http-api.md` | auth and CSRF flow, endpoint list, a curl example |

## Install it for an agent

```sh
# Claude Code (in your project)
mkdir -p .claude/skills
curl -fsSL https://codeload.github.com/mnr73/dock-duck/tar.gz/refs/heads/main \
  | tar -xz --strip-components=2 -C .claude/skills dock-duck-main/skills/dock-duck

# Hermes
mkdir -p ~/.hermes/skills
curl -fsSL https://codeload.github.com/mnr73/dock-duck/tar.gz/refs/heads/main \
  | tar -xz --strip-components=2 -C ~/.hermes/skills dock-duck-main/skills/dock-duck
```

Any agent that reads Markdown can also read `SKILL.md` directly.

## Keeping it right

When a command, a flag, a setting or an endpoint changes, update `skills/dock-duck/` in the same commit. The error messages in the troubleshooting table are the exact strings DockDuck prints, so a rename in the code must be mirrored there.
