import { Link } from "react-router-dom";
import { Github, Star } from "lucide-react";
import logo from "@/assets/hydradns-logo.png";

export function DocsNavbar() {
  return (
    <nav className="sticky top-0 z-50 bg-surface/80 backdrop-blur-2xl border-b border-outline-variant/40">
      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          <Link to="/" className="flex items-center gap-3 group">
            <img
              src={logo}
              alt="HydraDNS"
              className="h-8 w-8 rounded-full shadow-[0_0_20px_rgba(0,212,170,0.25)] transition-transform group-hover:scale-105"
            />
            <span className="font-headline text-[14px] font-bold tracking-tighter uppercase text-foreground">
              HydraDNS
            </span>
            <span className="eyebrow hidden sm:inline-block pl-3 border-l border-outline-variant/50 ml-1">
              Docs
            </span>
          </Link>

          <div className="hidden md:flex items-center gap-6">
            <Link
              to="/"
              className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              Home
            </Link>
            <Link
              to="/docs"
              className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              Docs
            </Link>
            <a
              href="https://github.com/hydradns/hydradns"
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              GitHub
            </a>
          </div>

          <div className="flex items-center gap-2">
            <a
              href="https://github.com/hydradns/hydradns"
              target="_blank"
              rel="noopener noreferrer"
              className="hidden sm:inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-outline-variant/40 bg-surface-container/60 text-xs font-medium text-brand-sky hover:bg-surface-container-high/80 transition-colors"
            >
              <Star className="h-3.5 w-3.5" />
              Star
            </a>
            <a
              href="https://github.com/hydradns/hydradns"
              target="_blank"
              rel="noopener noreferrer"
              className="btn-primary inline-flex items-center gap-2 px-4 py-1.5 rounded-md text-xs font-semibold shadow-[0_0_20px_rgba(0,212,170,0.25)] hover:shadow-[0_0_28px_rgba(0,212,170,0.4)] transition-shadow"
            >
              <Github className="h-3.5 w-3.5" />
              Get Started
            </a>
          </div>
        </div>
      </div>
    </nav>
  );
}
