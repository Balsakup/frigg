import { hasSearchableTerm } from "./account-filters";

// Critères de la liste des membres, portés par l'URL (FR-011, FR-012a) :
// `?q=…&rattachement=sans-compte&page=…`.

export interface MemberFilters {
  q: string;
  noAccount: boolean;
  page: number;
}

export function parseMemberFilters(params: URLSearchParams): MemberFilters {
  const page = Number.parseInt(params.get("page") ?? "", 10);
  return {
    q: params.get("q") ?? "",
    noAccount: params.get("rattachement") === "sans-compte",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

export function hasActiveMemberFilters(f: MemberFilters) {
  return f.q.trim() !== "" || f.noAccount;
}

// Paramètres de list-members ; une saisie trop courte n'est pas envoyée.
export function toApiQuery(f: MemberFilters) {
  return {
    ...(hasSearchableTerm(f.q) ? { q: f.q.trim() } : {}),
    ...(f.noAccount ? { account: "none" } : {}),
    page: f.page,
  };
}

export function toMemberSearchParams(f: MemberFilters) {
  const params = new URLSearchParams();
  if (f.q.trim() !== "") params.set("q", f.q.trim());
  if (f.noAccount) params.set("rattachement", "sans-compte");
  if (f.page > 1) params.set("page", String(f.page));
  return params;
}
