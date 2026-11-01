import { Combobox } from "@base-ui/react/combobox";
import { useQuery } from "@tanstack/react-query";
import { Check, ChevronDown, CircleAlert } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { accountListQuery } from "~/api/accounts";
import { hasSearchableTerm, SEARCH_MIN_LENGTH } from "~/lib/account-filters";
import { cn } from "~/lib/cn";
import { Icon } from "~/ui/Icon";

export interface AccountOption {
  id: number | null;
  label: string;
  disabled?: boolean;
}

export const NO_ACCOUNT: AccountOption = { id: null, label: "Sans compte" };

// Texte du champ : « Sans compte » est affiché en placeholder, la saisie part
// donc d'un champ vide.
const shown = (o: AccountOption) => (o.id === null ? "" : o.label);

interface AccountPickerProps {
  value: AccountOption;
  onChange: (value: AccountOption) => void;
  error?: string;
  hint?: string;
}

// Choix du compte de rattachement (contracts/ui-routes.md 005) : Combobox
// base-ui alimenté par list-accounts (recherche FTS côté serveur, aucun
// filtrage côté client), avec l'option fixe « Sans compte ».
export function AccountPicker({ value, onChange, error, hint }: AccountPickerProps) {
  const id = useId();
  const [input, setInput] = useState(shown(value));
  const [query, setQuery] = useState("");

  // Recherche après 250 ms sans frappe ; la saisie qui reprend le libellé
  // choisi n'est pas une recherche.
  useEffect(() => {
    const term = input === shown(value) ? "" : input.trim();
    const timer = setTimeout(() => setQuery(term), 250);
    return () => clearTimeout(timer);
  }, [input, value]);

  const searchable = hasSearchableTerm(query);
  const results = useQuery({
    ...accountListQuery({ q: query, role: "", status: "", page: 1 }),
    enabled: searchable,
  });
  const found: AccountOption[] = searchable
    ? (results.data?.items ?? []).map((a) => ({
        id: a.id,
        label: a.email,
        disabled: a.status === "disabled",
      }))
    : [];
  const options = [NO_ACCOUNT, ...found.filter((o) => o.id !== value.id)];
  if (value.id !== null) options.splice(1, 0, value);

  const typing = input !== shown(value) && input.trim() !== "";
  const status = !typing
    ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères de l'adresse du compte.`
    : !hasSearchableTerm(input)
      ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères.`
      : results.isFetching
        ? "Recherche…"
        : `${found.length} compte${found.length > 1 ? "s" : ""}`;

  const hintId = hint ? `${id}-aide` : undefined;
  const errorId = error ? `${id}-erreur` : undefined;

  return (
    <Combobox.Root<AccountOption>
      items={options}
      filter={null}
      value={value}
      onValueChange={(v) => {
        const next = v ?? NO_ACCOUNT;
        onChange(next);
        setInput(shown(next));
      }}
      inputValue={input}
      onInputValueChange={setInput}
      itemToStringLabel={shown}
      isItemEqualToValue={(a, b) => a.id === b.id}
      modal={false}
    >
      <div className="min-w-0">
        <label id={`${id}-libelle`} htmlFor={id} className="mb-1 block font-bold">
          Compte de rattachement
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
            placeholder="Sans compte"
            // Un compte déjà choisi est sélectionné au focus : la saisie le remplace.
            onFocus={(e) => e.currentTarget.select()}
            aria-invalid={error ? true : undefined}
            aria-describedby={[hintId, errorId].filter(Boolean).join(" ") || undefined}
            className="h-full min-w-0 flex-1 rounded-md bg-transparent px-3 outline-none placeholder:text-muted"
          />
          <Combobox.Trigger aria-label="Afficher les comptes" className="px-3 text-muted">
            <Icon icon={ChevronDown} className="h-4 w-4" />
          </Combobox.Trigger>
        </Combobox.InputGroup>
        {hint ? (
          <p id={hintId} className="mt-1 text-sm text-muted">
            {hint}
          </p>
        ) : null}
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
              {(item: AccountOption) => (
                <Combobox.Item
                  key={item.id ?? "none"}
                  value={item}
                  className="flex cursor-pointer items-center gap-2 px-3 py-2 outline-none [overflow-wrap:anywhere] data-[highlighted]:bg-brand/10"
                >
                  <Combobox.ItemIndicator className="w-4 text-brand-ink" keepMounted={false}>
                    <Icon icon={Check} className="h-4 w-4" />
                  </Combobox.ItemIndicator>
                  <span>
                    {item.label}
                    {item.disabled ? <span className="text-muted"> (désactivé)</span> : null}
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
