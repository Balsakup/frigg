import type { Delivery } from "~/api/communications";
import { Badge, type BadgeTone } from "~/ui/Badge";

export type DeliveryStatus = "pending" | "sent" | "failed";

const statuses: Record<DeliveryStatus, { tone: BadgeTone; label: string }> = {
  sent: { tone: "success", label: "Envoyé" },
  pending: { tone: "info", label: "En attente" },
  failed: { tone: "danger", label: "En échec" },
};

// État d'envoi d'un destinataire.
export function DeliveryBadge({ status }: { status: DeliveryStatus }) {
  const s = statuses[status];
  return <Badge tone={s.tone}>{s.label}</Badge>;
}

// Résumé : « 18 envoyés · 1 en attente · 1 en échec ».
export function deliverySummary(d: Delivery) {
  const parts = [`${d.sent} envoyé${d.sent > 1 ? "s" : ""}`];
  if (d.pending > 0) parts.push(`${d.pending} en attente`);
  if (d.failed > 0) parts.push(`${d.failed} en échec`);
  return parts.join(" · ");
}

// Teinte d'une ligne de l'historique : échec, envoi en cours ou terminé.
export function deliveryTone(d: Delivery): BadgeTone {
  if (d.failed > 0) return "danger";
  if (d.pending > 0) return "info";
  return "success";
}
