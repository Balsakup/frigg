import { useMatches } from "react-router";

// Métadonnées déclarées par les routes dans leur export `handle`.
export interface PageHandle {
  /** Libellé dans le fil d'Ariane, fixe ou calculé depuis les données du loader. */
  crumb: string | ((data: unknown) => string | undefined);
}

export interface PageMatch {
  id: string;
  pathname: string;
  label: string;
}

// Routes actives qui déclarent un libellé de fil d'Ariane.
export function usePageMatches(): PageMatch[] {
  return useMatches().flatMap((m) => {
    const crumb = (m.handle as PageHandle | undefined)?.crumb;
    const label = typeof crumb === "function" ? crumb(m.loaderData) : crumb;
    return label ? [{ id: m.id, pathname: m.pathname, label }] : [];
  });
}
