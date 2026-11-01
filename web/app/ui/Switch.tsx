import { Switch as BaseSwitch } from "@base-ui/react/switch";
import { useId } from "react";

interface SwitchProps {
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

// Interrupteur accessible (base-ui) avec son libellé.
export function Switch({ label, checked, onCheckedChange }: SwitchProps) {
  const id = useId();
  return (
    <div className="flex items-center gap-3">
      <BaseSwitch.Root
        id={id}
        checked={checked}
        onCheckedChange={(c) => onCheckedChange(c)}
        className="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border border-line-strong bg-ink/10 transition-colors data-[checked]:border-brand data-[checked]:bg-brand"
      >
        <BaseSwitch.Thumb className="m-0.5 block h-[18px] w-[18px] rounded-full bg-surface shadow transition-transform motion-reduce:transition-none data-[checked]:translate-x-5" />
      </BaseSwitch.Root>
      <label htmlFor={id} className="cursor-pointer font-bold">
        {label}
      </label>
    </div>
  );
}
