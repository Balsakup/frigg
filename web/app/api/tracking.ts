import { type QueryClient, useQuery } from "@tanstack/react-query";
import { isProblem } from "./problem";
import { $api } from "./queries";
import type { components } from "./schema";

// Suivi par lien personnel (016, US2) : l'accès est porté par la session après
// `open-tracking-link` ; le jeton n'apparaît jamais dans une URL envoyée au
// serveur (research.md › R1). Un 401 signifie « lien absent ou expiré ».

export type Tracking = components["schemas"]["Tracking"];
export type TrackedItem = components["schemas"]["TrackedItem"];
export type TrackedMembership = components["schemas"]["TrackedMembership"];
export type TrackedRegistration = components["schemas"]["TrackedRegistration"];

// Accès par lien absent, expiré ou révoqué.
export function isLinkExpired(error: unknown): boolean {
  return isProblem(error) && error.status === 401;
}

export function trackingQuery() {
  return $api.queryOptions("get", "/tracking");
}

export function trackedMembershipQuery(id: number) {
  return $api.queryOptions("get", "/tracking/memberships/{id}", { params: { path: { id } } });
}

export function trackedRegistrationQuery(id: number) {
  return $api.queryOptions("get", "/tracking/registrations/{id}", { params: { path: { id } } });
}

// Espace de suivi ; `expired` traduit le 401.
export function useTracking(enabled = true) {
  const query = useQuery({ ...trackingQuery(), enabled });
  return { ...query, expired: isLinkExpired(query.error) };
}

// Une adhésion du périmètre du lien ; `expired` traduit le 401.
export function useTrackedMembership(id: number, enabled = true) {
  const query = useQuery({ ...trackedMembershipQuery(id), enabled });
  return { ...query, expired: isLinkExpired(query.error) };
}

// Une inscription du périmètre du lien ; `expired` traduit le 401.
export function useTrackedRegistration(id: number, enabled = true) {
  const query = useQuery({ ...trackedRegistrationQuery(id), enabled });
  return { ...query, expired: isLinkExpired(query.error) };
}

// Après une écriture par lien : l'espace et la page de paiement sont relus.
export async function invalidateTracking(qc: QueryClient) {
  await Promise.all(
    [
      ["get", "/tracking"],
      ["get", "/tracking/memberships/{id}"],
      ["get", "/tracking/registrations/{id}"],
    ].map((queryKey) => qc.invalidateQueries({ queryKey })),
  );
}

export function useCancelTrackedMembership() {
  return $api.useMutation("post", "/tracking/memberships/{id}/cancel");
}

export function useAnnounceTrackedMembership() {
  return $api.useMutation("put", "/tracking/memberships/{id}/payment");
}

export function useStartTrackedMembershipCard() {
  return $api.useMutation("post", "/tracking/memberships/{id}/card-payment");
}

export function useConfirmTrackedMembershipCard() {
  return $api.useMutation("post", "/tracking/memberships/{id}/card-payment/confirm");
}

export function useCancelTrackedRegistration() {
  return $api.useMutation("post", "/tracking/registrations/{id}/cancel");
}

export function useAnnounceTrackedRegistration() {
  return $api.useMutation("put", "/tracking/registrations/{id}/payment");
}

export function useStartTrackedRegistrationCard() {
  return $api.useMutation("post", "/tracking/registrations/{id}/card-payment");
}

export function useConfirmTrackedRegistrationCard() {
  return $api.useMutation("post", "/tracking/registrations/{id}/card-payment/confirm");
}

// Ouverture d'un lien d'e-mail : le jeton passe dans le corps (R1).
export function useOpenTrackingLink() {
  return $api.useMutation("post", "/public/tracking/open");
}

// « Recevoir mon lien de suivi » : réponse neutre (202).
export function useRequestTrackingLink() {
  return $api.useMutation("post", "/public/tracking/link-request");
}

export function useCloseTracking() {
  return $api.useMutation("post", "/tracking/close");
}

// Création d'un compte depuis le suivi (016, US4) : la réponse est la session
// du compte créé ; l'accès par lien est retiré de la session.
export function useCreateTrackingAccount() {
  return $api.useMutation("post", "/tracking/account");
}
