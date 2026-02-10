import logo from "@/assets/hydradns-logo.png";

const links = [
  { label: "GitHub", href: "https://github.com/lopster568/HydraDNS" },
  { label: "Documentation", href: "https://github.com/lopster568/HydraDNS#readme" },
  { label: "Contributing", href: "https://github.com/lopster568/HydraDNS/blob/main/CONTRIBUTING.md" },
  { label: "License", href: "https://github.com/lopster568/HydraDNS/blob/main/LICENSE" },
];

export function Footer() {
  return (
    <footer className="border-t border-border/50 py-12">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row items-center justify-between gap-6">
          <div className="flex items-center gap-2">
            <img src={logo} alt="HydraDNS" className="h-6 w-6" />
            <span className="text-sm text-muted-foreground">
              © 2026 HydraDNS. Open-source under GPL-3.0.
            </span>
          </div>

          <div className="flex items-center gap-6">
            {links.map((link) => (
              <a
                key={link.label}
                href={link.href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                {link.label}
              </a>
            ))}
          </div>

          <p className="text-sm text-muted-foreground italic">
            Made with Go, gRPC, and paranoia.
          </p>
        </div>
      </div>
    </footer>
  );
}
