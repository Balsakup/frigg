import { useId, useRef } from "react";
import { formatAmount } from "~/lib/format";
import { Button } from "~/ui/Button";
import { Field, TextAreaField } from "~/ui/Field";
import type { TrackedAmount } from "./useTrackedAmount";

interface AmountToPayFieldProps {
  amount: TrackedAmount;
  // Montant théorique fourni par l'API ; undefined tant que le tarif et le
  // type ne sont pas choisis.
  theoreticalCents: number | undefined;
  errors?: { amount?: string; adjustmentReason?: string };
}

// Montant à payer d'une adhésion ou d'une inscription, prérempli par le
// montant théorique, avec le motif d'un ajustement (contracts/ui-routes.md
// 014 › AmountToPayField). Le serveur décide si le motif est requis.
export function AmountToPayField({ amount, theoreticalCents, errors }: AmountToPayFieldProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const id = useId();
  const hint =
    theoreticalCents === undefined
      ? "Choisissez d'abord le tarif et le type."
      : `Montant théorique : ${formatAmount(theoreticalCents)}.`;
  return (
    <div className="flex flex-col gap-3">
      <Field
        ref={inputRef}
        id={`${id}-montant`}
        label="Montant à payer (€)"
        inputMode="decimal"
        autoComplete="off"
        className="sm:max-w-60"
        value={amount.text}
        onChange={(e) => amount.setText(e.target.value)}
        hint={hint}
        error={errors?.amount}
      />
      {!amount.tracked ? (
        <Button
          type="button"
          variant="link"
          className="self-start"
          onClick={() => {
            amount.reset();
            inputRef.current?.focus();
          }}
        >
          Revenir au montant théorique
        </Button>
      ) : null}
      <div aria-live="polite">
        {!amount.tracked ? (
          <TextAreaField
            id={`${id}-motif`}
            label="Motif de l'ajustement"
            maxLength={200}
            value={amount.reason}
            onChange={(e) => amount.setReason(e.target.value)}
            hint={`Obligatoire si le montant diffère du montant théorique. ${amount.reason.length}/200 caractères. Visible seulement par les administrateurs.`}
            error={errors?.adjustmentReason}
          />
        ) : null}
      </div>
    </div>
  );
}
