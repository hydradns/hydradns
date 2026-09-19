import type { ReactNode } from "react";
import { useEffect } from "react";
import { useLocation, Link } from "react-router-dom";
import { ArrowLeft, ArrowRight } from "lucide-react";
import { DocsNavbar } from "./DocsNavbar";
import { DocsSidebar } from "./DocsSidebar";
import { DocsToc, type TocItem } from "./DocsToc";
import { Prose } from "./Prose";
import { findAdjacent, findSection } from "@/pages/docs/nav";

type Props = {
  slug?: string;
  title: string;
  eyebrow?: string;
  description?: string;
  toc?: TocItem[];
  children: ReactNode;
};

export function DocsLayout({
  slug,
  title,
  eyebrow,
  description,
  toc = [],
  children,
}: Props) {
  const { pathname } = useLocation();

  useEffect(() => {
    if (!window.location.hash) window.scrollTo({ top: 0, behavior: "auto" });
  }, [pathname]);

  const { prev, next } = slug ? findAdjacent(slug) : {};
  const section = slug ? findSection(slug) : undefined;

  return (
    <div className="min-h-screen bg-background relative">
      <div className="hero-ambient fixed inset-0 pointer-events-none -z-0" />
      <DocsNavbar />

      <div className="relative max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 flex gap-10">
        <DocsSidebar />

        <main className="flex-1 min-w-0 py-10 lg:py-14 max-w-3xl">
          {/* Breadcrumbs */}
          <div className="mb-6 flex items-center gap-2 text-[11px] font-mono tracking-wider uppercase text-muted-foreground">
            <Link to="/docs" className="hover:text-brand-teal transition-colors">
              Docs
            </Link>
            {section && (
              <>
                <span className="text-outline-variant">/</span>
                <span className="text-foreground/70">{section.title}</span>
              </>
            )}
          </div>

          {/* Header */}
          <header className="mb-10 pb-8 border-b border-outline-variant/30">
            {eyebrow && <p className="eyebrow mb-3">{eyebrow}</p>}
            <h1 className="font-headline text-4xl sm:text-5xl font-bold tracking-tighter text-foreground mb-4">
              {title}
            </h1>
            {description && (
              <p className="text-lg text-muted-foreground leading-relaxed max-w-2xl">
                {description}
              </p>
            )}
          </header>

          {/* Content */}
          <Prose>{children}</Prose>

          {/* Prev / Next pager */}
          {(prev || next) && (
            <nav className="mt-16 pt-8 border-t border-outline-variant/30 grid grid-cols-1 sm:grid-cols-2 gap-4">
              {prev ? (
                <Link
                  to={`/docs/${prev.slug}`}
                  className="group rounded-xl border border-outline-variant/40 bg-surface-container/40 hover:bg-surface-container hover:border-brand-teal/40 p-4 transition-all"
                >
                  <p className="text-xs uppercase font-mono tracking-wider text-muted-foreground mb-1 flex items-center gap-1">
                    <ArrowLeft className="h-3 w-3" /> Previous
                  </p>
                  <p className="font-headline font-semibold text-foreground group-hover:text-brand-teal-bright transition-colors">
                    {prev.title}
                  </p>
                </Link>
              ) : (
                <span />
              )}
              {next ? (
                <Link
                  to={`/docs/${next.slug}`}
                  className="group rounded-xl border border-outline-variant/40 bg-surface-container/40 hover:bg-surface-container hover:border-brand-teal/40 p-4 transition-all text-right"
                >
                  <p className="text-xs uppercase font-mono tracking-wider text-muted-foreground mb-1 flex items-center justify-end gap-1">
                    Next <ArrowRight className="h-3 w-3" />
                  </p>
                  <p className="font-headline font-semibold text-foreground group-hover:text-brand-teal-bright transition-colors">
                    {next.title}
                  </p>
                </Link>
              ) : (
                <span />
              )}
            </nav>
          )}
        </main>

        <DocsToc items={toc} />
      </div>

      <footer className="border-t border-outline-variant/30 mt-16 py-8">
        <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-muted-foreground">
          <p>
            © {new Date().getFullYear()} HydraDNS · GPL-3.0
          </p>
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
