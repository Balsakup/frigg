import { type AccountFilters, toApiQuery } from "~/lib/account-filters";
import { $api } from "./queries";

// Liste des comptes selon les critères de l'URL (FR-001 à FR-004).
export function accountListQuery(filters: AccountFilters) {
  return $api.queryOptions("get", "/admin/accounts", { params: { query: toApiQuery(filters) } });
}

// Fiche d'un compte (FR-005).
export function accountQuery(id: number) {
  return $api.queryOptions("get", "/admin/accounts/{id}", { params: { path: { id } } });
}

// Préfixes d'invalidation : toutes les pages et tous les critères de la liste,
// et les invitations (auteur supprimé).
export const accountListKey = ["get", "/admin/accounts"] as const;
export const invitationListKey = $api.queryOptions("get", "/admin/invitations").queryKey;
