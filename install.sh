#!/usr/bin/env sh
# DockDuck installer and updater. Run it again to update.
#
#   curl -fsSL https://mnr73.github.io/dock-duck/install.sh | sh
#
# Options (after `sh -s --` when piped):
#   --version vX.Y.Z   install that release instead of the latest
#   --dir PATH         install into PATH (default: /usr/local/bin if writable or via sudo, else ~/.local/bin)
#   --service          also install and start a systemd service (Linux, needs sudo)
#   --user NAME        user the service runs as (default: the current user; must be in the docker group)
#
# The download is verified against the release's checksums.txt before anything is installed.
set -eu

REPO="mnr73/dock-duck"
VERSION=""
DIR=""
SERVICE=0
RUN_USER="$(id -un)"

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="${2:?--version needs a value}"; shift 2 ;;
    --dir) DIR="${2:?--dir needs a value}"; shift 2 ;;
    --service) SERVICE=1; shift ;;
    --user) RUN_USER="${2:?--user needs a value}"; shift 2 ;;
    -h|--help) sed -n '2,13p' "$0" 2>/dev/null || true; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
done

say() { printf '%s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not installed"; }

need tar
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
  fetch_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
  fetch_stdout() { wget -qO- "$1"; }
else
  die "curl or wget is required"
fi

# ---- platform
case "$(uname -s)" in
  Linux) OS=linux ;;
  Darwin) OS=darwin ;;
  *) die "unsupported system: $(uname -s) (Linux and macOS only)" ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported CPU: $(uname -m) (amd64 and arm64 only)" ;;
esac

# ---- version
if [ -z "$VERSION" ]; then
  VERSION="$(fetch_stdout "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$VERSION" ] || die "could not find the latest release (is there one yet? https://github.com/$REPO/releases)"
fi
case "$VERSION" in v[0-9]*.[0-9]*.[0-9]*) ;; *) die "bad version: $VERSION" ;; esac

# ---- destination
if [ -z "$DIR" ]; then
  if [ -w /usr/local/bin ] || [ "$(id -u)" = 0 ]; then DIR=/usr/local/bin
  elif command -v sudo >/dev/null 2>&1 && [ "$SERVICE" = 1 ]; then DIR=/usr/local/bin
  else DIR="$HOME/.local/bin"; fi
fi
TARGET="$DIR/dockduck"

CURRENT=""
if [ -x "$TARGET" ]; then CURRENT="$("$TARGET" version 2>/dev/null || true)"; fi
if [ "$CURRENT" = "$VERSION" ]; then
  say "DockDuck $VERSION is already installed at $TARGET"
  [ "$SERVICE" = 1 ] || exit 0
fi

# ---- download and verify
NAME="dockduck_${VERSION#v}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT INT TERM

if [ "$CURRENT" != "$VERSION" ]; then
  say "Downloading DockDuck $VERSION ($OS/$ARCH)..."
  fetch "$BASE/$NAME" "$TMP/$NAME" || die "download failed: $BASE/$NAME"
  fetch "$BASE/checksums.txt" "$TMP/checksums.txt" || die "could not download checksums.txt"

  WANT="$(awk -v n="$NAME" '$2==n || $2=="*"n {print $1}' "$TMP/checksums.txt" | head -n 1)"
  [ -n "$WANT" ] || die "checksums.txt has no entry for $NAME"
  if command -v sha256sum >/dev/null 2>&1; then GOT="$(sha256sum "$TMP/$NAME" | awk '{print $1}')"
  else GOT="$(shasum -a 256 "$TMP/$NAME" | awk '{print $1}')"; fi
  [ "$WANT" = "$GOT" ] || die "checksum mismatch: the download is corrupt or was tampered with; nothing was installed"
  say "Checksum OK."

  tar -xzf "$TMP/$NAME" -C "$TMP" dockduck || die "the archive has no dockduck binary"
  chmod 755 "$TMP/dockduck"
  "$TMP/dockduck" version >/dev/null 2>&1 || die "the downloaded binary does not run on this system"

  # install next to the target and rename over it: a running DockDuck keeps working
  SUDO=""
  if [ ! -d "$DIR" ]; then mkdir -p "$DIR" 2>/dev/null || SUDO="sudo"; [ -z "$SUDO" ] || $SUDO mkdir -p "$DIR"; fi
  if [ ! -w "$DIR" ]; then
    command -v sudo >/dev/null 2>&1 || die "$DIR is not writable and sudo is not available; use --dir"
    SUDO="sudo"
  fi
  $SUDO cp "$TMP/dockduck" "$DIR/.dockduck-new.$$"
  $SUDO chmod 755 "$DIR/.dockduck-new.$$"
  $SUDO mv -f "$DIR/.dockduck-new.$$" "$TARGET"
  if [ -n "$CURRENT" ]; then say "Updated DockDuck $CURRENT -> $VERSION ($TARGET)"
  else say "Installed DockDuck $VERSION ($TARGET)"; fi
fi

# ---- service (Linux)
if [ "$SERVICE" = 1 ]; then
  [ "$OS" = linux ] || die "--service is only for Linux (systemd)"
  command -v systemctl >/dev/null 2>&1 || die "systemd was not found"
  UNIT=/etc/systemd/system/dockduck.service
  SUDO=""; [ "$(id -u)" = 0 ] || SUDO="sudo"
  id "$RUN_USER" >/dev/null 2>&1 || die "no such user: $RUN_USER"
  cat > "$TMP/dockduck.service" <<UNIT
[Unit]
Description=DockDuck - web GUI for Docker Compose projects
Documentation=https://mnr73.github.io/dock-duck/
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=$RUN_USER
ExecStart=$TARGET --addr 127.0.0.1:9473
Restart=always
RestartSec=2
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
UNIT
  if [ -f "$UNIT" ] && ! $SUDO cmp -s "$TMP/dockduck.service" "$UNIT"; then
    say "Keeping your edited $UNIT (a new default is at $TMP, not used)."
    cp "$TMP/dockduck.service" "$HOME/dockduck.service.new" 2>/dev/null || true
  else
    $SUDO cp "$TMP/dockduck.service" "$UNIT"
  fi
  $SUDO systemctl daemon-reload
  $SUDO systemctl enable dockduck >/dev/null 2>&1 || true
  $SUDO systemctl restart dockduck
  say "Service dockduck is running as $RUN_USER on http://127.0.0.1:9473"
  if ! id -nG "$RUN_USER" | tr ' ' '\n' | grep -qx docker; then
    say "Note: $RUN_USER is not in the 'docker' group, so DockDuck cannot reach Docker: sudo usermod -aG docker $RUN_USER"
  fi
fi

# ---- next steps
case ":$PATH:" in *":$DIR:"*) ;; *) say "Note: $DIR is not in your PATH. Add it, or run $TARGET directly." ;; esac
say ""
say "Next:"
if [ "$SERVICE" = 1 ]; then
  say "  Open http://127.0.0.1:9473  (set a password in Settings, or: dockduck passwd)"
  say "  To reach it from other machines, put it behind a TLS reverse proxy: https://mnr73.github.io/dock-duck/server/"
else
  say "  dockduck                 # starts on http://127.0.0.1:9473"
  say "  dockduck passwd          # set a login password (required to listen on a network address)"
fi
say "  Update later: run this installer again, or use Settings > Updates."
