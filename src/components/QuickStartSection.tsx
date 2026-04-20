import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { useScrollAnimation } from "@/hooks/useScrollAnimation";

type Line =
  | { kind: "comment"; text: string }
  | { kind: "cmd"; text: string }
  | { kind: "blank" };

const lines: Line[] = [
  { kind: "comment", text: "# Clone and launch" },
  { kind: "cmd", text: "git clone --recursive https://github.com/hydradns/hydradns.git" },
  { kind: "cmd", text: "cd hydradns" },
  { kind: "cmd", text: "docker compose up -d" },
  { kind: "blank" },
  { kind: "comment", text: "# Test that DNS is responding" },
  { kind: "cmd", text: "dig @localhost -p 1053 example.com" },
  { kind: "blank" },
  { kind: "comment", text: "# Open the dashboard" },
  { kind: "cmd", text: "open http://localhost:3000" },
];

const plain = lines
  .map((l) => (l.kind === "blank" ? "" : l.text))
  .join("\n");

export function QuickStartSection() {
  const ref = useScrollAnimation();
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(plain);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      /* ignore clipboard errors */
    }
  };

  return (
    <section className="relative py-28 lg:py-36" ref={ref}>
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-12">
          <p className="eyebrow fade-in-up">QUICK START</p>
          <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
            Up and running in <span className="text-gradient">30 seconds.</span>
          </h2>
        </div>

        <div className="relative rounded-xl overflow-hidden bg-surface-container-high border border-outline-variant/30 shadow-[0_20px_60px_rgba(0,0,0,0.4)] glow-primary fade-in-up">
          {/* Terminal chrome */}
          <div className="flex items-center justify-between h-11 px-4 bg-surface-container border-b border-outline-variant/30">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-brand-red/80" />
              <span className="w-3 h-3 rounded-full bg-brand-amber/80" />
              <span className="w-3 h-3 rounded-full bg-brand-green/80" />
            </div>
            <span className="font-mono text-xs text-muted-foreground">zsh</span>
            <button
              onClick={copy}
              aria-label="Copy commands"
              className="flex items-center justify-center h-7 w-7 rounded text-muted-foreground hover:text-brand-teal hover:bg-surface-container-lowest transition-colors"
            >
              {copied ? (
                <Check className="h-4 w-4 text-brand-teal" />
              ) : (
                <Copy className="h-3.5 w-3.5" />
              )}
            </button>
          </div>

          {/* Terminal body */}
          <div className="p-6 bg-surface-container-lowest">
            <pre className="font-mono text-sm leading-relaxed overflow-x-auto">
              <code>
                {lines.map((l, i) => {
                  if (l.kind === "blank") {
                    return (
                      <span key={i} className="block h-3.5" aria-hidden>
                        {"\u00a0"}
                      </span>
                    );
                  }
                  if (l.kind === "comment") {
                    return (
                      <span key={i} className="block text-muted-foreground/60">
                        {l.text}
                      </span>
                    );
                  }
                  return (
                    <span key={i} className="block">
                      <span className="text-brand-teal">$ </span>
                      <span className="text-foreground/90">{l.text}</span>
                    </span>
                  );
                })}
              </code>
            </pre>
          </div>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground fade-in-up">
          No config files to edit. No dependencies to install. That's the whole setup.
        </p>
      </div>
    </section>
  );
}
