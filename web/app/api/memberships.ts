import type { QueryClient } from "@tanstack/react-query";
import { $api } from "./queries";
import type { components } from "./schema";

export type MyMembership = components["schemas"]["MyMembership"];
export type AdminMembership = components["schemas"]["AdminMembership"];
export type MembershipFeeRef = components["schemas"]["MembershipFeeRef"];
export type MembershipOffer = components["schemas"]["MembershipOffer"];
export type Payment = components["schemas"]["Payment"];
export type PaymentMethodInfo = components["schemas"]["PaymentMethodInfo"];
export type MembershipStats = components["schemas"]["MembershipStats"];

// Espace membre (006 : US1, US3, US4).
export function membershipOfferQuery() {
  return $api.queryOptions("get", "/me/memberships/offer");
}

export function myMembershipListQuery(memberId?: number) {
  return $api.queryOptions("get", "/me/memberships", {
    params: { query: memberId ? { member: memberId } : {} },
  });
}

export function myMembershipQuery(id: number) {
  return $api.queryOptions("get", "/me/memberships/{id}", { params: { path: { id } } });
}

// Administration (US2, US6 à US8).
export function membershipPageQuery(query: {
  season?: string;
  status?: "pending" | "validated" | "refused" | "cancelled";
  payment?: "due" | "announced" | "paid" | "free";
  paidMethod?: "card" | "cash" | "cheque" | "transfer";
  alerts?: "open";
  field?: string[];
  member?: number;
  q?: string;
  page?: number;
  pageSize?: number;
}) {
  return $api.queryOptions("get", "/admin/memberships", { params: { query } });
}

export function membershipQuery(id: number) {
  return $api.queryOptions("get", "/admin/memberships/{id}", { params: { path: { id } } });
}

export function membershipStatsQuery(season: number) {
  return $api.queryOptions("get", "/admin/memberships/stats", { params: { query: { season } } });
}

export function membershipDashboardQuery() {
  return $api.queryOptions("get", "/admin/memberships/dashboard");
}

// Une écriture dans un espace peut changer ce que montrent l'autre, les fiches
// membre et les tarifs (inUse) : on invalide l'ensemble.
export async function invalidateMemberships(qc: QueryClient) {
  await Promise.all(
    [
      ["get", "/me/memberships"],
      ["get", "/me/memberships/offer"],
      ["get", "/me/memberships/{id}"],
      ["get", "/admin/memberships"],
      ["get", "/admin/memberships/{id}"],
      ["get", "/admin/memberships/stats"],
      ["get", "/admin/memberships/dashboard"],
      ["get", "/admin/members/{id}"],
      ["get", "/admin/seasons/{id}/fees"],
      // Compteurs d'adhésions sans valeur des champs personnalisés (011).
      ["get", "/admin/custom-fields"],
    ].map((queryKey) => qc.invalidateQueries({ queryKey })),
  );
}
