import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Shield, Wifi, CheckCircle2, XCircle } from "lucide-react";

export function ProblemSection() {
  const ref = useScrollAnimation();

  return (
    <section id="why" className="relative py-28 lg:py-36" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid lg:grid-cols-2 gap-16 lg:gap-20 items-center">
          {/* Narrative */}
          <div>
            <p className="eyebrow fade-in-up">WHY HYDRADNS</p>
            <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground leading-[1.1] fade-in-up">
              Your DNS is leaking
              <br />
              <span className="text-gradient">everything.</span>
            </h2>
            <div className="mt-8 space-y-5 text-muted-foreground leading-relaxed text-lg fade-in-up">
              <p>
                Every site you visit, every app you open, every smart thing in your
                house. It all starts with a DNS query. By default those queries go
                unencrypted to your ISP, who can log them, sell them, or hand them over.
              </p>
              <p>
                Ads, trackers, and malware talk to your devices through DNS all day.
                Most people never see any of it.
              </p>
              <p>
                HydraDNS sits between your network and the internet and looks at every
                query on the way out. Bad domains get stopped at the gateway, not on
                the device.
              </p>
            </div>
          </div>

          {/* Flow diagram */}
          <div className="fade-in-up">
            <div className="relative w-full max-w-sm mx-auto">
              {/* Devices */}
              <FlowCard
                icon={<Wifi className="h-7 w-7 text-muted-foreground" strokeWidth={1.75} />}
                title="Your Devices"
                caption="Unfiltered queries"
              />

              <FlowLine />

              {/* HydraDNS */}
              <div className="relative rounded-xl border border-brand-teal/30 bg-surface-container-high glow-primary p-5 text-center">
                <span className="absolute top-3 right-3 px-2 py-0.5 rounded-full border border-outline-variant/40 font-mono text-[10px] tracking-wider text-brand-teal">
                  FILTER
                </span>
                <Shield className="h-10 w-10 mx-auto text-brand-teal mb-2" strokeWidth={2} />
                <p className="font-headline text-base font-bold text-foreground">HydraDNS</p>
                <p className="mt-1 font-mono text-[11px] text-brand-teal/90">
                  Inspect · Score · Decide
                </p>
              </div>

              {/* Branch connectors */}
              <div className="relative h-10 flex justify-between">
                <div className="absolute top-0 left-1/2 w-1/2 h-full border-t border-r border-brand-teal/40 rounded-tr-lg -translate-x-1/2" />
                <div className="absolute top-0 left-0 w-1/2 h-full border-t border-l border-brand-teal/40 rounded-tl-lg translate-x-1/2" />
              </div>

              {/* Outcomes */}
              <div className="grid grid-cols-2 gap-4">
                <OutcomeCard
                  icon={<CheckCircle2 className="h-5 w-5 text-brand-green" />}
                  title="Allowed"
                  caption="Clean traffic"
                  accent="green"
                />
                <OutcomeCard
                  icon={<XCircle className="h-5 w-5 text-brand-red" />}
                  title="Blocked"
                  caption="Threat stopped"
                  accent="red"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function FlowCard({
  icon,
  title,
  caption,
}: {
  icon: React.ReactNode;
  title: string;
  caption: string;
}) {
  return (
    <div className="rounded-xl border border-outline-variant/30 bg-surface-container p-5 text-center">
      <div className="flex justify-center mb-2">{icon}</div>
      <p className="font-headline text-base font-bold text-foreground">{title}</p>
      <p className="mt-1 font-mono text-[11px] text-muted-foreground">{caption}</p>
    </div>
  );
}

function FlowLine() {
  return (
    <div className="flex justify-center my-4">
      <div className="w-px h-8 bg-gradient-to-b from-outline-variant/10 via-brand-teal/40 to-brand-teal/80" />
    </div>
  );
}

function OutcomeCard({
  icon,
  title,
  caption,
  accent,
}: {
  icon: React.ReactNode;
  title: string;
  caption: string;
  accent: "green" | "red";
}) {
  const border = accent === "green" ? "border-brand-green/30" : "border-brand-red/30";
  const text = accent === "green" ? "text-brand-green" : "text-brand-red";
  return (
    <div
      className={`rounded-xl border ${border} bg-surface-container-lowest p-4 text-center`}
    >
      <div className="flex justify-center mb-1.5">{icon}</div>
      <p className={`font-headline text-sm font-bold ${text}`}>{title}</p>
      <p className="mt-0.5 font-mono text-[10px] text-muted-foreground">{caption}</p>
    </div>
  );
}
