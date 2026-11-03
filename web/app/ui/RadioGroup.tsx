import { Radio } from "@base-ui/react/radio";
import { RadioGroup as BaseRadioGroup } from "@base-ui/react/radio-group";
import { CircleAlert } from "lucide-react";
import { type ReactNode, useId } from "react";
import { cn } from "~/lib/cn";
import { Icon } from "./Icon";

export interface RadioOption<V extends string> {
  value: V;
  label: ReactNode;
  /** Contenu secondaire sous le libellé (montant, description…). */
  description?: ReactNode;
}

interface RadioGroupProps<V extends string> {
  legend: string;
  value: V | "";
  onValueChange: (value: V) => void;
  options: RadioOption<V>[];
  hint?: string;
  error?: string;
  required?: boolean;
  /** Options présentées en cartes (choix riches) plutôt qu'en liste simple. */
  cards?: boolean;
  className?: string;
}

// Groupe de boutons radio accessible (base-ui) : légende, aide et erreur
// reliées au groupe (contracts/ui-routes.md 006 › Accessibilité).
export function RadioGroup<V extends string>({
  legend,
  value,
  onValueChange,
  options,
  hint,
  error,
  required,
  cards,
  className,
}: RadioGroupProps<V>) {
  const id = useId();
  const hintId = hint ? `${id}-aide` : undefined;
  const errorId = error ? `${id}-erreur` : undefined;
  return (
    <div className={cn("min-w-0", className)}>
      <p id={`${id}-legende`} className="mb-1 font-bold">
        {legend}
        {required ? <span className="font-normal text-muted"> (obligatoire)</span> : null}
      </p>
      {hint ? (
        <p id={hintId} className="mb-2 text-sm text-muted">
          {hint}
        </p>
      ) : null}
      <BaseRadioGroup
        value={value === "" ? null : value}
        onValueChange={(v) => onValueChange(v as V)}
        aria-labelledby={`${id}-legende`}
        aria-describedby={[hintId, errorId].filter(Boolean).join(" ") || undefined}
        aria-invalid={error ? true : undefined}
        className={cn("flex flex-col", cards ? "gap-2" : "gap-2.5")}
      >
        {options.map((o) => (
          // biome-ignore lint/a11y/noLabelWithoutControl: le contrôle est le bouton radio base-ui imbriqué.
          <label
            key={o.value}
            className={cn(
              "flex cursor-pointer items-start gap-2.5",
              cards &&
                "rounded-md border border-line-strong bg-surface p-3 has-[[data-checked]]:border-brand-ink has-[[data-checked]]:bg-brand/5",
            )}
          >
            <Radio.Root
              value={o.value}
              className="mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full border border-line-strong bg-surface data-[checked]:border-brand-ink"
            >
              <Radio.Indicator className="h-2.5 w-2.5 rounded-full bg-brand-ink" />
            </Radio.Root>
            <span className="min-w-0 [overflow-wrap:anywhere]">
              <span className={cards ? "font-bold" : undefined}>{o.label}</span>
              {o.description ? (
                <span className="mt-0.5 block text-sm text-muted">{o.description}</span>
              ) : null}
            </span>
          </label>
        ))}
      </BaseRadioGroup>
      {error ? (
        <p id={errorId} className="mt-1 flex items-start gap-1.5 text-sm font-bold text-danger">
          <Icon icon={CircleAlert} className="mt-0.5 h-4 w-4" />
          <span>{error}</span>
        </p>
      ) : null}
    </div>
  );
}
