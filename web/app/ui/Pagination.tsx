import { ChevronLeft, ChevronRight } from "lucide-react";
import { Link } from "react-router";
import { Icon } from "./Icon";

interface PaginationProps {
  page: number;
  pageCount: number;
  /** Adresse de la page n (critères de l'URL conservés). */
  hrefFor: (page: number) => string;
}

const linkClass =
  "inline-flex h-10 items-center gap-1 rounded-md border border-line-strong bg-surface px-3 font-bold hover:bg-canvas";

// Navigation entre les pages d'une liste, par liens (l'URL porte la page).
export function Pagination({ page, pageCount, hrefFor }: PaginationProps) {
  if (pageCount <= 1) return null;
  return (
    <nav aria-label="Pagination" className="flex flex-wrap items-center justify-between gap-3">
      <p aria-current="page">
        Page {page} sur {pageCount}
      </p>
      <div className="flex gap-2">
        {page > 1 ? (
          <Link to={hrefFor(page - 1)} className={linkClass}>
            <Icon icon={ChevronLeft} className="h-4 w-4" />
            Précédente
          </Link>
        ) : null}
        {page < pageCount ? (
          <Link to={hrefFor(page + 1)} className={linkClass}>
            Suivante
            <Icon icon={ChevronRight} className="h-4 w-4" />
          </Link>
        ) : null}
      </div>
    </nav>
  );
}
