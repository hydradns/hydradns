type Props = {
  src: string;
  alt: string;
  caption?: string;
};

export function Screenshot({ src, alt, caption }: Props) {
  return (
    <figure className="not-prose my-8">
      <div className="relative rounded-xl border border-outline-variant/40 bg-surface-container-lowest overflow-hidden shadow-[0_20px_60px_-20px_rgba(0,212,170,0.25)]">
        <div className="flex items-center gap-1.5 px-4 py-2.5 border-b border-outline-variant/30 bg-surface-container-low">
          <span className="h-2.5 w-2.5 rounded-full bg-brand-red/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-brand-amber/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-brand-teal/70" />
        </div>
        <img src={src} alt={alt} className="w-full h-auto block" loading="lazy" />
      </div>
      {caption && (
        <figcaption className="mt-3 text-center text-xs text-muted-foreground font-mono tracking-wide">
          {caption}
        </figcaption>
      )}
    </figure>
  );
}
