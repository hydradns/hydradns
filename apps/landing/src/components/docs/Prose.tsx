import type { ReactNode } from "react";

type Props = { children: ReactNode };

export function Prose({ children }: Props) {
  return (
    <div
      className="
        prose prose-invert max-w-none
        prose-headings:font-headline prose-headings:tracking-tight prose-headings:text-foreground
        prose-h2:text-2xl prose-h2:font-bold prose-h2:mt-12 prose-h2:mb-4 prose-h2:scroll-mt-20 prose-h2:pb-2 prose-h2:border-b prose-h2:border-outline-variant/30
        prose-h3:text-lg prose-h3:font-semibold prose-h3:mt-8 prose-h3:mb-3 prose-h3:scroll-mt-20 prose-h3:text-foreground
        prose-h4:text-base prose-h4:font-semibold prose-h4:mt-6 prose-h4:mb-2 prose-h4:text-foreground
        prose-p:text-foreground/85 prose-p:leading-relaxed prose-p:text-[15px]
        prose-strong:text-foreground prose-strong:font-semibold
        prose-a:text-brand-teal-bright prose-a:no-underline hover:prose-a:text-brand-teal prose-a:font-medium prose-a:border-b prose-a:border-brand-teal/40 hover:prose-a:border-brand-teal
        prose-code:text-brand-teal-bright prose-code:bg-surface-container-high prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded prose-code:text-[0.85em] prose-code:font-mono prose-code:before:content-none prose-code:after:content-none
        prose-ul:my-4 prose-ul:text-foreground/85 prose-li:my-1 prose-li:text-[15px] prose-li:marker:text-brand-teal
        prose-ol:my-4 prose-ol:text-foreground/85
        prose-blockquote:border-l-brand-teal prose-blockquote:text-muted-foreground prose-blockquote:not-italic prose-blockquote:font-normal
        prose-hr:border-outline-variant/30
        prose-table:text-sm
        prose-th:font-headline prose-th:text-foreground prose-th:border-outline-variant/40
        prose-td:border-outline-variant/30 prose-td:text-foreground/80
        prose-img:rounded-xl prose-img:border prose-img:border-outline-variant/40
      "
    >
      {children}
    </div>
  );
}
