import { useState, useEffect } from "react";
import { Menu, X, Github, Star } from "lucide-react";
import logo from "@/assets/hydradns-logo.png";

const navLinks = [
  { label: "Features", href: "#features" },
  { label: "How It Works", href: "#how-it-works" },
  { label: "Comparison", href: "#comparison" },
  { label: "Open Source", href: "#open-source" },
];

export function Navbar() {
  const [scrolled, setScrolled] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 24);
    window.addEventListener("scroll", onScroll);
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
        scrolled
          ? "bg-surface-container/70 backdrop-blur-2xl shadow-[0_10px_30px_rgba(0,212,170,0.08)]"
          : "bg-transparent"
      }`}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-20">
          {/* Logo */}
          <a href="#" className="flex items-center gap-3 group">
            <img
              src={logo}
              alt="HydraDNS"
              className="h-9 w-9 rounded-full shadow-[0_0_20px_rgba(0,212,170,0.25)] transition-transform group-hover:scale-105"
            />
            <span className="font-headline text-[15px] font-bold tracking-tighter uppercase text-foreground">
              HydraDNS
            </span>
          </a>

          {/* Desktop links */}
          <div className="hidden md:flex items-center gap-7">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
              >
                {link.label}
              </a>
            ))}
          </div>

          {/* CTAs */}
          <div className="hidden md:flex items-center gap-3">
            <a
              href="https://github.com/hydradns/hydradns"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full border border-outline-variant/40 bg-surface-container/60 text-sm font-medium text-brand-sky hover:bg-surface-container-high/80 transition-colors"
            >
              <Star className="h-3.5 w-3.5" />
              <span>Star on GitHub</span>
            </a>
            <a
              href="https://github.com/hydradns/hydradns"
              target="_blank"
              rel="noopener noreferrer"
              className="btn-primary inline-flex items-center gap-2 px-5 py-2 rounded-md text-sm font-semibold shadow-[0_0_20px_rgba(0,212,170,0.25)] hover:shadow-[0_0_28px_rgba(0,212,170,0.4)] transition-shadow"
            >
              <Github className="h-4 w-4" />
              Get Started
            </a>
          </div>

          {/* Mobile toggle */}
          <button
            className="md:hidden text-foreground p-2 -mr-2"
            onClick={() => setMobileOpen(!mobileOpen)}
            aria-label="Toggle menu"
          >
            {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>

        {/* Mobile menu */}
        {mobileOpen && (
          <div className="md:hidden glass-strong rounded-lg mt-2 mb-3 p-4 space-y-3">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                className="block text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => setMobileOpen(false)}
              >
                {link.label}
              </a>
            ))}
            <div className="pt-3 border-t border-outline-variant/40 flex flex-col gap-2">
              <a
                href="https://github.com/hydradns/hydradns"
                target="_blank"
                rel="noopener noreferrer"
                className="btn-primary inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-md text-sm font-semibold"
              >
                <Github className="h-4 w-4" />
                Get Started
              </a>
            </div>
          </div>
        )}
      </div>
    </nav>
  );
}
