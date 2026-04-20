import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { BookOpen, Star } from "lucide-react";

const techStack = ["Go 1.24", "gRPC", "Docker", "SQLite", "React 19"];

export function OpenSourceSection() {
  const ref = useScrollAnimation();

  return (
    <section
      id="open-source"
      className="relative py-20 sm:py-28 lg:py-36 border-t border-outline-variant/10 bg-gradient-to-b from-background to-[hsl(var(--surface-container-lowest))]"
      ref={ref}
    >
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <p className="eyebrow fade-in-up">OPEN SOURCE</p>
        <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
          Built in the open.
          <br />
          <span className="text-gradient">Owned by the community.</span>
        </h2>
        <p className="mt-6 text-lg text-muted-foreground leading-relaxed max-w-2xl mx-auto fade-in-up">
          HydraDNS is GPL-3.0. The whole stack is on GitHub, nothing phones home,
          and the roadmap lives in public issues. Fork it, break it, send a PR.
        </p>

        <div className="mt-10 flex flex-wrap gap-4 justify-center fade-in-up">
          <a
            href="https://github.com/hydradns/hydradns"
            target="_blank"
            rel="noopener noreferrer"
            className="btn-primary inline-flex items-center gap-2.5 px-7 py-3.5 rounded-md font-semibold shadow-[0_0_30px_rgba(0,212,170,0.25)] hover:shadow-[0_0_40px_rgba(0,212,170,0.4)] transition-shadow"
          >
            <Star className="h-5 w-5" fill="currentColor" />
            Star on GitHub
          </a>
          <a
            href="https://github.com/hydradns/hydradns/blob/main/CONTRIBUTING.md"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2.5 px-7 py-3.5 rounded-md border border-outline-variant/40 bg-surface-container/30 text-brand-sky font-medium hover:bg-surface-container-high/60 transition-colors"
          >
            <BookOpen className="h-5 w-5" />
            Read Contributing Guide
          </a>
        </div>

        <div className="mt-12 flex flex-wrap gap-2.5 justify-center fade-in-up">
          {techStack.map((tech) => (
            <span
              key={tech}
              className="px-3 py-1 rounded-full border border-outline-variant/30 bg-surface-container/60 font-mono text-xs text-muted-foreground"
            >
              {tech}
            </span>
          ))}
        </div>
      </div>
    </section>
  );
}
