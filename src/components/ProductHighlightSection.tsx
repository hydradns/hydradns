import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Terminal, ListFilter, FileText } from "lucide-react";
import overview from "@/assets/dashboard/overview.png";
import queryLogs from "@/assets/dashboard/query-logs.png";
import policies from "@/assets/dashboard/policies.png";


type ChapterLink = {
  label: string;
  icon: typeof Terminal;
  href: string;
};

const chapters: ChapterLink[] = [
  { label: "Query Logs", icon: Terminal, href: "#" },
  { label: "Policies", icon: ListFilter, href: "#" },
  { label: "Blocklists", icon: FileText, href: "#" },
];

export function ProductHighlightSection() {
  const ref = useScrollAnimation();

  return (
    <section id="highlights" className="relative py-28 lg:py-36 overflow-hidden" ref={ref}>
      {/* Ambient glow behind the stack */}
      <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/3 w-[70%] max-w-[900px] aspect-square rounded-full bg-brand-teal/[0.06] blur-3xl pointer-events-none" />
      <div className="absolute inset-x-0 top-1/2 -translate-y-1/2 h-px bg-gradient-to-r from-transparent via-outline-variant/30 to-transparent pointer-events-none" />

      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="text-center mb-14">
          <p className="eyebrow fade-in-up">SEE IT IN ACTION</p>
          <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
            Not a mock.
            <br />
            <span className="text-gradient">A real dashboard.</span>
          </h2>
          <p className="mt-5 text-lg text-muted-foreground max-w-2xl mx-auto fade-in-up">
            Every chart, table, and toggle on this page ships in the product. This is
            what you get the moment <code className="font-mono text-brand-sky">docker compose up</code> finishes.
          </p>
        </div>

        {/* Stacked screenshots */}
        <div className="relative fade-in-up">
          {/* Desktop stack: three layered windows */}
          <div className="relative mx-auto max-w-5xl hidden lg:block" style={{ aspectRatio: "16 / 11" }}>
            <BackCard
              image={queryLogs}
              alt="HydraDNS Query Logs view"
              position="left"
            />
            <BackCard
              image={policies}
              alt="HydraDNS Policies view"
              position="right"
            />
            <FrontCard image={overview} alt="HydraDNS Dashboard overview" />
          </div>

          {/* Tablet/mobile: stack vertically */}
          <div className="lg:hidden space-y-6">
            <MobileCard image={overview} alt="Dashboard overview" featured />
            <div className="grid sm:grid-cols-2 gap-6">
              <MobileCard image={queryLogs} alt="Query logs" />
              <MobileCard image={policies} alt="Policies" />
            </div>
          </div>
        </div>

        {/* Chapter links */}
        <div className="mt-16 flex flex-wrap gap-3 justify-center fade-in-up">
          {chapters.map((c) => (
            <a
              key={c.label}
              href={c.href}
              className="group inline-flex items-center gap-2 px-4 py-2 rounded-full border border-outline-variant/40 bg-surface-container/50 hover:border-brand-teal/50 hover:bg-surface-container-high/60 transition-colors"
            >
              <c.icon className="h-3.5 w-3.5 text-muted-foreground group-hover:text-brand-teal transition-colors" />
              <span className="font-mono text-xs text-muted-foreground group-hover:text-foreground transition-colors">
                {c.label}
              </span>
              <span className="text-muted-foreground/60 group-hover:text-brand-teal transition-colors text-xs">
                ↗
              </span>
            </a>
          ))}
        </div>
      </div>
    </section>
  );
}

function FrontCard({ image, alt }: { image: string; alt: string }) {
  return (
    <div
      className="absolute left-1/2 top-[8%] -translate-x-1/2 w-[78%] rounded-xl overflow-hidden border border-outline-variant/40 bg-surface-container shadow-[0_30px_80px_rgba(0,0,0,0.55)] z-30"
      style={{
        boxShadow:
          "0 30px 80px rgba(0,0,0,0.55), 0 0 60px rgba(0,212,170,0.18)",
      }}
    >
      <WindowChrome title="Dashboard · Overview" live />
      <img src={image} alt={alt} className="block w-full" loading="lazy" />
    </div>
  );
}

function BackCard({
  image,
  alt,
  position,
}: {
  image: string;
  alt: string;
  position: "left" | "right";
}) {
  const transform =
    position === "left"
      ? "translate(-22%, 4%) rotate(-5deg) scale(0.86)"
      : "translate(22%, 4%) rotate(5deg) scale(0.86)";

  return (
    <div
      className="absolute top-0 w-[70%] left-1/2 -translate-x-1/2 rounded-xl overflow-hidden border border-outline-variant/40 bg-surface-container shadow-[0_20px_60px_rgba(0,0,0,0.45)] opacity-70 z-10"
      style={{ transform, transformOrigin: "center top" }}
    >
      <WindowChrome title={position === "left" ? "Query Logs" : "Policies"} />
      <img src={image} alt={alt} className="block w-full opacity-90" loading="lazy" />
    </div>
  );
}

function MobileCard({
  image,
  alt,
  featured,
}: {
  image: string;
  alt: string;
  featured?: boolean;
}) {
  return (
    <div
      className={`rounded-xl overflow-hidden border border-outline-variant/40 bg-surface-container ${
        featured ? "glow-primary" : ""
      }`}
    >
      <WindowChrome title={featured ? "Dashboard · Overview" : alt} live={featured} />
      <img src={image} alt={alt} className="block w-full" loading="lazy" />
    </div>
  );
}

function WindowChrome({ title, live }: { title: string; live?: boolean }) {
  return (
    <div className="flex items-center justify-between h-9 px-4 bg-surface-container-highest border-b border-outline-variant/20">
      <div className="flex items-center gap-1.5">
        <span className="w-2.5 h-2.5 rounded-full bg-surface-variant/80 border border-outline-variant/30" />
        <span className="w-2.5 h-2.5 rounded-full bg-surface-variant/80 border border-outline-variant/30" />
        <span className="w-2.5 h-2.5 rounded-full bg-surface-variant/80 border border-outline-variant/30" />
      </div>
      <span className="font-mono text-[10px] text-muted-foreground tracking-wide truncate max-w-[50%]">
        {title}
      </span>
      {live ? (
        <div className="flex items-center gap-1.5 px-2 py-0.5 rounded border border-outline-variant/30 bg-surface-container-lowest">
          <span className="relative flex h-1.5 w-1.5">
            <span className="absolute inline-flex h-full w-full rounded-full bg-brand-teal opacity-75 animate-ping" />
            <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-brand-teal" />
          </span>
          <span className="font-mono text-[9px] font-semibold tracking-wider text-brand-teal">
            LIVE
          </span>
        </div>
      ) : (
        <span className="w-10" aria-hidden />
      )}
    </div>
  );
}

