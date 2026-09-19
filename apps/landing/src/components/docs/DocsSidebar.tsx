import { NavLink } from "react-router-dom";
import { docsSections } from "@/pages/docs/nav";

export function DocsSidebar() {
  return (
    <aside className="hidden lg:block w-60 shrink-0">
      <div className="sticky top-20 py-10 pr-6">
        <p className="eyebrow mb-4">On this site</p>
        <nav className="space-y-1">
          <NavLink
            to="/docs"
            end
            className={({ isActive }) =>
              `block rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                isActive
                  ? "bg-surface-container text-brand-teal-bright"
                  : "text-muted-foreground hover:text-foreground hover:bg-surface-container-low"
              }`
            }
          >
            Overview
          </NavLink>
          <div className="h-px bg-outline-variant/30 my-3" />
          {docsSections.map((section) => {
            const Icon = section.icon;
            return (
              <NavLink
                key={section.slug}
                to={`/docs/${section.slug}`}
                className={({ isActive }) =>
                  `flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                    isActive
                      ? "bg-surface-container text-brand-teal-bright shadow-[inset_2px_0_0_0_hsl(var(--brand-teal))]"
                      : "text-muted-foreground hover:text-foreground hover:bg-surface-container-low"
                  }`
                }
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span>{section.title}</span>
              </NavLink>
            );
          })}
        </nav>
      </div>
    </aside>
  );
}
