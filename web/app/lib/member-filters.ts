import { hasSearchableTerm } from "./account-filters";

// Critères de la liste des membres, portés par l'URL (FR-011, FR-012a) :
// `?q=…&rattachement=sans-compte&page=…`.

export interface MemberFilters {
  q: string;
  noAccount: boolean;
  /** Filtres par champ personnalisé (011). */
  fields?: string[];
  page: number;
}

// Filtres par champ personnalisé, paramètre d'URL répété `champ` :
// « <champ>:true|false|empty|c<choix> » (011, FR-023). Le serveur vérifie
// que le champ et le choix existent.
const fieldFilterRe = /^\d+:(true|false|empty|c\d+)$/;

export function parseFieldFilters(params: URLSearchParams): string[] {
  return params
    .getAll("champ")
    .filter((v) => fieldFilterRe.test(v))
    .slice(0, 10);
}

export function parseMemberFilters(params: URLSearchParams): MemberFilters {
  const page = Number.parseInt(params.get("page") ?? "", 10);
  return {
    q: params.get("q") ?? "",
    noAccount: params.get("rattachement") === "sans-compte",
    fields: parseFieldFilters(params),
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

export function hasActiveMemberFilters(f: MemberFilters) {
  return f.q.trim() !== "" || f.noAccount || (f.fields?.length ?? 0) > 0;
}

// Paramètres de list-members ; une saisie trop courte n'est pas envoyée.
export function toApiQuery(f: MemberFilters) {
  return {
    ...(hasSearchableTerm(f.q) ? { q: f.q.trim() } : {}),
    ...(f.noAccount ? { account: "none" } : {}),
    ...(f.fields?.length ? { field: f.fields } : {}),
    page: f.page,
  };
}

export function toMemberSearchParams(f: MemberFilters) {
  const params = new URLSearchParams();
  if (f.q.trim() !== "") params.set("q", f.q.trim());
  if (f.noAccount) params.set("rattachement", "sans-compte");
  for (const v of f.fields ?? []) params.append("champ", v);
  if (f.page > 1) params.set("page", String(f.page));
  return params;
}
