import { hasSearchableTerm } from "./account-filters";

// Critères de la liste des adhésions, portés par l'URL (FR-011, FR-018) :
// `?season=…&status=…&payment=…&q=…&page=…`. Saison absente : choix du
// serveur (saison courante, sinon la plus récente ouverte).

export interface MembershipFilters {
  season: string;
  status: string;
  payment: string;
  q: string;
  page: number;
}

const statuses = ["pending", "validated", "refused", "cancelled"] as const;
const payments = ["due", "announced", "paid", "free"] as const;
type Status = (typeof statuses)[number];
type PaymentFilter = (typeof payments)[number];

export function parseMembershipFilters(params: URLSearchParams): MembershipFilters {
  const page = Number.parseInt(params.get("page") ?? "", 10);
  const season = params.get("season") ?? "";
  const status = params.get("status") ?? "";
  const payment = params.get("payment") ?? "";
  return {
    season: season === "all" || /^\d+$/.test(season) ? season : "",
    status: (statuses as readonly string[]).includes(status) ? status : "",
    payment: (payments as readonly string[]).includes(payment) ? payment : "",
    q: params.get("q") ?? "",
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  };
}

export function hasActiveMembershipFilters(f: MembershipFilters) {
  return f.q.trim() !== "" || f.status !== "" || f.payment !== "";
}

// Paramètres de list-memberships ; une saisie trop courte n'est pas envoyée.
export function toMembershipApiQuery(f: MembershipFilters) {
  return {
    ...(f.season ? { season: f.season } : {}),
    ...(f.status ? { status: f.status as Status } : {}),
    ...(f.payment ? { payment: f.payment as PaymentFilter } : {}),
    ...(hasSearchableTerm(f.q) ? { q: f.q.trim() } : {}),
    page: f.page,
  };
}

export function toMembershipSearchParams(f: MembershipFilters) {
  const params = new URLSearchParams();
  if (f.season) params.set("season", f.season);
  if (f.status) params.set("status", f.status);
  if (f.payment) params.set("payment", f.payment);
  if (f.q.trim() !== "") params.set("q", f.q.trim());
  if (f.page > 1) params.set("page", String(f.page));
  return params;
}
