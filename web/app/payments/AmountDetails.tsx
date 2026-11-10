import { formatMoney } from "~/lib/format";

// Affichage des remises et des dons (contracts/ui-routes.md 014). Les valeurs
// viennent de l'API : aucun montant n'est calculé ici.

// « Théorique 75,00 € · Remise 5,00 € » ; null sans ajustement.
export function adjustmentLabel(baseAmountCents: number, adjustmentCents: number) {
  if (adjustmentCents === 0) return null;
  const kind = adjustmentCents < 0 ? "Remise" : "Majoration";
  return `Théorique ${formatMoney(baseAmountCents)} · ${kind} ${formatMoney(Math.abs(adjustmentCents))}`;
}

interface AdjustmentNoteProps {
  baseAmountCents: number;
  adjustmentCents: number;
  reason?: string | null;
}

// Ligne d'ajustement et motif sous un montant à payer (administration).
export function AdjustmentNote({ baseAmountCents, adjustmentCents, reason }: AdjustmentNoteProps) {
  const label = adjustmentLabel(baseAmountCents, adjustmentCents);
  if (!label) return null;
  return (
    <span className="block text-sm font-normal text-muted">
      {label}
      {reason ? <span className="block whitespace-pre-line">Motif : {reason}</span> : null}
    </span>
  );
}

interface ReceivedProps {
  paidAmountCents?: number | null;
  donationCents: number;
  // Formulation de l'espace membre : remerciement.
  thanks?: boolean;
}

// « 80,00 € dont 35,00 € de don ».
export function ReceivedAmount({ paidAmountCents, donationCents, thanks }: ReceivedProps) {
  if (paidAmountCents == null) return null;
  return (
    <>
      {formatMoney(paidAmountCents)}
      {donationCents > 0 ? (
        <span className="block text-sm font-normal text-muted">
          dont {formatMoney(donationCents)} de don{thanks ? ", merci !" : ""}
        </span>
      ) : null}
    </>
  );
}

// Phrase de l'espace membre : « Montant reçu : 80,00 €, dont 35,00 € de don :
// merci ! » ; vide sans montant reçu.
export function receivedSentence(
  paidAmountCents: number | null | undefined,
  donationCents: number,
) {
  if (paidAmountCents == null) return "";
  const donation =
    donationCents > 0 ? `, dont ${formatMoney(donationCents)} de don : merci !` : ".";
  return ` Montant reçu : ${formatMoney(paidAmountCents)}${donation}`;
}
