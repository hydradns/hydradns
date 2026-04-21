import { Check, Copy } from "lucide-react";
import { useState } from "react";

type Props = {
  children: string;
  language?: string;
  title?: string;
};

export function CodeBlock({ children, language, title }: Props) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
      setTimeout(() => setCopied(false), 1600);
    } catch {
      /* noop */
    }
  };

  return (
    <div className="not-prose my-5 rounded-xl border border-outline-variant/40 bg-surface-container-lowest overflow-hidden group">
      {(title || language) && (
        <div className="flex items-center justify-between px-4 py-2 border-b border-outline-variant/30 bg-surface-container-low">
          <div className="flex items-center gap-2">
            {title && (
              <span className="text-xs font-medium text-foreground/90">{title}</span>
            )}
            {language && (
              <span className="text-[10px] uppercase tracking-wider font-mono text-muted-foreground px-1.5 py-0.5 rounded bg-surface-container-high border border-outline-variant/40">
                {language}
              </span>
            )}
          </div>
          <button
            onClick={handleCopy}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1.5"
            aria-label="Copy code"
          >
            {copied ? (
              <>
                <Check className="h-3.5 w-3.5 text-brand-teal-bright" />
                <span className="text-brand-teal-bright">Copied</span>
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>
      )}
      {!title && !language && (
        <button
          onClick={handleCopy}
          className="absolute right-3 top-3 z-10 px-2 py-1 rounded-md bg-surface-container-high/90 border border-outline-variant/40 text-xs text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100 transition-opacity"
          aria-label="Copy code"
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 text-brand-teal-bright" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </button>
      )}
      <pre className="relative overflow-x-auto text-[13px] leading-relaxed p-4 font-mono text-foreground/90">
        <code>{children}</code>
      </pre>
    </div>
  );
}
