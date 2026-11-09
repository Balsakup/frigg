import type { EventSummary } from "~/api/events";
import { formatAmount } from "~/lib/format";
import { Badge, type BadgeTone } from "~/ui/Badge";

const phases: Record<EventSummary["phase"], { label: string; tone: BadgeTone }> = {
  draft: { label: "Brouillon", tone: "muted" },
  upcoming: { label: "Publié", tone: "success" },
  finished: { label: "Terminé", tone: "info" },
  cancelled: { label: "Annulé", tone: "danger" },
};

export function phaseLabel(phase: EventSummary["phase"]) {
  return phases[phase].label;
}

export function priceLabel(cents: number) {
  return cents === 0 ? "Gratuit" : formatAmount(cents);
}

// Badges d'un événement : état (administration), complet, prix
// (contracts/ui-routes.md 012).
export function EventBadges({
  event,
  showPhase = true,
}: {
  event: Pick<EventSummary, "phase" | "placesLeft" | "priceCents">;
  showPhase?: boolean;
}) {
  const full = event.placesLeft === 0 && event.phase === "upcoming";
  return (
    <span className="inline-flex flex-wrap items-center gap-2">
      {showPhase || event.phase === "cancelled" ? (
        <Badge tone={phases[event.phase].tone}>{phases[event.phase].label}</Badge>
      ) : null}
      {full ? <Badge tone="warning">Complet</Badge> : null}
      <Badge tone={event.priceCents === 0 ? "success" : "brand"}>
        {priceLabel(event.priceCents)}
      </Badge>
    </span>
  );
}
