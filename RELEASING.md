# Releasing

DockDuck follows [SemVer](https://semver.org): `MAJOR.MINOR.PATCH`. First release: **v0.1.0**.

- **patch**: bug fixes only.
- **minor**: new features, nothing breaks.
- **major**: a break in the config file, the HTTP API or the command line.

The version is not written in any file. It comes from the git tag, and `scripts/build-release.sh` puts it into the binary (`-X main.version=vX.Y.Z`). So a release is just a tag.

## Cut a release

```sh
git status                       # clean, on main
make lint test                   # the same gate CI runs
git tag v0.1.0
git push origin main v0.1.0      # push the tag by name
```

Pushing the tag runs `.github/workflows/release.yml`:

1. lint and tests,
2. builds the web UI and embeds it,
3. `scripts/build-release.sh` makes `dockduck_<version>_<os>_<arch>.tar.gz` for linux and darwin, amd64 and arm64, plus `checksums.txt`,
4. publishes one GitHub release for the tag.

A tag with a `-` (`v0.2.0-rc1`) becomes a **prerelease**. The installer and the in-app updater ignore prereleases, so use one to try a build first.

## Check it worked

```sh
curl -fsSL https://api.github.com/repos/mnr73/dock-duck/releases/latest | grep tag_name
curl -fsSL https://mnr73.github.io/dock-duck/install.sh | sh
dockduck version                 # v0.1.0
```

## First time only

1. **Pages:** repo Settings → Pages → Source: **GitHub Actions**. The workflow `.github/workflows/docs.yml` then publishes `docs/` to https://mnr73.github.io/dock-duck/ on every push to `main` that touches `docs/`, `install.sh` or the workflow. The site also serves `install.sh`.
2. **Actions permissions:** Settings → Actions → General → Workflow permissions → **Read and write** (the release job creates the release with the built-in token; the workflow also asks for `contents: write`).
3. Push the commits first and wait for the `docs` run to finish **before** the first tag, so the install command already works when the release appears.

## Things to know

- A published tag should not be moved. If a release is wrong, publish the next patch. To undo a bad one: delete the release and the tag (`git push origin :refs/tags/v0.1.0`) **before** anyone installs it.
- The installer and the updater verify each archive against `checksums.txt` of the same release, so a release with no `checksums.txt` cannot be installed. Do not edit assets by hand after the workflow runs.
- Builds are not code-signed or notarised. macOS may warn on the first launch of a binary that was downloaded by a browser. The installer uses `curl`, which does not trigger that.
- Under systemd the unit needs `Restart=always`, or an in-app update leaves the service stopped (see the [docs](https://mnr73.github.io/dock-duck/server/systemd)).
- Rehearse locally without GitHub: `make web && scripts/build-release.sh v0.0.1`, then serve `dist/` with any fake GitHub as the repository's tests do. `DOCKDUCK_UPDATE_API` points a server at it.
- If the repo is renamed or moved, change `Repo` in `apps/server/internal/updater/updater.go`, `REPO` in `install.sh`, `base` in `docs/.vitepress/config.mts` and the URLs in the docs and README.

## Next releases

```sh
git tag v0.1.1
git push origin main v0.1.1
```

Users get it by running the install command again, `dockduck update`, or **Settings → Updates**.
