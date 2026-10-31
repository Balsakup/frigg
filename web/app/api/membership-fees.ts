import type { QueryClient } from "@tanstack/react-query";
import { $api } from "./queries";
import type { components } from "./schema";

export type MembershipFee = components["schemas"]["MembershipFee"];
export type MembershipFeeList = components["schemas"]["MembershipFeeList"];

// Tarifs de l'adhésion d'une saison et actions autorisées (FR-005).
export function membershipFeesQuery(seasonId: number) {
  return $api.queryOptions("get", "/admin/seasons/{id}/fees", {
    params: { path: { id: seasonId } },
  });
}

// Toute écriture de tarif invalide les listes de tarifs.
export function invalidateMembershipFees(qc: QueryClient) {
  return qc.invalidateQueries({ queryKey: ["get", "/admin/seasons/{id}/fees"] });
}
