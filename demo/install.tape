# HydraDNS demo: install to first block, one loop
#
# Prerequisites to actually render this (not needed just to read it):
#   - vhs, ttyd, ffmpeg (https://github.com/charmbracelet/vhs)  ->  vhs demo/install.tape
#   - A clean host with git, Docker + Compose v2, and a Go toolchain matching
#     apps/cli/go.mod (no tagged release exists yet, so the `hydra` CLI must
#     be built locally; there is no prebuilt binary to download until
#     v0.1.0 ships GitHub Release assets per .github/workflows/release.yml).
#   - Port 53 free on the host (see README's "Port 53 already in use?" note).
#   - Default docker-compose.yml env: no CORS_ORIGINS/BLOCK_RESPONSE override,
#     so the dataplane answers blocked queries with A 0.0.0.0 (the "zero"
#     default; see apps/core/internal/dnsengine/engine.go respondBlocked).
#
# Keep output under 2 MB or it will not load on mobile previews.

Output demo/demo.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set TypingSpeed 40ms
Set Theme "Catppuccin Mocha"

Type "git clone https://github.com/hydradns/hydradns.git && cd hydradns"
Enter
Sleep 2s

Type "docker compose up -d"
Enter
Sleep 4s

# doubleclick.net is blocked out of the box by the "block-ads" policy shipped
# in apps/core/configs/policies.json; no setup wizard needed for this part.
Type "dig @localhost doubleclick.net +short"
Enter
Sleep 2s
Type "# 0.0.0.0 - blocked (BLOCK_RESPONSE=zero, the default; not REFUSED)"
Enter
Sleep 1.5s

# The CLI has no prebuilt binary pre-release, so build it first.
Type "cd apps/cli && go build -o hydra . && cd ../.."
Enter
Sleep 3s

# hydra block calls the control-plane API, which requires a bearer token.
# First boot has no admin account yet, so mint one with `hydra setup`
# (equivalent to the dashboard's first-run setup wizard) before blocking.
# The password prompt disables terminal echo, so nothing typed below
# appears in the recording. This is a real interactive TTY under vhs,
# not a piped/non-interactive run.
Type "./apps/cli/hydra setup"
Enter
Sleep 1s
Type "changeme12345"
Enter
Sleep 500ms
Type "changeme12345"
Enter
Sleep 2s

Type "./apps/cli/hydra block tiktok.com"
Enter
Sleep 2s

Type "dig @localhost tiktok.com +short"
Enter
Sleep 2s
Type "# 0.0.0.0 - blocked"
Enter
Sleep 1.5s
