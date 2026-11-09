import { $api } from "./queries";
import type { components } from "./schema";

export type MyRegistration = components["schemas"]["MyRegistration"];
export type AdminRegistration = components["schemas"]["AdminRegistration"];
export type AdminRegistrationRow = components["schemas"]["AdminRegistrationRow"];
export type Due = components["schemas"]["Due"];

export { invalidateEvents as invalidateRegistrations } from "./events";

// Espace membre (012 : US3, US4).
export function myRegistrationListQuery() {
  return $api.queryOptions("get", "/me/registrations");
}

export function myRegistrationQuery(id: number) {
  return $api.queryOptions("get", "/me/registrations/{id}", { params: { path: { id } } });
}

export function myDuesQuery() {
  return $api.queryOptions("get", "/me/dues");
}

// Administration (US5).
export function eventRegistrationsQuery(id: number, state: "confirmed" | "all" = "confirmed") {
  return $api.queryOptions("get", "/admin/events/{id}/registrations", {
    params: { path: { id }, query: { state } },
  });
}

export function registrationPageQuery(query: {
  event?: number;
  state?: "confirmed" | "cancelled";
  payment?: "due" | "announced" | "paid" | "free";
  alerts?: "open";
  q?: string;
  page?: number;
  pageSize?: number;
}) {
  return $api.queryOptions("get", "/admin/registrations", { params: { query } });
}

export function registrationQuery(id: number) {
  return $api.queryOptions("get", "/admin/registrations/{id}", { params: { path: { id } } });
}
