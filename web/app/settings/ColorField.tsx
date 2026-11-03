import { CircleAlert } from "lucide-react";
import { useId } from "react";
import { Icon } from "~/ui/Icon";

// Couleur saisie au sélecteur ou au clavier (code #rrggbb) : un seul libellé,
// deux contrôles synchronisés (contracts/ui-routes.md 010).
export function ColorField({
  label,
  value,
  onChange,
  error,
  hint,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  hint?: string;
}) {
  const id = useId();
  const hintId = hint ? `${id}-aide` : undefined;
  const errorId = error ? `${id}-erreur` : undefined;
  const valid = /^#[0-9a-fA-F]{6}$/.test(value);
  return (
    <div className="min-w-0">
      <label htmlFor={id} className="mb-1 block font-bold">
        {label}
      </label>
      <div className="flex items-center gap-2">
        <input
          type="color"
          aria-label={`${label} : sélecteur`}
          value={valid ? value.toLowerCase() : "#000000"}
          onChange={(e) => onChange(e.target.value)}
          className="h-11 w-14 shrink-0 cursor-pointer rounded-md border border-line-strong bg-surface p-1"
        />
        <input
          id={id}
          value={value}
          maxLength={7}
          spellCheck={false}
          autoComplete="off"
          aria-invalid={error ? true : undefined}
          aria-describedby={[hintId, errorId].filter(Boolean).join(" ") || undefined}
          onChange={(e) => onChange(e.target.value.trim())}
          className={`h-11 w-32 min-w-0 rounded-md border bg-surface px-3 font-mono ${error ? "border-danger" : "border-line-strong"}`}
        />
      </div>
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
  );
}
