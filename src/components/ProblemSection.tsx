import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Shield, Wifi, Globe, XCircle } from "lucide-react";

export function ProblemSection() {
  const ref = useScrollAnimation();

  return (
    <section id="why" className="py-24 lg:py-32" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid lg:grid-cols-2 gap-16 items-center">
          <div>
            <p className="text-sm font-semibold uppercase tracking-widest text-accent mb-4 fade-in-up">
              Why HydraDNS
            </p>
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground leading-tight fade-in-up">
              Your DNS is leaking<br />
              <span className="text-gradient">everything.</span>
            </h2>
            <div className="mt-8 space-y-4 text-muted-foreground leading-relaxed fade-in-up">
              <p>
                Every website you visit, every app you open, every smart device in your home — they all start with a DNS query. And by default, those queries travel unencrypted to your ISP, who can see, log, and sell your browsing history.
              </p>
              <p>
                Ads, trackers, and malware domains connect to your devices thousands of times a day — all through DNS. Most people have zero visibility into it.
              </p>
              <p>
                HydraDNS sits between your network and the internet, intercepting every DNS query. Threats are blocked before a single byte of malicious data reaches your devices.
              </p>
            </div>
          </div>

          {/* Diagram */}
          <div className="flex justify-center fade-in-up">
            <div className="relative w-full max-w-sm">
              {/* Device */}
              <div className="glass rounded-xl p-4 text-center mb-6">
                <Wifi className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                <p className="text-sm font-medium text-foreground">Your Devices</p>
                <p className="text-xs text-muted-foreground">DNS Query</p>
              </div>

              {/* Arrow down */}
              <div className="flex justify-center mb-6">
                <div className="w-px h-8 bg-gradient-to-b from-muted-foreground/50 to-primary" />
              </div>

              {/* HydraDNS Shield */}
              <div className="glass glow-primary rounded-xl p-4 text-center mb-6 border-primary/30">
                <Shield className="h-10 w-10 mx-auto text-primary mb-2" />
                <p className="text-sm font-bold text-foreground">HydraDNS</p>
                <p className="text-xs text-muted-foreground">Inspect & Filter</p>
              </div>

              {/* Two paths */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="flex justify-center mb-3">
                    <div className="w-px h-6 bg-gradient-to-b from-primary to-[hsl(142,71%,45%)]" />
                  </div>
                  <div className="glass rounded-xl p-3 text-center border-[hsl(142,71%,45%)]/30">
                    <Globe className="h-6 w-6 mx-auto text-[hsl(142,71%,45%)] mb-1" />
                    <p className="text-xs font-medium text-[hsl(142,71%,45%)]">Clean</p>
                    <p className="text-[10px] text-muted-foreground">Allowed</p>
                  </div>
                </div>
                <div>
                  <div className="flex justify-center mb-3">
                    <div className="w-px h-6 bg-gradient-to-b from-primary to-destructive border-dashed" />
                  </div>
                  <div className="glass rounded-xl p-3 text-center border-destructive/30">
                    <XCircle className="h-6 w-6 mx-auto text-destructive mb-1" />
                    <p className="text-xs font-medium text-destructive">Blocked</p>
                    <p className="text-[10px] text-muted-foreground">Threat stopped</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
