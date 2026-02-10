import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Github, BookOpen } from "lucide-react";

export function OpenSourceSection() {
  const ref = useScrollAnimation();

  return (
    <section id="open-source" className="py-24 lg:py-32" ref={ref}>
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <p className="text-sm font-semibold uppercase tracking-widest text-accent mb-4 fade-in-up">Open Source</p>
        <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground fade-in-up">
          Built in the open.<br />
          <span className="text-gradient">Owned by the community.</span>
        </h2>
        <p className="mt-6 text-lg text-muted-foreground leading-relaxed max-w-xl mx-auto fade-in-up">
          HydraDNS is free and open-source under GPL-3.0. No telemetry, no tracking, no vendor lock-in. Your DNS, your rules. We welcome contributors of all experience levels.
        </p>
        <div className="mt-8 flex flex-wrap gap-4 justify-center fade-in-up">
          <a
            href="https://github.com/lopster568/HydraDNS"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors glow-primary"
          >
            <Github className="h-5 w-5" />
            Star on GitHub
          </a>
          <a
            href="https://github.com/lopster568/HydraDNS/blob/main/CONTRIBUTING.md"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-lg border border-border text-foreground font-medium hover:bg-muted/50 transition-colors"
          >
            <BookOpen className="h-5 w-5" />
            Read Contributing Guide
          </a>
        </div>
      </div>
    </section>
  );
}
