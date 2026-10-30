import { type UIMatch, useMatches } from "react-router";

// Métadonnées déclarées par les routes dans leur export `handle`.
export interface PageHandle {
  /** Libellé dans le fil d'Ariane. */
  crumb: string;
}

// Routes actives qui déclarent un libellé de fil d'Ariane.
export function usePageMatches() {
  return useMatches().filter((m): m is UIMatch<unknown, PageHandle> => {
    const h = m.handle as PageHandle | undefined;
    return typeof h?.crumb === "string";
  });
}
