# Cutting a Release

Runbook for tagging the first (and every subsequent) HydraDNS release. As of
this writing the repo has zero tags and zero GitHub Releases —
`.github/workflows/release.yml` has never run. Read it end to end before
tagging; it only triggers on `push: tags: v*` and cannot be dry-run.

This document describes what to do; it does not tag, push, or publish
anything itself.

## What release.yml produces

Two jobs, both gated on a `v*` tag push:

**`build-and-push`** — builds and pushes `linux/amd64` + `linux/arm64` images
via `docker/build-push-action@v6` (QEMU + Buildx), for two services:

| Service | Build context | Image |
|---|---|---|
| core | `apps/core` | `ghcr.io/hydradns/core` |
| ui | `apps/ui` | `ghcr.io/hydradns/ui` |

Tags come from `docker/metadata-action@v5` with `type=semver,pattern={{version}}`,
`type=semver,pattern={{major}}.{{minor}}`, and `type=sha`. For a `v0.1.0` tag
push this produces, per image: `0.1.0`, `0.1`, `sha-<7-char-sha>`, **and
`latest`** — `metadata-action`'s default `flavor: latest=auto` adds `latest`
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
```

`docker-compose.yml` pulls `ghcr.io/hydradns/<service>:${HYDRA_VERSION:-latest}`,
so a plain `docker compose up -d` after this release tracks `latest` unless
`HYDRA_VERSION` is pinned in `.env`.

**`release-cli`** — cross-compiles the `hydra` CLI for
`linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` (`GOWORK=off`, so
the workspace's `go 1.25.4` directive doesn't leak in; the build uses
`apps/cli/go.mod`'s own `go 1.25.0`) and attaches the four binaries to a
GitHub Release for the pushed tag via `softprops/action-gh-release@v3`. That
action creates the release automatically if one doesn't already exist for the
tag — you don't need to create it by hand first.

### NEXT_PUBLIC_API_URL is baked, but the dashboard resolves the API at runtime

The `ui` image is still built with `NEXT_PUBLIC_API_URL=http://localhost:8080`
(`release.yml`'s `build-args`), and Next.js still inlines `NEXT_PUBLIC_*`
values into the client JS bundle at build time. But the dashboard's runtime
code (`apps/ui/lib/api-base.ts`) treats that specific baked value as a
sentinel: unless the page itself is being viewed on localhost, it's ignored
in favor of deriving the API host from the page's own URL (same protocol and
hostname, port 8080). So the published `ghcr.io/hydradns/ui` image works
correctly over a LAN IP with no rebuild — opening the dashboard at
`http://192.168.1.53:3000` calls `http://192.168.1.53:8080` automatically.
The corresponding control-plane CORS change is in
`apps/core/cmd/controlplane/middlewares/cors.go`.

`NEXT_PUBLIC_API_URL` is still needed, and still requires a rebuild
(`docker compose build ui`), for two cases: a genuinely custom API address
(e.g. a reverse proxy in front of the API), and a dashboard served over
HTTPS — the derived API URL then defaults to `https://`, but the control
plane has no TLS of its own, so that setup needs a proxy in front of the API
too. See `docs/pi-deployment.md` for both.

## Pre-flight checks (before tagging)

Run these from a clean checkout of `main` — not this worktree, not a stale
clone:

1. `git log --oneline -1` — confirm you're tagging the commit you think you
   are, and that CI (`ci.yml`) is green on it.
2. `git tag -l` — confirm no `v0.1.0` tag already exists locally or on the
   remote (`git ls-remote --tags origin`).
3. Confirm `.github/workflows/release.yml` permissions are intact:
   `build-and-push` needs `packages: write`, `release-cli` needs
   `contents: write`. (Both are already set at time of writing — recheck if
   the workflow has changed.)
4. Confirm the GHCR org (`hydradns`) allows Actions to publish packages: this
   is controlled by the *organization's* Actions package-creation settings,
   not by anything in this repo. If the org has never published a package
   before, check **Organization Settings → Actions → General → Workflow
   permissions**, and that package creation isn't blocked.
5. Sanity-build both Dockerfiles locally if you have Docker available:
   `docker build apps/core` and `docker build apps/ui` (add
   `--build-arg NEXT_PUBLIC_API_URL=http://localhost:8080` for parity with
   CI). This catches Dockerfile breakage before CI does, on a tag push you
   can't easily retry cleanly (see Rollback below).
6. Decide the version number. First release is `v0.1.0` (project has no
   prior tags, so this is not "0.0.x" or "1.0.0").
7. Update `CHANGELOG.md`: rename the `## [Unreleased]` heading to
   `## [0.1.0] - YYYY-MM-DD` (today's date, UTC), add a fresh empty
   `## [Unreleased]` section above it, and add the new
   `[0.1.0]: https://github.com/hydradns/hydradns/releases/tag/v0.1.0`
   link reference at the bottom, keeping the existing `[Unreleased]` compare
   link pointed at `main`. Commit this on `main` *before* tagging — the tag
   should point at a commit where the changelog already describes it.

## Cutting the tag

From the prepared commit on `main`:

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

Do not use `git push --tags` (pushes every local tag, not just this one).
Pushing the tag is what triggers `release.yml` — there is no separate
"publish" step.

## Verifying the release

1. **Workflow ran and succeeded**:
   `gh run list --workflow=release.yml --limit 5`, then
   `gh run watch <run-id>` or check the Actions tab. Both jobs
   (`build-and-push` matrix ×2, `release-cli` matrix ×4) must be green.

2. **GitHub Release exists with 4 assets**:
   `gh release view v0.1.0` should list
   `hydra-linux-amd64`, `hydra-linux-arm64`, `hydra-darwin-amd64`,
   `hydra-darwin-arm64`.

3. **Images exist on GHCR**:
   ```bash
   gh api /orgs/hydradns/packages/container/core/versions
   gh api /orgs/hydradns/packages/container/ui/versions
   ```
   Look for versions tagged `0.1.0`, `0.1`, `latest`, and a `sha-` tag.

4. **Images are public.** New GHCR packages default to **private**, even
   when pushed from a public repo's workflow — visibility is not inherited,
   only access permissions are. Anonymous `docker pull` will 401/403 until
   you flip this manually:
   - On GitHub: the `hydradns` org's **Packages** tab → click `core` (repeat
     for `ui`) → **Package settings** (top right) → scroll to **Danger
     Zone** → **Change visibility** → **Public** → type the package name to
     confirm.
   - **This is one-way**: GitHub will not let you make a public package
     private again. Don't flip it until you're actually ready to publish.
   - Verify anonymously from a machine with no `docker login` to ghcr.io:
     `docker pull ghcr.io/hydradns/core:0.1.0` should succeed without
     credentials once public.

5. **Multi-arch manifest is real**, not just amd64 relabeled:
   ```bash
   docker buildx imagetools inspect ghcr.io/hydradns/core:0.1.0
   docker buildx imagetools inspect ghcr.io/hydradns/ui:0.1.0
   ```
   Both `linux/amd64` and `linux/arm64` should be listed. This only proves
   the manifest is multi-arch, not that the arm64 image actually runs — see
   next step.

6. **arm64 actually runs, on real arm64 hardware** (a Pi, not `--platform`
   emulation on an amd64 dev box, which can mask a broken arm64 build):
   ```bash
   # on the Pi
   docker pull ghcr.io/hydradns/core:0.1.0
   docker inspect ghcr.io/hydradns/core:0.1.0 --format '{{.Architecture}}'   # expect arm64
   docker run --rm ghcr.io/hydradns/core:0.1.0 /app/controlplane --help 2>&1 | head -5
   ```
   Then actually run the stack there (next section) — a binary that starts
   isn't the same as a stack that answers DNS.

7. **The stranger test.** On a machine that has never touched this project
   (a fresh VM or a spare Pi), with a stopwatch running from the first
   command:
   ```bash
   git clone https://github.com/hydradns/hydradns.git && cd hydradns
   docker compose up -d
   dig @localhost doubleclick.net +short     # expect 0.0.0.0 (BLOCK_RESPONSE=zero)
   ```
   Then open `http://localhost:3000` and complete the setup wizard. Record
   the wall-clock time to "dashboard loads and DNS blocks a domain" — this
   is the number that matters for the "~5 minute install" claim, not a
   guess. `docker compose up -d` should **pull**, not build, here (verify
   with `docker compose ps` / `docker images` showing pulled images, no
   local build layers) — if it builds instead, `HYDRA_VERSION`/image tags
   are wrong or the images aren't public yet.

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
   cleaned up — a deleted release/tag does not delete the images already
   pushed):
   ```bash
   gh api /orgs/hydradns/packages/container/core/versions | jq '.[] | {id, tags: .metadata.container.tags}'
   gh api -X DELETE /orgs/hydradns/packages/container/core/versions/<version-id>
   ```
   Repeat for `ui`. Do this before re-tagging `v0.1.0`, or the old `latest`
   / `0.1.0` tags may linger alongside (or be silently overwritten by) the
   new push depending on registry caching on client machines that already
   pulled.
   - **Do not** re-flip a package from public back to private as part of
     rollback — GitHub doesn't allow it. If a bad image was already public,
     removing the version is the only lever.

3. **Re-run pre-flight, fix the underlying issue, and re-tag** once the
   broken artifacts are cleaned up.
