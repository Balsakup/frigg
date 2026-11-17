import { Checkbox as BaseCheckbox } from "@base-ui/react/checkbox";
import { Check, CircleAlert } from "lucide-react";
import { useId } from "react";
import { Icon } from "~/ui/Icon";

interface ConsentFieldProps {
  /** Nom de l'association, destinataire des données (FR-007). */
  association: string;
  /** Finalité du traitement : gestion de l'adhésion ou de l'inscription. */
  purpose: "membership" | "registration";
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  error?: string;
}

const purposes = {
  membership: "la gestion de ma demande d'adhésion et de mon adhésion",
  registration: "la gestion de mon inscription à l'événement",
};

// Case de consentement obligatoire des formulaires publics (FR-007) : elle
// nomme l'association destinataire, la finalité et les durées de
// conservation. L'erreur est reliée à la case.
export function ConsentField({
  association,
  purpose,
  checked,
  onCheckedChange,
  error,
}: ConsentFieldProps) {
  const id = useId();
  const errorId = error ? `${id}-erreur` : undefined;
  return (
    <div className="min-w-0">
      {/* biome-ignore lint/a11y/noLabelWithoutControl: le contrôle est la case base-ui imbriquée. */}
      <label className="flex cursor-pointer items-start gap-2.5">
        <BaseCheckbox.Root
          name="consent"
          checked={checked}
          onCheckedChange={(c) => onCheckedChange(c)}
          aria-describedby={errorId}
          aria-invalid={error ? true : undefined}
          aria-required
          className="mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded border border-line-strong bg-surface text-white data-[checked]:border-brand-ink data-[checked]:bg-brand-ink"
        >
          <BaseCheckbox.Indicator>
            <Icon icon={Check} className="h-3.5 w-3.5" />
          </BaseCheckbox.Indicator>
        </BaseCheckbox.Root>
        <span className="min-w-0 [overflow-wrap:anywhere]">
          J'accepte que {association} traite les données saisies pour {purposes[purpose]}. Une
          demande non confirmée est supprimée sous 7 jours, et une fiche sans activité depuis 12
          mois est proposée à la suppression.
          <span className="text-muted"> (obligatoire)</span>
        </span>
      </label>
      {error ? (
        <p id={errorId} className="mt-1 flex items-start gap-1.5 text-sm font-bold text-danger">
          <Icon icon={CircleAlert} className="mt-0.5 h-4 w-4" />
          <span>{error}</span>
        </p>
      ) : null}
    </div>
  );
}
