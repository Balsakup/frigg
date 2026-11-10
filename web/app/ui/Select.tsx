import { Select as BaseSelect } from "@base-ui/react/select";
import { Check, ChevronDown, CircleAlert } from "lucide-react";
import { useId } from "react";
import { cn } from "~/lib/cn";
import { Icon } from "./Icon";

export interface SelectItem<V extends string> {
  value: V;
  label: string;
  /** Option affichée mais non choisissable. */
  disabled?: boolean;
  /** Mention secondaire après le libellé (« (colonne « Mail ») »). */
  note?: string;
  /** Groupe d'options ; les options d'un même groupe se suivent. */
  group?: string;
}

interface SelectProps<V extends string> {
  label: string;
  value: V;
  onValueChange: (value: V) => void;
  items: SelectItem<V>[];
  name?: string;
  className?: string;
  /** Message d'erreur affiché sous la liste et relié au déclencheur. */
  error?: string;
  /** Libellé réservé aux lecteurs d'écran (le contexte visuel suffit). */
  hideLabel?: boolean;
}

// Liste déroulante accessible (base-ui) habillée des tokens de l'artefact.
export function Select<V extends string>({
  label,
  value,
  onValueChange,
  items,
  name,
  className,
  error,
  hideLabel,
}: SelectProps<V>) {
  const errorId = `${useId()}-erreur`;
  const groups: { label?: string; items: SelectItem<V>[] }[] = [];
  for (const item of items) {
    const last = groups.at(-1);
    if (last && last.label === item.group) last.items.push(item);
    else groups.push({ label: item.group, items: [item] });
  }
  const renderItem = (item: SelectItem<V>) => (
    <BaseSelect.Item
      key={item.value}
      value={item.value}
      disabled={item.disabled}
      className="flex cursor-pointer items-center gap-2 px-3 py-2 outline-none data-[disabled]:cursor-not-allowed data-[disabled]:text-muted data-[highlighted]:bg-brand/10"
    >
      <BaseSelect.ItemIndicator className="w-4 text-brand-ink" keepMounted={false}>
        <Icon icon={Check} className="h-4 w-4" />
      </BaseSelect.ItemIndicator>
      <BaseSelect.ItemText>{item.label}</BaseSelect.ItemText>
      {item.note ? <span className="text-sm text-muted">{item.note}</span> : null}
    </BaseSelect.Item>
  );
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
        <BaseSelect.Label className={hideLabel ? "sr-only" : "mb-1 block font-bold"}>
          {label}
        </BaseSelect.Label>
        <BaseSelect.Trigger
          aria-describedby={error ? errorId : undefined}
          aria-invalid={error ? true : undefined}
          className={cn(
            "flex h-11 w-full min-w-0 items-center justify-between gap-2 rounded-md border border-line-strong bg-surface px-3 text-left",
            error && "border-danger",
          )}
        >
          <BaseSelect.Value className="truncate" />
          <BaseSelect.Icon className="text-muted">
            <Icon icon={ChevronDown} className="h-4 w-4" />
          </BaseSelect.Icon>
        </BaseSelect.Trigger>
        {error ? (
          <p id={errorId} className="mt-1 flex items-start gap-1.5 text-sm font-bold text-danger">
            <Icon icon={CircleAlert} className="mt-0.5 h-4 w-4" />
            <span>{error}</span>
          </p>
        ) : null}
      </div>
      <BaseSelect.Portal>
        <BaseSelect.Positioner
          className="z-[60] outline-none"
          sideOffset={4}
          alignItemWithTrigger={false}
        >
          <BaseSelect.Popup className="min-w-[var(--anchor-width)] rounded-md border border-line bg-surface py-1 text-ink shadow-lg outline-none">
            <BaseSelect.List>
              {groups.map((g) =>
                g.label ? (
                  <BaseSelect.Group key={g.label}>
                    <BaseSelect.GroupLabel className="px-3 pt-2 pb-1 text-sm font-bold text-muted">
                      {g.label}
                    </BaseSelect.GroupLabel>
                    {g.items.map(renderItem)}
                  </BaseSelect.Group>
                ) : (
                  g.items.map(renderItem)
                ),
              )}
            </BaseSelect.List>
          </BaseSelect.Popup>
        </BaseSelect.Positioner>
      </BaseSelect.Portal>
    </BaseSelect.Root>
  );
}
