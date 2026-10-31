import type { QueryClient } from "@tanstack/react-query";
import { $api } from "./queries";
import type { components } from "./schema";

export type Season = components["schemas"]["Season"];
export type SeasonList = components["schemas"]["SeasonList"];

// Liste des saisons et état de leur mise en place (FR-020, FR-023).
export function seasonListQuery() {
  return $api.queryOptions("get", "/admin/seasons");
}

// Fiche d'une saison (FR-022).
export function seasonQuery(id: number) {
  return $api.queryOptions("get", "/admin/seasons/{id}", { params: { path: { id } } });
}

// Toute écriture peut changer libellés, états et actions des autres saisons :
// la liste et toutes les fiches sont invalidées.
export function invalidateSeasons(qc: QueryClient) {
  return Promise.all([
    qc.invalidateQueries({ queryKey: ["get", "/admin/seasons"] }),
    qc.invalidateQueries({ queryKey: ["get", "/admin/seasons/{id}"] }),
  ]);
}
