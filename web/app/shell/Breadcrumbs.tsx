import { ChevronRight } from "lucide-react";
import { Link } from "react-router";
import { Icon } from "~/ui/Icon";
import { usePageMatches } from "./usePageMeta";

// Fil d'Ariane construit depuis les `handle` des routes actives.
export function Breadcrumbs() {
  const matches = usePageMatches();
  if (matches.length === 0) return null;
  return (
    <nav aria-label="Fil d'Ariane" className="mb-2">
      <ol className="flex flex-wrap items-center gap-1.5 text-sm text-muted">
        {matches.map((m, i) => {
          const label = m.handle.crumb;
          const last = i === matches.length - 1;
          return (
            <li key={m.id} className="flex items-center gap-1.5">
              {i > 0 ? <Icon icon={ChevronRight} className="h-4 w-4" /> : null}
              {last ? (
                <span aria-current="page">{label}</span>
              ) : (
                <Link to={m.pathname} className="underline underline-offset-2 hover:text-brand-ink">
                  {label}
                </Link>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
