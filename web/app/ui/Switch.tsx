import { Switch as BaseSwitch } from "@base-ui/react/switch";
import { useId } from "react";

interface SwitchProps {
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  /** Interrupteur inactif ; l'explication est reliée par describedBy. */
  disabled?: boolean;
  describedBy?: string;
  error?: string;
}

// Interrupteur accessible (base-ui) avec son libellé.
export function Switch({
  label,
  checked,
  onCheckedChange,
  disabled,
  describedBy,
  error,
}: SwitchProps) {
  const id = useId();
  const errorId = useId();
  const described = [describedBy, error ? errorId : undefined].filter(Boolean).join(" ");
  return (
    <div>
      <div className="flex items-center gap-3">
        <BaseSwitch.Root
          id={id}
          checked={checked}
          disabled={disabled}
          onCheckedChange={(c) => onCheckedChange(c)}
          aria-describedby={described || undefined}
          aria-invalid={error ? true : undefined}
          className="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border border-line-strong bg-ink/10 transition-colors data-[checked]:border-brand-ink data-[checked]:bg-brand-ink data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50"
        >
          <BaseSwitch.Thumb className="m-0.5 block h-[18px] w-[18px] rounded-full bg-surface shadow transition-transform motion-reduce:transition-none data-[checked]:translate-x-5" />
        </BaseSwitch.Root>
        <label
          htmlFor={id}
          className={disabled ? "font-bold text-muted" : "cursor-pointer font-bold"}
        >
          {label}
        </label>
      </div>
      {error ? (
        <p id={errorId} className="mt-1 text-sm font-bold text-danger">
          {error}
        </p>
      ) : null}
    </div>
  );
}
