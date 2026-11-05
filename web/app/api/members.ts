import type { QueryClient } from "@tanstack/react-query";
import { type MemberFilters, toApiQuery } from "~/lib/member-filters";
import { $api } from "./queries";

// « Mes membres » : fiches du compte connecté (005, US1).
export function myMemberListQuery() {
  return $api.queryOptions("get", "/me/members");
}

export function myMemberQuery(id: number) {
  return $api.queryOptions("get", "/me/members/{id}", { params: { path: { id } } });
}

// Administration : toutes les fiches selon les critères de l'URL (US3).
export function memberListQuery(filters: MemberFilters) {
  return $api.queryOptions("get", "/admin/members", { params: { query: toApiQuery(filters) } });
}

// Fiches gérées par un compte (dialogue de suppression d'un compte, US5).
export function accountMembersQuery(accountId: number) {
  return $api.queryOptions("get", "/admin/members", {
    params: { query: { account: String(accountId), pageSize: 20 } },
  });
}

export function memberQuery(id: number) {
  return $api.queryOptions("get", "/admin/members/{id}", { params: { path: { id } } });
}

// Préfixes d'invalidation des listes et fiches des deux espaces.
export const myMemberListKey = ["get", "/me/members"] as const;
export const memberListKey = ["get", "/admin/members"] as const;

// Une écriture dans un espace peut changer ce que montre l'autre : on
// invalide les deux.
export async function invalidateMembers(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: myMemberListKey }),
    qc.invalidateQueries({ queryKey: ["get", "/me/members/{id}"] }),
    qc.invalidateQueries({ queryKey: memberListKey }),
    qc.invalidateQueries({ queryKey: ["get", "/admin/members/{id}"] }),
    // Compteurs de fiches sans valeur des champs personnalisés (011).
    qc.invalidateQueries({ queryKey: ["get", "/admin/custom-fields"] }),
  ]);
}
