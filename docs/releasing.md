# Cutting a Release

Runbook for tagging the first (and every subsequent) HydraDNS release. As of
this writing the repo has zero tags and zero GitHub Releases.
`.github/workflows/release.yml` has never run. Read it end to end before
tagging; it only triggers on `push: tags: v*` and cannot be dry-run.

This document describes what to do; it does not tag, push, or publish
anything itself.

## What release.yml produces

Two jobs, both gated on a `v*` tag push:

**`build-and-push`**: builds and pushes `linux/amd64` + `linux/arm64` images
via `docker/build-push-action@v6` (QEMU + Buildx), for three services:

| Service | Build context | Image |
|---|---|---|
| core | `apps/core` | `ghcr.io/hydradns/core` |
| ui | `apps/ui` | `ghcr.io/hydradns/ui` |
| hydra-cli | `apps/cli` | `ghcr.io/hydradns/hydra-cli` |

`hydra-cli`'s image additionally gets a `VERSION` build-arg (`steps.meta.outputs.version`,
e.g. `0.1.0` for a `v0.1.0` tag push) stamped into the binary via
`-X github.com/hydradns/hydradns/apps/cli/cmd.Version=...`; `core` and `ui` don't take that arg. Its
default command runs `hydra mcp` (stdio), for listing in the official MCP registry; see
`docs/mcp.md`.

Tags come from `docker/metadata-action@v5` with `type=semver,pattern={{version}}`,
`type=semver,pattern={{major}}.{{minor}}`, and `type=sha`. For a `v0.1.0` tag
push this produces, per image: `0.1.0`, `0.1`, `sha-<7-char-sha>`, **and
`latest`**. `metadata-action`'s default `flavor: latest=auto` adds `latest`
for any `type=semver` rule, including pre-1.0 versions (confirmed against
`docker/metadata-action`'s README; the "major version zero" caveat there only
concerns the bare `{{major}}` pattern, e.g. tag `0`, which this workflow does
not use). So after `v0.1.0`, all of these exist:

```
ghcr.io/hydradns/core:0.1.0
ghcr.io/hydradns/core:0.1
ghcr.io/hydradns/core:sha-xxxxxxx
ghcr.io/hydradns/core:latest
ghcr.io/hydradns/ui:0.1.0
ghcr.io/hydradns/ui:0.1
ghcr.io/hydradns/ui:sha-xxxxxxx
ghcr.io/hydradns/ui:latest
ghcr.io/hydradns/hydra-cli:0.1.0
ghcr.io/hydradns/hydra-cli:0.1
ghcr.io/hydradns/hydra-cli:sha-xxxxxxx
ghcr.io/hydradns/hydra-cli:latest
```

`docker-compose.yml` pulls `ghcr.io/hydradns/<service>:${HYDRA_VERSION:-latest}`,
so a plain `docker compose up -d` after this release tracks `latest` unless
`HYDRA_VERSION` is pinned in `.env`.

**`release-cli`**: cross-compiles the `hydra` CLI for
`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` (`GOWORK=off`, so
the workspace's `go 1.25.4` directive doesn't leak in; the build uses
`apps/cli/go.mod`'s own `go 1.25.0`) and attaches the four binaries to a
GitHub Release for the pushed tag via `softprops/action-gh-release@v3`. That
action creates the release automatically if one doesn't already exist for the
tag; you don't need to create it by hand first.

**`release-cli-checksums`**: runs after all four `release-cli` matrix legs
finish (`needs: release-cli`). It `gh release download`s the four
`hydra-<goos>-<goarch>` binaries just uploaded, runs `sha256sum hydra-* >
checksums.txt` over them, and `gh release upload --clobber`s that file back
onto the same release. This is required, not cosmetic: `hydra update`
(`apps/cli/selfupdate`) refuses to install a binary it cannot verify against a
published checksum, so a release with binaries but no `checksums.txt` makes
`hydra update` fail for everyone on every platform. The feed it reads is
`https://api.github.com/repos/hydradns/hydradns/releases/latest`: this repo,
not the old, now-archived `hydradns/hydra-cli` standalone repo.

### NEXT_PUBLIC_API_URL is baked, but the dashboard resolves the API at runtime

The `ui` image is still built with `NEXT_PUBLIC_API_URL=http://localhost:8080`
(`release.yml`'s `build-args`), and Next.js still inlines `NEXT_PUBLIC_*`
values into the client JS bundle at build time. But the dashboard's runtime
code (`apps/ui/lib/api-base.ts`) treats that specific baked value as a
sentinel: unless the page itself is being viewed on localhost, it's ignored
in favor of deriving the API host from the page's own URL (same protocol and
hostname, port 8080). So the published `ghcr.io/hydradns/ui` image works
correctly over a LAN IP with no rebuild. Opening the dashboard at
`http://192.168.1.53:3000` calls `http://192.168.1.53:8080` automatically.
The corresponding control-plane CORS change is in
`apps/core/cmd/controlplane/middlewares/cors.go`.

`NEXT_PUBLIC_API_URL` is still needed, and still requires a rebuild
(`docker compose build ui`), for two cases: a genuinely custom API address
(e.g. a reverse proxy in front of the API), and a dashboard served over
HTTPS. The derived API URL then defaults to `https://`, but the control
plane has no TLS of its own, so that setup needs a proxy in front of the API
too. See `docs/pi-deployment.md` for both.

### Database migrations run automatically, but the first start after this release can be slow on an upgrade

GORM auto-migrates the schema at startup. Both the controlplane and the dataplane call this
independently against the same SQLite file (they can start at the same time in the combined
`core` container); `busy_timeout=30s` (`internal/storage/db/db.go`) makes them wait for each
other instead of one failing with "database is locked."

This release adds two new indexes: `dns_queries.action` and `blocklist_entries.source_id`. On
a fresh install this is instant: the tables are empty. On an existing installation being
upgraded to this version, `CREATE INDEX` on a large `dns_queries` or `blocklist_entries` table
runs synchronously at startup and holds a write lock for the duration, which can take
noticeably longer than a normal restart on slow storage (an SD card in particular). There is
no separate migration command to run and nothing to configure. Just expect the first start
after the upgrade to take longer than usual on a large, slow-storage install.

## Pre-flight checks (before tagging)

Run these from a clean checkout of `main`, not this worktree, not a stale
clone:

1. `git log --oneline -1`: confirm you're tagging the commit you think you
   are, and that CI (`ci.yml`) is green on it.
2. `git tag -l`: confirm no `v0.1.0` tag already exists locally or on the
   remote (`git ls-remote --tags origin`).
3. Confirm `.github/workflows/release.yml` permissions are intact:
   `build-and-push` needs `packages: write`, `release-cli` and
   `release-cli-checksums` both need `contents: write` (the latter both
   downloads and uploads release assets). (All are already set at time of
   writing; recheck if the workflow has changed.)
4. Confirm the GHCR org (`hydradns`) allows Actions to publish packages: this
   is controlled by the *organization's* Actions package-creation settings,
   not by anything in this repo. If the org has never published a package
   before, check **Organization Settings → Actions → General → Workflow
   permissions**, and that package creation isn't blocked.
5. Sanity-build all three Dockerfiles locally if you have Docker available:
   `docker build apps/core`, `docker build apps/ui --build-arg
   NEXT_PUBLIC_API_URL=http://localhost:8080` (for parity with CI), and
   `docker build apps/cli --build-arg VERSION=test`. This catches Dockerfile
   breakage before CI does, on a tag push you can't easily retry cleanly (see
   Rollback below).
6. Decide the version number. First release is `v0.1.0` (project has no
   prior tags, so this is not "0.0.x" or "1.0.0").
7. Update `CHANGELOG.md`: rename the `## [Unreleased]` heading to
   `## [0.1.0] - YYYY-MM-DD` (today's date, UTC), add a fresh empty
   `## [Unreleased]` section above it, and add the new
   `[0.1.0]: https://github.com/hydradns/hydradns/releases/tag/v0.1.0`
   link reference at the bottom, keeping the existing `[Unreleased]` compare
   link pointed at `main`. Commit this on `main` *before* tagging. The tag
   should point at a commit where the changelog already describes it.

## Cutting the tag

From the prepared commit on `main`:

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

Do not use `git push --tags` (pushes every local tag, not just this one).
Pushing the tag is what triggers `release.yml`. There is no separate
"publish" step.

### Optional: tagging for `go install` / pkg.go.dev / Go Report Card

The three Go modules now declare `github.com/hydradns/hydradns/apps/core`,
`github.com/hydradns/hydradns/apps/cli`, and
`github.com/hydradns/hydradns/apps/scanner` — paths that resolve to this
monorepo instead of the old, now-archived, per-service repos. Nothing above
depends on this: `v0.1.0` alone is sufficient to trigger `release.yml` and
produce the Docker images and cross-compiled CLI binaries.

However, per the Go modules reference
(https://go.dev/ref/mod#vcs-version, "Mapping versions to commits"), a module
that lives in a subdirectory of a repository — not at the repo root — needs
its version tags *prefixed with that subdirectory*, e.g. `apps/cli/v0.1.0`,
not just `v0.1.0`. A plain `v0.1.0` tag versions the (nonexistent) repo-root
module; it does not make the `apps/cli` module resolvable to that version.

So `go install github.com/hydradns/hydradns/apps/cli@latest` and
`go install github.com/hydradns/hydradns/apps/cli@v0.1.0` will not resolve
until a prefixed tag exists. If Go-toolchain installability (and pkg.go.dev /
Go Report Card indexing, which awesome-go requires) matters for a release,
also push, after the `v0.1.0` tag above:

```bash
git tag -a apps/core/v0.1.0 -m "apps/core v0.1.0"
git tag -a apps/cli/v0.1.0 -m "apps/cli v0.1.0"
git tag -a apps/scanner/v0.1.0 -m "apps/scanner v0.1.0"
git push origin apps/core/v0.1.0 apps/cli/v0.1.0 apps/scanner/v0.1.0
```

These tags do not trigger `release.yml` (it only matches `v*`, not
`apps/*/v*`), so they're safe to push independently and don't risk
re-running the Docker/CLI-binary release. `go install
github.com/hydradns/hydradns/apps/cli@v0.1.0` (and `pkg.go.dev` for that
module) starts working once the corresponding prefixed tag is pushed;
`@latest` picks it up on the next `go` module-proxy refresh. This step is
optional and independent of the Docker release above — skip it if Go-toolchain
installability isn't needed for a given release.

## Verifying the release

1. **Workflow ran and succeeded**:
   `gh run list --workflow=release.yml --limit 5`, then
   `gh run watch <run-id>` or check the Actions tab. All three jobs
   (`build-and-push` matrix ×3, `release-cli` matrix ×4,
   `release-cli-checksums` ×1) must be green. `release-cli-checksums` only
   starts once every `release-cli` leg has finished.

2. **GitHub Release exists with 5 assets**:
   `gh release view v0.1.0` should list
   `hydra-linux-amd64`, `hydra-linux-arm64`, `hydra-darwin-amd64`,
   `hydra-darwin-arm64`, and **`checksums.txt`**. The last one is easy to miss
   in a quick glance at the release page but is not optional. See the
   `release-cli-checksums` note above.

3. **Images exist on GHCR**:
   ```bash
   gh api /orgs/hydradns/packages/container/core/versions
   gh api /orgs/hydradns/packages/container/ui/versions
   gh api /orgs/hydradns/packages/container/hydra-cli/versions
   ```
   Look for versions tagged `0.1.0`, `0.1`, `latest`, and a `sha-` tag.

4. **Images are public.** New GHCR packages default to **private**, even
   when pushed from a public repo's workflow. Visibility is not inherited,
   only access permissions are. Anonymous `docker pull` will 401/403 until
   you flip this manually, **once per package** (`hydra-cli` is a brand new
   package the first time this runs and needs the same flip as `core`/`ui`,
   independently):
   - On GitHub: the `hydradns` org's **Packages** tab → click `core` (repeat
     for `ui` and `hydra-cli`) → **Package settings** (top right) → scroll to
     **Danger Zone** → **Change visibility** → **Public** → type the package
     name to confirm.
   - **This is one-way**: GitHub will not let you make a public package
     private again. Don't flip it until you're actually ready to publish.
   - Verify anonymously from a machine with no `docker login` to ghcr.io:
     `docker pull ghcr.io/hydradns/core:0.1.0` (and `.../ui:0.1.0`,
     `.../hydra-cli:0.1.0`) should succeed without credentials once public.

5. **Multi-arch manifest is real**, not just amd64 relabeled:
   ```bash
   docker buildx imagetools inspect ghcr.io/hydradns/core:0.1.0
   docker buildx imagetools inspect ghcr.io/hydradns/ui:0.1.0
   docker buildx imagetools inspect ghcr.io/hydradns/hydra-cli:0.1.0
   ```
   Both `linux/amd64` and `linux/arm64` should be listed for each. This only
   proves the manifest is multi-arch, not that the arm64 image actually
   runs. See next step.

6. **arm64 actually runs, on real arm64 hardware** (a Pi, not `--platform`
   emulation on an amd64 dev box, which can mask a broken arm64 build):
   ```bash
   # on the Pi
   docker pull ghcr.io/hydradns/core:0.1.0
   docker inspect ghcr.io/hydradns/core:0.1.0 --format '{{.Architecture}}'   # expect arm64
   docker run --rm ghcr.io/hydradns/core:0.1.0 /app/controlplane --help 2>&1 | head -5
   ```
   Then actually run the stack there (next section). A binary that starts
   isn't the same as a stack that answers DNS.

   Same idea for `hydra-cli`, plus its stdio contract (nothing but JSON-RPC on stdout):
   ```bash
   docker pull ghcr.io/hydradns/hydra-cli:0.1.0
   docker inspect ghcr.io/hydradns/hydra-cli:0.1.0 --format '{{.Architecture}}'   # expect arm64
   docker run --rm ghcr.io/hydradns/hydra-cli:0.1.0 version   # expect "hydra 0.1.0"
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
     docker run -i --rm -e HYDRA_API_URL=http://localhost:8080 -e HYDRA_TOKEN=x \
       ghcr.io/hydradns/hydra-cli   # stdout: one JSON-RPC line, nothing else
   ```
   Also worth a one-time check before the first MCP-registry publish attempt (see
   `docs/mcp.md` and the launch kit's `mcp-registry/publish-steps.md`): confirm the
   registry's required ownership label made it into the pushed image:
   `docker inspect ghcr.io/hydradns/hydra-cli:0.1.0 --format '{{json .Config.Labels}}'`
   should include `"io.modelcontextprotocol.server.name":"io.github.hydradns/hydra-mcp"`.

7. **The stranger test.** On a machine that has never touched this project
   (a fresh VM or a spare Pi), with a stopwatch running from the first
   command:
   ```bash
   git clone https://github.com/hydradns/hydradns.git && cd hydradns
   docker compose up -d
   dig @localhost doubleclick.net +short     # expect 0.0.0.0 (BLOCK_RESPONSE=zero)
   ```
   Then open `http://localhost:3000` and complete the setup wizard. Record
   the wall-clock time to "dashboard loads and DNS blocks a domain": this
   is the number that matters for the "~5 minute install" claim, not a
   guess. `docker compose up -d` should **pull**, not build, here (verify
   with `docker compose ps` / `docker images` showing pulled images, no
   local build layers); if it builds instead, `HYDRA_VERSION`/image tags
   are wrong or the images aren't public yet.

8. **`hydra update --check` sees the new release.** This is the actual proof
   that self-update works end to end, including the checksums asset from
   `release-cli-checksums`, not just that the binaries exist. On a machine
   with an *older* `hydra` binary installed (built before this tag, or with
   `cmd.Version` stamped to something lower):
   ```bash
   hydra update --check
   ```
   Expect `Update available: <old> -> v0.1.0` followed by
   `Run 'hydra update' to install. (--check: no changes made)` (see
   `apps/cli/cmd/update.go`), and no files touched. If it instead reports
   `is up to date` when it shouldn't, or errors, check in order: the release
   has all 5 assets (previous step), `DefaultFeedURL`
   (`apps/cli/selfupdate/update.go`) still points at
   `api.github.com/repos/hydradns/hydradns/releases/latest`, and (for an
   error mentioning checksums specifically) that `release-cli-checksums`
   actually ran and succeeded rather than being skipped. `--check` never
   downloads or installs anything either way, so it's safe to run against a
   real release before trusting a plain `hydra update`.

## Rollback

If the release is broken (bad image, wrong assets, tagged the wrong commit):

1. **Delete the GitHub Release and tag:**
   ```bash
   gh release delete v0.1.0 --cleanup-tag -y
   ```
   `--cleanup-tag` also removes the underlying git tag on the remote. If you
   tagged locally but the push hasn't gone out yet, `git tag -d v0.1.0` is
   enough and nothing has run.

2. **Remove the bad image versions from GHCR** (tags are not automatically
   cleaned up: a deleted release/tag does not delete the images already
   pushed):
   ```bash
   gh api /orgs/hydradns/packages/container/core/versions | jq '.[] | {id, tags: .metadata.container.tags}'
   gh api -X DELETE /orgs/hydradns/packages/container/core/versions/<version-id>
   ```
   Repeat for `ui` and `hydra-cli`. Do this before re-tagging `v0.1.0`, or the old `latest`
   / `0.1.0` tags may linger alongside (or be silently overwritten by) the
   new push depending on registry caching on client machines that already
   pulled.
   - **Do not** re-flip a package from public back to private as part of
     rollback. GitHub doesn't allow it. If a bad image was already public,
     removing the version is the only lever.

3. **Re-run pre-flight, fix the underlying issue, and re-tag** once the
   broken artifacts are cleaned up.
