#!/usr/bin/env sh
# Cross-compiles DockDuck for every supported platform into dist/ as
# dockduck_<version>_<os>_<arch>.tar.gz plus checksums.txt.
#
#   scripts/build-release.sh v0.1.0        (the web UI must already be built and embedded: `make web`)
#
# CGO is off, so the binaries are static and need nothing on the target.
set -eu

VERSION="${1:?usage: build-release.sh vX.Y.Z}"
case "$VERSION" in v[0-9]*.[0-9]*.[0-9]*) ;; *) echo "version must look like v1.2.3, got: $VERSION" >&2; exit 2 ;; esac

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist"
rm -rf "$OUT" && mkdir -p "$OUT"

if [ ! -f "$ROOT/apps/server/internal/webui/dist/index.html" ]; then
  echo "the web UI is not embedded yet: run 'make web' first" >&2
  exit 1
fi

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os="${target%/*}"; arch="${target#*/}"
  name="dockduck_${VERSION#v}_${os}_${arch}"
  stage="$OUT/$name"
  mkdir -p "$stage"
  ( cd "$ROOT/apps/server" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
      go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$stage/dockduck" ./cmd/dockduck )
  cp "$ROOT/LICENSE" "$ROOT/README.md" "$stage/"
  # the archive holds the binary at its root: the updater reads exactly `dockduck`
  tar -C "$stage" -czf "$OUT/$name.tar.gz" dockduck LICENSE README.md
  rm -rf "$stage"
  echo "built $name.tar.gz"
done

( cd "$OUT" && if command -v sha256sum >/dev/null 2>&1; then sha256sum ./*.tar.gz; else shasum -a 256 ./*.tar.gz; fi \
    | sed 's#  \./#  #' > checksums.txt )
cat "$OUT/checksums.txt"
