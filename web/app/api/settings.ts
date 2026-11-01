import type { QueryClient } from "@tanstack/react-query";
import { $api } from "./queries";
import type { components } from "./schema";

export type PaymentSettings = components["schemas"]["PaymentSettings"];

// Moyens de paiement proposés aux adhérents (006, US5).
export function paymentSettingsQuery() {
  return $api.queryOptions("get", "/admin/settings/payment");
}

export async function invalidatePaymentSettings(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: ["get", "/admin/settings/payment"] }),
    qc.invalidateQueries({ queryKey: ["get", "/admin/memberships/dashboard"] }),
    qc.invalidateQueries({ queryKey: ["get", "/me/memberships/{id}"] }),
  ]);
}
