# HydraDNS demo: install to first block, one loop
# Render: vhs demo/install.tape  (needs vhs + ttyd + ffmpeg)
# Keep output under 2 MB or it will not load on mobile previews.

Output demo/demo.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set TypingSpeed 40ms
Set Theme "Catppuccin Mocha"

Type "git clone --recursive https://github.com/hydradns/hydradns.git && cd hydradns"
Enter
Sleep 2s

Type "docker compose up -d"
Enter
Sleep 4s

Type "dig @localhost doubleclick.net +short"
Enter
Sleep 2s
Type "# REFUSED. Blocked by HydraDNS."
Enter
Sleep 1.5s

Type "./apps/cli/hydra block tiktok.com"
Enter
Sleep 2s

Type "dig @localhost tiktok.com +short"
Enter
Sleep 3s
