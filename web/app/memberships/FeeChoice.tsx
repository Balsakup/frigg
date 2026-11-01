import { formatAmount, type MembershipKind } from "~/lib/format";
import { RadioGroup } from "~/ui/RadioGroup";

export interface FeeOption {
  id: number;
  label: string;
  amountCents: number;
  renewalAmountCents?: number | null;
  description?: string | null;
  amounts: { first: number; renewal: number };
}

// Montants d'un tarif : « 60,00 € · Renouvellement 40,00 € » ou « Gratuit ».
export function feeAmountsLabel(fee: FeeOption) {
  const main = formatAmount(fee.amountCents);
  return fee.renewalAmountCents != null
    ? `${main} · Renouvellement ${formatAmount(fee.renewalAmountCents)}`
    : main;
}

interface FeeChoiceProps {
  fees: FeeOption[];
  value: number | null;
  onChange: (id: number) => void;
  error?: string;
  legend?: string;
}

// Choix d'un tarif de la saison (contracts/ui-routes.md 006 › Demande).
export function FeeChoice({ fees, value, onChange, error, legend = "Tarif" }: FeeChoiceProps) {
  return (
    <RadioGroup
      legend={legend}
      required
      cards
      value={value === null ? "" : String(value)}
      onValueChange={(v) => onChange(Number(v))}
      error={error}
      options={fees.map((f) => ({
        value: String(f.id),
        label: f.label,
        description: (
          <>
            {feeAmountsLabel(f)}
            {f.description ? <span className="mt-0.5 block">{f.description}</span> : null}
          </>
        ),
      }))}
    />
  );
}

// Récapitulatif du montant selon le tarif et le type choisis : la valeur vient
// de l'API (amounts), le frontend ne calcule rien (principe VI).
export function AmountSummary({ fee, kind }: { fee?: FeeOption; kind: MembershipKind | "" }) {
  const text = fee && kind ? `Montant : ${formatAmount(fee.amounts[kind])}` : "";
  return (
    <p aria-live="polite" className="font-display text-lg font-bold empty:hidden">
      {text}
    </p>
  );
}
