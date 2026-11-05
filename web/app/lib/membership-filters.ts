import { hasSearchableTerm } from "./account-filters";
import { parseFieldFilters } from "./member-filters";

// Critères de la liste des adhésions, portés par l'URL (FR-011, FR-018) :
// `?season=…&status=…&payment=…&paidMethod=…&alerts=open&q=…&page=…`. Saison absente : choix du
// serveur (saison courante, sinon la plus récente ouverte).

export interface MembershipFilters {
  season: string;
  status: string;
  payment: string;
  /** Moyen constaté d'une adhésion payée (008). */
  paidMethod?: string;
  /** « open » : adhésions portant un signalement de paiement à traiter. */
  alerts?: string;
  /** Filtres par champ personnalisé d'adhésion (011), paramètre `champ`. */
  fields?: string[];
  q: string;
  page: number;
}

const statuses = ["pending", "validated", "refused", "cancelled"] as const;
const payments = ["due", "announced", "paid", "free"] as const;
const paidMethods = ["card", "cash", "cheque", "transfer"] as const;
type PaidMethod = (typeof paidMethods)[number];
type Status = (typeof statuses)[number];
type PaymentFilter = (typeof payments)[number];

export function parseMembershipFilters(params: URLSearchParams): MembershipFilters {
  const page = Number.parseInt(params.get("page") ?? "", 10);
  const season = params.get("season") ?? "";
  const status = params.get("status") ?? "";
  const payment = params.get("payment") ?? "";
  const paidMethod = params.get("paidMethod") ?? "";
  return {
    season: season === "all" || /^\d+$/.test(season) ? season : "",
    status: (statuses as readonly string[]).includes(status) ? status : "",
    payment: (payments as readonly string[]).includes(payment) ? payment : "",
    paidMethod: (paidMethods as readonly string[]).includes(paidMethod) ? paidMethod : "",
    alerts: params.get("alerts") === "open" ? "open" : "",
    fields: parseFieldFilters(params),
    q: params.get("q") ?? "",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

export function hasActiveMembershipFilters(f: MembershipFilters) {
  return (
    f.q.trim() !== "" ||
    f.status !== "" ||
    f.payment !== "" ||
    !!f.paidMethod ||
    !!f.alerts ||
    (f.fields?.length ?? 0) > 0
  );
}

// Paramètres de list-memberships ; une saisie trop courte n'est pas envoyée.
export function toMembershipApiQuery(f: MembershipFilters) {
  return {
    ...(f.season ? { season: f.season } : {}),
    ...(f.status ? { status: f.status as Status } : {}),
    ...(f.payment ? { payment: f.payment as PaymentFilter } : {}),
    ...(f.paidMethod ? { paidMethod: f.paidMethod as PaidMethod } : {}),
    ...(f.alerts === "open" ? { alerts: "open" as const } : {}),
    ...(f.fields?.length ? { field: f.fields } : {}),
    ...(hasSearchableTerm(f.q) ? { q: f.q.trim() } : {}),
    page: f.page,
  };
}

export function toMembershipSearchParams(f: MembershipFilters) {
  const params = new URLSearchParams();
  if (f.season) params.set("season", f.season);
  if (f.status) params.set("status", f.status);
  if (f.payment) params.set("payment", f.payment);
  if (f.paidMethod) params.set("paidMethod", f.paidMethod);
  if (f.alerts) params.set("alerts", f.alerts);
  for (const v of f.fields ?? []) params.append("champ", v);
  if (f.q.trim() !== "") params.set("q", f.q.trim());
  if (f.page > 1) params.set("page", String(f.page));
  return params;
}
