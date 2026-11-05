import type { FieldDefinition } from "~/api/custom-fields";
import { Select } from "~/ui/Select";

// Champs filtrables d'une liste : oui/non et listes de choix actifs (FR-023).
export function filterableFields(fields: FieldDefinition[] | null | undefined) {
  return (fields ?? []).filter((f) => !f.archived && (f.type === "bool" || f.type === "choice"));
}

// Un filtre par champ, sous la forme « <champ>:<valeur> » (contracts/api.md 011).
export function CustomFieldFilters({
  fields,
  value,
  onChange,
}: {
  fields: FieldDefinition[];
  value: string[];
  onChange: (next: string[]) => void;
}) {
  if (fields.length === 0) return null;
  return (
    <>
      {fields.map((f) => {
        const prefix = `${f.id}:`;
        const current = value.find((v) => v.startsWith(prefix));
        const items =
          f.type === "bool"
            ? [
                { value: "", label: "Tous" },
                { value: `${prefix}true`, label: "Oui" },
                { value: `${prefix}false`, label: "Non" },
              ]
            : [
                { value: "", label: "Tous" },
                ...(f.choices ?? []).map((c) => ({ value: `${prefix}c${c.id}`, label: c.label })),
                { value: `${prefix}empty`, label: "Non renseigné" },
              ];
        return (
          <Select
            key={f.id}
            label={f.label}
            value={current ?? ""}
            items={items}
            onValueChange={(v) => {
              const others = value.filter((x) => !x.startsWith(prefix));
              onChange(v ? [...others, v] : others);
            }}
          />
        );
      })}
    </>
  );
}
