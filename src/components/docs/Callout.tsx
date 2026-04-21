import { AlertTriangle, Info, Lightbulb, ShieldAlert } from "lucide-react";
import type { ReactNode } from "react";

type Variant = "note" | "tip" | "warning" | "danger";

const variants: Record<
  Variant,
  { icon: typeof Info; label: string; ring: string; tint: string; text: string }
> = {
  note: {
    icon: Info,
    label: "Note",
    ring: "border-brand-sky/40",
    tint: "bg-brand-sky/5",
    text: "text-brand-sky",
  },
  tip: {
    icon: Lightbulb,
    label: "Tip",
    ring: "border-brand-teal/40",
    tint: "bg-brand-teal/5",
    text: "text-brand-teal-bright",
  },
  warning: {
    icon: AlertTriangle,
    label: "Heads up",
    ring: "border-brand-amber/40",
    tint: "bg-brand-amber/5",
    text: "text-brand-amber",
  },
  danger: {
    icon: ShieldAlert,
    label: "Warning",
    ring: "border-brand-red/40",
    tint: "bg-brand-red/5",
    text: "text-brand-red",
  },
};

type Props = {
  variant?: Variant;
  title?: string;
  children: ReactNode;
};

export function Callout({ variant = "note", title, children }: Props) {
  const v = variants[variant];
  const Icon = v.icon;
  return (
    <div
      className={`not-prose my-6 rounded-xl border ${v.ring} ${v.tint} p-4 sm:p-5 flex gap-3`}
    >
      <Icon className={`h-5 w-5 shrink-0 mt-0.5 ${v.text}`} />
      <div className="min-w-0 flex-1">
        <p className={`text-xs font-semibold uppercase tracking-wider mb-1 ${v.text}`}>
          {title ?? v.label}
        </p>
        <div className="text-sm text-foreground/90 leading-relaxed [&_code]:text-brand-teal-bright [&_code]:bg-surface-container-high [&_code]:px-1 [&_code]:py-0.5 [&_code]:rounded [&_code]:text-[0.85em] [&_code]:font-mono [&_a]:text-brand-teal-bright [&_a]:underline [&_a]:underline-offset-2 hover:[&_a]:text-brand-teal">
          {children}
        </div>
      </div>
    </div>
  );
}
