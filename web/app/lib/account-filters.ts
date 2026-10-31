// Critères de la liste des comptes, portés par l'URL (FR-004) : valeurs
// françaises visibles dans l'adresse, valeurs anglaises pour l'API
// (contracts/ui-routes.md de 002).

export type RoleFilter = "" | "member" | "admin";
export type StatusFilter = "" | "active" | "disabled";

export interface AccountFilters {
  q: string;
  role: RoleFilter;
  status: StatusFilter;
  page: number;
}

const roleFromUrl: Record<string, RoleFilter> = { membre: "member", admin: "admin" };
const roleToUrl: Record<Exclude<RoleFilter, "">, string> = { member: "membre", admin: "admin" };
const statusFromUrl: Record<string, StatusFilter> = { actif: "active", desactive: "disabled" };
const statusToUrl: Record<Exclude<StatusFilter, "">, string> = {
  active: "actif",
  disabled: "desactive",
};

/** Longueur minimale d'un fragment pour la recherche trigram (constitution, principe VI). */
export const SEARCH_MIN_LENGTH = 3;

export function parseFilters(params: URLSearchParams): AccountFilters {
  const page = Number.parseInt(params.get("page") ?? "", 10);
  return {
    q: params.get("q") ?? "",
    role: roleFromUrl[params.get("role") ?? ""] ?? "",
    status: statusFromUrl[params.get("statut") ?? ""] ?? "",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

// Vrai si la saisie contient au moins un fragment utilisable par la recherche.
export function hasSearchableTerm(q: string) {
  return q.split(/\s+/).some((word) => Array.from(word).length >= SEARCH_MIN_LENGTH);
}

export function hasActiveFilters(f: AccountFilters) {
  return f.q.trim() !== "" || f.role !== "" || f.status !== "";
}

// Paramètres de l'opération list-accounts ; une saisie trop courte n'est pas
// envoyée (la liste reste entière).
export function toApiQuery(f: AccountFilters) {
  return {
    ...(hasSearchableTerm(f.q) ? { q: f.q.trim() } : {}),
    ...(f.role ? { role: f.role } : {}),
    ...(f.status ? { status: f.status } : {}),
    page: f.page,
  };
}

// Paramètres d'URL, sans les valeurs par défaut.
export function toSearchParams(f: AccountFilters) {
  const params = new URLSearchParams();
  if (f.q.trim() !== "") params.set("q", f.q.trim());
  if (f.role) params.set("role", roleToUrl[f.role]);
  if (f.status) params.set("statut", statusToUrl[f.status]);
  if (f.page > 1) params.set("page", String(f.page));
  return params;
}
