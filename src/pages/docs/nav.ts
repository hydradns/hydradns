import type { LucideIcon } from "lucide-react";
import {
  Rocket,
  Package,
  LayoutDashboard,
  Shield,
  Terminal,
  Bot,
  Network,
  LifeBuoy,
} from "lucide-react";

export type DocsSection = {
  slug: string;
  title: string;
  eyebrow: string;
  description: string;
  icon: LucideIcon;
};

export const docsSections: DocsSection[] = [
  {
    slug: "getting-started",
    title: "Getting Started",
    eyebrow: "01 · Start here",
    description:
      "What HydraDNS is, how to bring up the stack in 60 seconds, and how to route your whole network through it.",
    icon: Rocket,
  },
  {
    slug: "installation",
    title: "Installation",
    eyebrow: "02 · Deploy",
    description:
      "Docker Compose for your workstation, Raspberry Pi for always-on home use, and how to keep it up to date.",
    icon: Package,
  },
  {
    slug: "dashboard",
    title: "Dashboard Guide",
    eyebrow: "03 · The UI",
    description:
      "A walkthrough of every screen in the dashboard — engine toggle, blocklists, policies, query logs, and metrics.",
    icon: LayoutDashboard,
  },
  {
    slug: "policies-and-blocklists",
    title: "Policies & Blocklists",
    eyebrow: "04 · Filtering",
    description:
      "How the 3-step DNS pipeline decides block vs allow, and how to shape it with your own rules and lists.",
    icon: Shield,
  },
  {
    slug: "cli",
    title: "CLI Reference",
    eyebrow: "05 · Terminal",
    description:
      "The hydra CLI — manage status, policies, blocklists, logs, and metrics from the terminal.",
    icon: Terminal,
  },
  {
    slug: "mcp",
    title: "MCP & AI Integration",
    eyebrow: "06 · Agents",
    description:
      "Wire HydraDNS into Claude Code or Gemini CLI so an AI assistant can manage your DNS firewall conversationally.",
    icon: Bot,
  },
  {
    slug: "architecture",
    title: "Architecture",
    eyebrow: "07 · Internals",
    description:
      "Control plane vs data plane, the query pipeline, auth model, and the REST envelope.",
    icon: Network,
  },
  {
    slug: "troubleshooting",
    title: "Troubleshooting",
    eyebrow: "08 · Help",
    description:
      "Port 53 on WSL, router fallback DNS pitfalls, health checks, and where to find the logs.",
    icon: LifeBuoy,
  },
];

export function findSection(slug: string): DocsSection | undefined {
  return docsSections.find((s) => s.slug === slug);
}

export function findAdjacent(slug: string): {
  prev?: DocsSection;
  next?: DocsSection;
} {
  const i = docsSections.findIndex((s) => s.slug === slug);
  if (i === -1) return {};
  return {
    prev: i > 0 ? docsSections[i - 1] : undefined,
    next: i < docsSections.length - 1 ? docsSections[i + 1] : undefined,
  };
}
