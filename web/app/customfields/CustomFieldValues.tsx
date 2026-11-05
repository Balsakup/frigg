import type { FieldValue } from "~/api/custom-fields";
import { formatDay, formatNumber } from "~/lib/format";

// Valeur lisible d'un champ ; null si non renseigné.
export function displayValue(f: FieldValue): string | null {
  switch (f.type) {
    case "bool":
      return f.checked ? "Oui" : "Non";
    case "number":
      return f.number != null ? formatNumber(f.number) : null;
    case "date":
      return f.date ? formatDay(f.date) : null;
    case "choice":
      return f.choices?.find((c) => c.id === f.choiceId)?.label ?? null;
    default:
      return f.text ?? null;
  }
}

// Champs personnalisés en lecture (contracts/ui-routes.md 011 › Composants).
export function CustomFieldValues({ fields }: { fields: FieldValue[] }) {
  if (fields.length === 0) return null;
  return (
    <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-[minmax(0,14rem)_1fr]">
      {fields.map((f) => {
        const value = displayValue(f);
        return (
          <div key={f.id} className="contents">
            <dt className="font-bold [overflow-wrap:anywhere]">{f.label}</dt>
            <dd className="whitespace-pre-line [overflow-wrap:anywhere]">
              {value ?? <span className="text-muted">Non renseigné</span>}
            </dd>
          </div>
        );
      })}
    </dl>
  );
}
