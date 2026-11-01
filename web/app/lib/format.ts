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

// Téléphone normalisé par l'API → « 06 12 34 56 78 » ; un numéro
// international reste tel qu'enregistré (research.md 005 › R3).
export function formatPhone(phone: string) {
  return /^0\d{9}$/.test(phone) ? phone.replace(/(\d{2})(?=\d)/g, "$1 ") : phone;
}

// Âge calculé par l'API → « 11 ans », « 1 an », « moins d'un an ».
export function formatAge(age: number) {
  if (age < 1) return "moins d'un an";
  return age === 1 ? "1 an" : `${age} ans`;
}

// Adhésions (006) : libellés des états, types et moyens de paiement.
export type MembershipStatus = "pending" | "validated" | "refused" | "cancelled";
export type MembershipKind = "first" | "renewal";
export type PaymentMethod = "cash" | "cheque" | "transfer";

export function membershipStatusLabel(status: MembershipStatus) {
  return {
    pending: "En attente de validation",
    validated: "Validée",
    refused: "Refusée",
    cancelled: "Annulée",
  }[status];
}

export function membershipKindLabel(kind: MembershipKind) {
  return kind === "renewal" ? "Renouvellement" : "Première adhésion";
}

export function paymentMethodLabel(method: PaymentMethod) {
  return { cash: "Espèces", cheque: "Chèque", transfer: "Virement" }[method];
}

interface PaymentLike {
  status: "due" | "announced" | "paid";
  announcedMethod?: PaymentMethod | null;
  paidOn?: string | null;
}

// État du paiement d'une adhésion validée : « À payer », « Paiement annoncé —
// virement », « Payée le 2 novembre 2026 ».
export function paymentStatusLabel(payment: PaymentLike) {
  switch (payment.status) {
    case "due":
      return "À payer";
    case "announced":
      return payment.announcedMethod
        ? `Paiement annoncé — ${paymentMethodLabel(payment.announcedMethod).toLowerCase()}`
        : "Paiement annoncé";
    case "paid":
      return payment.paidOn ? `Payée le ${formatDay(payment.paidOn)}` : "Payée";
  }
}

// IBAN normalisé par l'API → groupes de quatre caractères.
export function formatIban(iban: string) {
  return iban.replace(/(.{4})(?=.)/g, "$1 ");
}

export function previousSeasonLabel(hint: "yes" | "no" | "unknown") {
  return { yes: "Oui", no: "Non", unknown: "Inconnu" }[hint];
}

// Somme en centimes → « 0,00 € », « 1 250,00 € » (chiffres : jamais « Gratuit »).
export function formatMoney(cents: number) {
  return amountFormat.format(cents / 100);
}
