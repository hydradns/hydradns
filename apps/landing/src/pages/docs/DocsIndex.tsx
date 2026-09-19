import { Link } from "react-router-dom";
import { ArrowRight, BookOpen } from "lucide-react";
import { DocsNavbar } from "@/components/docs/DocsNavbar";
import { docsSections } from "./nav";

export default function DocsIndex() {
  return (
    <div className="min-h-screen bg-background relative">
      <div className="hero-ambient fixed inset-0 pointer-events-none -z-0" />
      <DocsNavbar />

      {/* Hero */}
      <section className="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pt-16 sm:pt-24 pb-14 text-center">
        <div className="inline-flex items-center gap-2 rounded-full border border-outline-variant/40 bg-surface-container/60 px-4 py-1.5 mb-6 backdrop-blur-sm">
          <BookOpen className="h-3.5 w-3.5 text-brand-teal-bright" />
          <span className="eyebrow !mb-0">Documentation</span>
        </div>

        <h1 className="font-headline text-5xl sm:text-6xl lg:text-7xl font-bold tracking-tighter mb-6">
          <span className="text-gradient-long">Everything you need</span>
          <br />
          <span className="text-foreground">to run HydraDNS.</span>
        </h1>

        <p className="text-lg sm:text-xl text-muted-foreground max-w-2xl mx-auto leading-relaxed">
          Bring up a private DNS firewall in 60 seconds, point your router at it,
          and shape exactly what your network can and can't reach.
        </p>

        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <Link
            to="/docs/getting-started"
            className="btn-primary inline-flex items-center gap-2 px-6 py-3 rounded-md text-sm font-semibold shadow-[0_0_30px_rgba(0,212,170,0.3)] hover:shadow-[0_0_40px_rgba(0,212,170,0.45)] transition-shadow"
          >
            Start here
            <ArrowRight className="h-4 w-4" />
          </Link>
          <Link
            to="/docs/dashboard"
            className="inline-flex items-center gap-2 px-5 py-3 rounded-md border border-outline-variant/50 bg-surface-container/50 backdrop-blur-sm text-sm font-medium text-foreground hover:bg-surface-container hover:border-brand-teal/40 transition-colors"
          >
            Tour the dashboard
          </Link>
        </div>
      </section>

      {/* Section grid */}
      <section className="relative max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pb-24">
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {docsSections.map((section) => {
            const Icon = section.icon;
            return (
              <Link
                key={section.slug}
                to={`/docs/${section.slug}`}
                className="group relative rounded-2xl border border-outline-variant/40 bg-surface-container/40 backdrop-blur-sm p-6 transition-all hover:bg-surface-container hover:border-brand-teal/40 hover:shadow-[0_0_40px_-10px_rgba(0,212,170,0.35)]"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="h-11 w-11 rounded-xl bg-surface-container-high border border-outline-variant/40 flex items-center justify-center text-brand-teal-bright group-hover:border-brand-teal/60 group-hover:text-brand-teal transition-colors">
                    <Icon className="h-5 w-5" />
                  </div>
                  <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-brand-teal group-hover:translate-x-1 transition-all" />
                </div>

                <p className="eyebrow mb-2">{section.eyebrow}</p>
                <h2 className="font-headline text-xl font-bold text-foreground mb-2 tracking-tight group-hover:text-brand-teal-bright transition-colors">
                  {section.title}
                </h2>
                <p className="text-sm text-muted-foreground leading-relaxed">
                  {section.description}
                </p>
              </Link>
            );
          })}
        </div>
      </section>

      <footer className="relative border-t border-outline-variant/30 py-8">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-muted-foreground">
          <p>© {new Date().getFullYear()} HydraDNS · GPL-3.0</p>
          <div className="flex items-center gap-5">
            <a
              href="https://github.com/hydradns/hydradns"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-foreground transition-colors"
            >
              GitHub
            </a>
            <Link to="/" className="hover:text-foreground transition-colors">
              Home
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
