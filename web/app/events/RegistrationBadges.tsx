import type { Payment } from "~/api/memberships";
import { paymentStatusLabel } from "~/lib/format";
import { Badge, type BadgeTone } from "~/ui/Badge";

// État d'une inscription et de son paiement (contracts/ui-routes.md 012).
export function RegistrationBadges({
  state,
  payment,
}: {
  state: "confirmed" | "cancelled";
  payment?: Payment | null;
}) {
  const tone: BadgeTone =
    payment?.status === "due" ? "warning" : payment?.status === "announced" ? "info" : "success";
  return (
    <span className="inline-flex flex-wrap items-center gap-2">
      {state === "cancelled" ? (
        <Badge tone="muted">Annulée</Badge>
      ) : (
        <Badge tone="success">Confirmée</Badge>
      )}
      {payment ? (
        <Badge tone={tone}>
          {paymentStatusLabel(payment)}
          {payment.status === "paid" && payment.paidMethod === "card" ? " — carte bancaire" : ""}
        </Badge>
      ) : (
        <Badge tone="success">Gratuite</Badge>
      )}
      {payment?.status === "paid" && payment.online?.test ? (
        <Badge tone="warning">Mode test</Badge>
      ) : null}
    </span>
  );
}
