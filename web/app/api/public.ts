import { useQuery } from "@tanstack/react-query";
import { $api } from "./queries";
import type { components } from "./schema";

// Parcours publics (016) : opérations sans session. Le jeton d'un lien
// d'e-mail passe toujours dans le corps, jamais dans l'URL (research.md › R1).

export type PublicMembershipPage = components["schemas"]["PublicMembershipPage"];
export type PublicMembershipBody = components["schemas"]["PublicMembershipBody"];
export type PublicSeason = components["schemas"]["PublicSeason"];
export type ConfirmOutcome = components["schemas"]["ConfirmOutcome"];
export type PublicEvent = components["schemas"]["PublicEvent"];
export type PublicSubmissionBody = components["schemas"]["PublicSubmissionBody"];

export function publicMembershipPageQuery() {
  return $api.queryOptions("get", "/public/membership");
}

// Page d'adhésion publique : saisons ouvertes, tarifs et champs à remplir.
export function usePublicMembershipPage() {
  return useQuery(publicMembershipPageQuery());
}

// Dépôt d'une demande d'adhésion publique : réponse neutre (202).
export function useSubmitPublicMembership() {
  return $api.useMutation("post", "/public/membership");
}

export function publicEventQuery(slug: string) {
  return $api.queryOptions("get", "/public/events/{slug}", { params: { path: { slug } } });
}

// Page publique d'un événement (US3) : 404 identique pour tout lien sans page.
export function usePublicEvent(slug: string) {
  return useQuery({ ...publicEventQuery(slug), retry: false });
}

// Dépôt d'une inscription publique : réponse neutre (202), 409 si
// l'inscription est devenue impossible.
export function useSubmitPublicRegistration() {
  return $api.useMutation("post", "/public/events/{slug}/registrations");
}

// Confirmation de l'adresse d'un dépôt, depuis le lien de l'e-mail.
export function useConfirmPublicSubmission() {
  return $api.useMutation("post", "/public/submissions/confirm");
}
