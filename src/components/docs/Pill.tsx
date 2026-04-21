import type { ReactNode } from "react";

type Variant = "teal" | "sky" | "amber" | "muted";

const variants: Record<Variant, string> = {
  teal: "border-brand-teal/40 bg-brand-teal/10 text-brand-teal-bright",
  sky: "border-brand-sky/40 bg-brand-sky/10 text-brand-sky",
  amber: "border-brand-amber/40 bg-brand-amber/10 text-brand-amber",
  muted: "border-outline-variant/40 bg-surface-container-high text-muted-foreground",
};

type Props = {
  children: ReactNode;
  variant?: Variant;
};

export function Pill({ children, variant = "muted" }: Props) {
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-[11px] font-mono font-medium tracking-wide ${variants[variant]}`}
    >
      {children}
    </span>
  );
}
