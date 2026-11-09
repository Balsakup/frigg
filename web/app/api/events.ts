import type { QueryClient } from "@tanstack/react-query";
import { $api } from "./queries";
import type { components } from "./schema";

export type EventSummary = components["schemas"]["EventSummary"];
export type AdminEvent = components["schemas"]["AdminEvent"];
export type MyEvent = components["schemas"]["MyEvent"];
export type MyEventMember = components["schemas"]["MyEventMember"];
export type EventBody = components["schemas"]["EventBody"];

export type EventStatusFilter = "draft" | "published" | "cancelled" | "finished";

// Administration (012 : US1, US6).
export function eventListQuery(query: { season?: string; status?: EventStatusFilter }) {
  return $api.queryOptions("get", "/admin/events", { params: { query } });
}

export function eventQuery(id: number) {
  return $api.queryOptions("get", "/admin/events/{id}", { params: { path: { id } } });
}

// Espace membre (US2, US3).
export function myEventListQuery() {
  return $api.queryOptions("get", "/me/events");
}

export function myEventQuery(id: number) {
  return $api.queryOptions("get", "/me/events/{id}", { params: { path: { id } } });
}

// Une écriture sur un événement change ce que voient les deux espaces, les
// inscriptions et « À régler » : on invalide l'ensemble.
export async function invalidateEvents(qc: QueryClient) {
  await Promise.all(
    [
      ["get", "/admin/events"],
      ["get", "/admin/events/{id}"],
      ["get", "/admin/events/{id}/registrations"],
      ["get", "/admin/registrations"],
      ["get", "/admin/registrations/{id}"],
      ["get", "/admin/memberships/dashboard"],
      ["get", "/me/events"],
      ["get", "/me/events/{id}"],
      ["get", "/me/registrations"],
      ["get", "/me/registrations/{id}"],
      ["get", "/me/dues"],
    ].map((queryKey) => qc.invalidateQueries({ queryKey })),
  );
}
