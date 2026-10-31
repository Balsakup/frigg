const dateFormat = new Intl.DateTimeFormat("fr-FR", { dateStyle: "long" });

// Date lisible en français, par exemple « 6 novembre 2026 ».
export function formatDate(value: string | Date) {
  return dateFormat.format(typeof value === "string" ? new Date(value) : value);
}

export function roleLabel(role: "member" | "admin") {
  return role === "admin" ? "Administrateur" : "Membre";
}

// Date civile « 2026-09-01 » → « 1 septembre 2026 ». La date est construite en
// heure locale : formatDate lit une chaîne ISO en UTC et décalerait le jour à
// l'ouest de Greenwich (research.md 003 › R9).
export function formatDay(value: string) {
  const [y, m, d] = value.split("-").map(Number);
  return dateFormat.format(new Date(y, m - 1, d));
}

// Teinte du badge d'état d'une saison (contracts/ui-routes.md 003).
export const seasonStatusTone = { draft: "warning", open: "success", closed: "muted" } as const;

export function seasonStatusLabel(status: "draft" | "open" | "closed") {
  return { draft: "Brouillon", open: "Ouverte", closed: "Close" }[status];
}

const amountFormat = new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" });

// Montant d'un tarif en centimes → « 15,50 € », ou « Gratuit » (FR-002).
export function formatAmount(cents: number) {
  return cents === 0 ? "Gratuit" : amountFormat.format(cents / 100);
}

// Valeur de préremplissage du champ Montant : « 30 », « 15,50 ». Le serveur
// analyse la saisie (research.md 004 › R3).
export function amountInput(cents: number) {
  const euros = Math.trunc(cents / 100);
  const rest = cents % 100;
  return rest === 0 ? String(euros) : `${euros},${String(rest).padStart(2, "0")}`;
}
