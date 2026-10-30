import createFetchClient, { type Middleware } from "openapi-fetch";
import { queryClient } from "./query-client";
import type { paths } from "./schema";

// Chemins du SPA accessibles sans session : un 401 n'y déclenche pas de redirection.
const PUBLIC_PAGES = [
  "/connexion",
  "/mot-de-passe-oublie",
  "/reinitialiser-mot-de-passe",
  "/confirmer-adresse",
  "/invitation",
];

export function isPublicPage(pathname: string) {
  return PUBLIC_PAGES.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

// Session expirée en cours de navigation : on vide le cache et on renvoie vers
// la connexion en conservant la page courante (FR-004).
const sessionExpired: Middleware = {
  onResponse({ request, response }) {
    if (response.status !== 401 || typeof window === "undefined") return;
    if (new URL(request.url).pathname === "/api/session") return;
    if (isPublicPage(window.location.pathname)) return;
    queryClient.clear();
    const next = window.location.pathname + window.location.search;
    window.location.assign(`/connexion?expiree=1&suite=${encodeURIComponent(next)}`);
  },
};

// Seul point d'accès HTTP du SPA (constitution, Frontend : aucun fetch direct).
export const client = createFetchClient<paths>({ baseUrl: "/api" });
client.use(sessionExpired);
