import {
  type MembershipStatus,
  membershipStatusLabel,
  type PaymentMethod,
  paymentStatusLabel,
} from "~/lib/format";
import { Badge, type BadgeTone } from "~/ui/Badge";

const statusTones: Record<MembershipStatus, BadgeTone> = {
  pending: "warning",
  validated: "success",
  refused: "danger",
  cancelled: "muted",
};

interface BadgesProps {
  status: MembershipStatus;
  amountCents?: number | null;
  payment?: {
    status: "due" | "announced" | "paid";
    announcedMethod?: PaymentMethod | null;
    paidOn?: string | null;
  } | null;
}

// Badges d'état et de paiement d'une adhésion (contracts/ui-routes.md 006).
export function MembershipBadges({ status, amountCents, payment }: BadgesProps) {
  const paymentTone: BadgeTone =
    payment?.status === "due" ? "warning" : payment?.status === "announced" ? "info" : "success";
  return (
    <span className="inline-flex flex-wrap items-center gap-2">
      <Badge tone={statusTones[status]}>{membershipStatusLabel(status)}</Badge>
      {status === "validated" && payment ? (
        <Badge tone={paymentTone}>{paymentStatusLabel(payment)}</Badge>
      ) : null}
      {status === "validated" && !payment && amountCents === 0 ? (
        <Badge tone="success">Gratuite</Badge>
      ) : null}
    </span>
  );
}
