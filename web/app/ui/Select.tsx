import { Select as BaseSelect } from "@base-ui/react/select";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "~/lib/cn";
import { Icon } from "./Icon";

export interface SelectItem<V extends string> {
  value: V;
  label: string;
}

interface SelectProps<V extends string> {
  label: string;
  value: V;
  onValueChange: (value: V) => void;
  items: SelectItem<V>[];
  name?: string;
  className?: string;
}

// Liste déroulante accessible (base-ui) habillée des tokens de l'artefact.
export function Select<V extends string>({
  label,
  value,
  onValueChange,
  items,
  name,
  className,
}: SelectProps<V>) {
  return (
    <BaseSelect.Root
      items={items}
      value={value}
      name={name}
      onValueChange={(v) => {
        if (v !== null) onValueChange(v as V);
      }}
    >
      <div className={cn("min-w-0", className)}>
        <BaseSelect.Label className="mb-1 block font-bold">{label}</BaseSelect.Label>
        <BaseSelect.Trigger className="flex h-11 w-full min-w-0 items-center justify-between gap-2 rounded-md border border-line-strong bg-surface px-3 text-left">
          <BaseSelect.Value className="truncate" />
          <BaseSelect.Icon className="text-muted">
            <Icon icon={ChevronDown} className="h-4 w-4" />
          </BaseSelect.Icon>
        </BaseSelect.Trigger>
      </div>
      <BaseSelect.Portal>
        <BaseSelect.Positioner
          className="z-[60] outline-none"
          sideOffset={4}
          alignItemWithTrigger={false}
        >
          <BaseSelect.Popup className="min-w-[var(--anchor-width)] rounded-md border border-line bg-surface py-1 text-ink shadow-lg outline-none">
            <BaseSelect.List>
              {items.map((item) => (
                <BaseSelect.Item
                  key={item.value}
                  value={item.value}
                  className="flex cursor-pointer items-center gap-2 px-3 py-2 outline-none data-[highlighted]:bg-brand/10"
                >
                  <BaseSelect.ItemIndicator className="w-4 text-brand-ink" keepMounted={false}>
                    <Icon icon={Check} className="h-4 w-4" />
                  </BaseSelect.ItemIndicator>
                  <BaseSelect.ItemText>{item.label}</BaseSelect.ItemText>
                </BaseSelect.Item>
              ))}
            </BaseSelect.List>
          </BaseSelect.Popup>
        </BaseSelect.Positioner>
      </BaseSelect.Portal>
    </BaseSelect.Root>
  );
}
