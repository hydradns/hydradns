import { useScrollAnimation } from "@/hooks/useScrollAnimation";

const code = `# Clone and launch
git clone https://github.com/hydradns/hydra-core.git
cd HydraDNS
docker-compose up --build

# Test it
dig @localhost -p 1053 example.com`;

export function QuickStartSection() {
  const ref = useScrollAnimation();

  return (
    <section className="py-24 lg:py-32" ref={ref}>
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-12">
          <p className="text-sm font-semibold uppercase tracking-widest text-accent mb-4 fade-in-up">Quick Start</p>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground fade-in-up">
            Up and running in <span className="text-gradient">30 seconds.</span>
          </h2>
        </div>

        <div className="glass rounded-2xl overflow-hidden glow-primary fade-in-up">
          {/* macOS title bar */}
          <div className="flex items-center gap-2 px-4 py-3 bg-muted/30 border-b border-border/50">
            <div className="w-3 h-3 rounded-full bg-destructive/80" />
            <div className="w-3 h-3 rounded-full bg-[hsl(45,93%,47%)]/80" />
            <div className="w-3 h-3 rounded-full bg-[hsl(142,71%,45%)]/80" />
            <span className="ml-2 text-xs text-muted-foreground">Terminal</span>
          </div>
          <pre className="p-6 text-sm leading-relaxed overflow-x-auto">
            <code className="text-muted-foreground">
              {code.split("\n").map((line, i) => (
                <span key={i} className="block">
                  {line.startsWith("#") ? (
                    <span className="text-muted-foreground/60">{line}</span>
                  ) : line.startsWith("git") || line.startsWith("cd") || line.startsWith("docker") || line.startsWith("dig") ? (
                    <>
                      <span className="text-accent">$ </span>
                      <span className="text-foreground">{line}</span>
                    </>
                  ) : (
                    <span>{line}</span>
                  )}
                </span>
              ))}
            </code>
          </pre>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground fade-in-up">
          That's it. No config files to edit, no dependencies to install.
        </p>
      </div>
    </section>
  );
}
