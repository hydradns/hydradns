import type { ReactNode } from "react";
import { Children } from "react";

type StepsProps = { children: ReactNode };

export function Steps({ children }: StepsProps) {
  const items = Children.toArray(children);
  return (
    <ol className="not-prose my-6 space-y-5 relative before:content-[''] before:absolute before:left-[15px] before:top-2 before:bottom-2 before:w-px before:bg-outline-variant/30">
      {items.map((child, i) => (
        <li key={i} className="flex gap-4 relative">
          <span className="shrink-0 flex items-center justify-center h-8 w-8 rounded-full bg-surface-container-high border border-brand-teal/40 text-sm font-mono font-semibold text-brand-teal-bright shadow-[0_0_14px_rgba(0,212,170,0.18)]">
            {i + 1}
          </span>
          <div className="flex-1 min-w-0 pt-1 pb-1">{child}</div>
        </li>
      ))}
    </ol>
  );
}

type StepProps = { title?: string; children: ReactNode };
export function Step({ title, children }: StepProps) {
  return (
    <div>
      {title && (
        <h4 className="font-headline text-base font-semibold text-foreground mb-1.5 tracking-tight">
          {title}
        </h4>
      )}
      <div className="text-sm text-muted-foreground leading-relaxed [&_p]:mb-2 [&_p:last-child]:mb-0 [&_code]:text-brand-teal-bright [&_code]:bg-surface-container-high [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded [&_code]:text-[0.85em] [&_code]:font-mono">
        {children}
      </div>
    </div>
  );
}
