import { useEffect, useState } from "react";

export type TocItem = { id: string; label: string; depth?: 2 | 3 };

type Props = { items: TocItem[] };

export function DocsToc({ items }: Props) {
  const [active, setActive] = useState<string | null>(items[0]?.id ?? null);

  useEffect(() => {
    if (items.length === 0) return;
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]) setActive(visible[0].target.id);
      },
      { rootMargin: "-80px 0px -70% 0px", threshold: 0 }
    );
    items.forEach(({ id }) => {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    });
    return () => observer.disconnect();
  }, [items]);

  if (items.length === 0) return null;

  return (
    <aside className="hidden xl:block w-56 shrink-0">
      <div className="sticky top-20 py-10">
        <p className="eyebrow mb-4">On this page</p>
        <ul className="space-y-2 text-sm">
          {items.map((item) => {
            const isActive = active === item.id;
            return (
              <li key={item.id} className={item.depth === 3 ? "pl-3" : ""}>
                <a
                  href={`#${item.id}`}
                  className={`block border-l-2 pl-3 py-0.5 transition-colors ${
                    isActive
                      ? "border-brand-teal text-brand-teal-bright"
                      : "border-outline-variant/30 text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {item.label}
                </a>
              </li>
            );
          })}
        </ul>
      </div>
    </aside>
  );
}
