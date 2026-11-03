import { Checkbox as BaseCheckbox } from "@base-ui/react/checkbox";
import { CheckboxGroup as BaseCheckboxGroup } from "@base-ui/react/checkbox-group";
import { Check, CircleAlert } from "lucide-react";
import { type ReactNode, useId } from "react";
import { cn } from "~/lib/cn";
import { Icon } from "./Icon";

export interface CheckboxOption<V extends string> {
  value: V;
  label: ReactNode;
  description?: ReactNode;
}

interface CheckboxGroupProps<V extends string> {
  legend: string;
  value: V[];
  onValueChange: (value: V[]) => void;
  options: CheckboxOption<V>[];
  hint?: string;
  error?: string;
  className?: string;
}

const boxClass =
  "mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded border border-line-strong bg-surface text-white data-[checked]:border-brand data-[checked]:bg-brand data-[disabled]:opacity-50";

// Case à cocher accessible (base-ui), habillée des tokens de l'artefact.
export function Checkbox({
  checked,
  onCheckedChange,
  children,
  value,
}: {
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  children: ReactNode;
  value?: string;
}) {
  return (
    // biome-ignore lint/a11y/noLabelWithoutControl: le contrôle est la case base-ui imbriquée.
    <label className="flex cursor-pointer items-start gap-2.5">
      <BaseCheckbox.Root
        checked={checked}
        onCheckedChange={onCheckedChange ? (c) => onCheckedChange(c) : undefined}
        value={value}
        className={boxClass}
      >
        <BaseCheckbox.Indicator>
          <Icon icon={Check} className="h-3.5 w-3.5" />
        </BaseCheckbox.Indicator>
      </BaseCheckbox.Root>
      <span className="min-w-0 [overflow-wrap:anywhere]">{children}</span>
    </label>
  );
}

// Groupe de cases à cocher : légende, aide et erreur reliées au groupe.
export function CheckboxGroup<V extends string>({
  legend,
  value,
  onValueChange,
  options,
  hint,
  error,
  className,
}: CheckboxGroupProps<V>) {
  const id = useId();
  const hintId = hint ? `${id}-aide` : undefined;
  const errorId = error ? `${id}-erreur` : undefined;
  return (
    <div className={cn("min-w-0", className)}>
      <p id={`${id}-legende`} className="mb-1 font-bold">
        {legend}
      </p>
      {hint ? (
        <p id={hintId} className="mb-2 text-sm text-muted">
          {hint}
        </p>
      ) : null}
      <BaseCheckboxGroup
        value={value}
        onValueChange={(v) => onValueChange(v as V[])}
        aria-labelledby={`${id}-legende`}
        aria-describedby={[hintId, errorId].filter(Boolean).join(" ") || undefined}
        className="flex flex-col gap-2.5"
      >
        {options.map((o) => (
          <Checkbox key={o.value} value={o.value}>
            {o.label}
            {o.description ? (
              <span className="mt-0.5 block text-sm text-muted">{o.description}</span>
            ) : null}
          </Checkbox>
        ))}
      </BaseCheckboxGroup>
      {error ? (
        <p id={errorId} className="mt-1 flex items-start gap-1.5 text-sm font-bold text-danger">
          <Icon icon={CircleAlert} className="mt-0.5 h-4 w-4" />
          <span>{error}</span>
        </p>
      ) : null}
    </div>
  );
}
