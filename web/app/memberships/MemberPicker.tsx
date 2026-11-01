import { Combobox } from "@base-ui/react/combobox";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronDown, CircleAlert } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { memberListQuery } from "~/api/members";
import { hasSearchableTerm, SEARCH_MIN_LENGTH } from "~/lib/account-filters";
import { cn } from "~/lib/cn";
import { formatAge } from "~/lib/format";
import { Icon } from "~/ui/Icon";

export interface MemberOption {
  id: number;
  label: string;
  detail: string;
}

interface MemberPickerProps {
  value: MemberOption | null;
  onChange: (value: MemberOption | null) => void;
  error?: string;
}

// Choix d'un membre (création directe d'une adhésion, 006 US6) : Combobox
// base-ui alimenté par list-members (recherche FTS côté serveur), sur le modèle
// de AccountPicker.
export function MemberPicker({ value, onChange, error }: MemberPickerProps) {
  const id = useId();
  const [input, setInput] = useState(value?.label ?? "");
  const [query, setQuery] = useState("");

  useEffect(() => {
    const term = input === (value?.label ?? "") ? "" : input.trim();
    const timer = setTimeout(() => setQuery(term), 250);
    return () => clearTimeout(timer);
  }, [input, value]);

  const searchable = hasSearchableTerm(query);
  const results = useQuery({
    ...memberListQuery({ q: query, noAccount: false, page: 1 }),
    enabled: searchable,
  });
  const found: MemberOption[] = searchable
    ? (results.data?.items ?? []).map((m) => ({
        id: m.id,
        label: `${m.lastName} ${m.firstName}`,
        detail: `${formatAge(m.age)} — ${m.account ? m.account.email : "Sans compte"}`,
      }))
    : [];
  const options = value ? [value, ...found.filter((o) => o.id !== value.id)] : found;

  const typing = input !== (value?.label ?? "") && input.trim() !== "";
  const status = !typing
    ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères du nom du membre.`
    : !hasSearchableTerm(input)
      ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères.`
      : results.isFetching
        ? "Recherche…"
        : `${found.length} membre${found.length > 1 ? "s" : ""}`;
  const errorId = error ? `${id}-erreur` : undefined;

  return (
    <Combobox.Root<MemberOption>
      items={options}
      filter={null}
      value={value}
      onValueChange={(v) => {
        onChange(v ?? null);
        setInput(v?.label ?? "");
      }}
      inputValue={input}
      onInputValueChange={setInput}
      itemToStringLabel={(o) => o.label}
      isItemEqualToValue={(a, b) => a.id === b.id}
      modal={false}
    >
      <div className="min-w-0">
        <label id={`${id}-libelle`} htmlFor={id} className="mb-1 block font-bold">
          Membre <span className="font-normal text-muted">(obligatoire)</span>
        </label>
        <Combobox.InputGroup
          className={cn(
            "flex h-11 w-full min-w-0 items-center rounded-md border bg-surface",
            error ? "border-danger" : "border-line-strong",
          )}
        >
          <Combobox.Input
            id={id}
            aria-labelledby={`${id}-libelle`}
            autoComplete="off"
            onFocus={(e) => e.currentTarget.select()}
            aria-invalid={error ? true : undefined}
            aria-describedby={errorId}
            className="h-full min-w-0 flex-1 rounded-md bg-transparent px-3 outline-none"
          />
          <Combobox.Trigger aria-label="Afficher les membres" className="px-3 text-muted">
            <Icon icon={ChevronDown} className="h-4 w-4" />
          </Combobox.Trigger>
        </Combobox.InputGroup>
        {value ? <p className="mt-1 text-sm text-muted">{value.detail}</p> : null}
        {error ? (
          <p id={errorId} className="mt-1 flex items-start gap-1.5 text-sm font-bold text-danger">
            <Icon icon={CircleAlert} className="mt-0.5 h-4 w-4" />
            <span>{error}</span>
          </p>
        ) : null}
      </div>
      <Combobox.Portal>
        <Combobox.Positioner className="z-[60] outline-none" sideOffset={4}>
          <Combobox.Popup className="min-w-[var(--anchor-width)] max-w-[calc(100vw-2rem)] rounded-md border border-line bg-surface py-1 text-ink shadow-lg outline-none">
            <Combobox.Status className="px-3 py-2 text-sm text-muted">{status}</Combobox.Status>
            <Combobox.List>
              {(item: MemberOption) => (
                <Combobox.Item
                  key={item.id}
                  value={item}
                  className="flex cursor-pointer items-center gap-2 px-3 py-2 outline-none [overflow-wrap:anywhere] data-[highlighted]:bg-brand/10"
                >
                  <Combobox.ItemIndicator className="w-4 text-brand-ink" keepMounted={false}>
                    <Icon icon={Check} className="h-4 w-4" />
                  </Combobox.ItemIndicator>
                  <span>
                    {item.label}
                    <span className="block text-sm text-muted">{item.detail}</span>
                  </span>
                </Combobox.Item>
              )}
            </Combobox.List>
          </Combobox.Popup>
        </Combobox.Positioner>
      </Combobox.Portal>
    </Combobox.Root>
  );
}
