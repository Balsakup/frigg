import { CircleAlert } from "lucide-react";
import { type ComponentProps, useId } from "react";
import { cn } from "~/lib/cn";
import { Icon } from "./Icon";

interface FieldProps extends ComponentProps<"input"> {
  label: string;
  hint?: string;
  error?: string;
}

// Champ de formulaire accessible (artefact › Champs de formulaire) : libellé
// au-dessus, aide sous le champ, erreur sous l'aide, reliées au champ.
export function Field({ label, hint, error, required, className, id, ...input }: FieldProps) {
  const autoId = useId();
  const fieldId = id ?? autoId;
  const hintId = hint ? `${fieldId}-aide` : undefined;
  const errorId = error ? `${fieldId}-erreur` : undefined;
  return (
    <div className={cn("min-w-0", className)}>
      <label htmlFor={fieldId} className="mb-1 block font-bold">
        {label}
        {required ? <span className="font-normal text-muted"> (obligatoire)</span> : null}
      </label>
      <input
        id={fieldId}
        required={required}
        aria-invalid={error ? true : undefined}
        aria-describedby={[hintId, errorId].filter(Boolean).join(" ") || undefined}
        className={cn(
          "h-11 w-full min-w-0 rounded-md border bg-surface px-3",
          error ? "border-danger" : "border-line-strong",
        )}
        {...input}
      />
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

interface TextAreaFieldProps extends ComponentProps<"textarea"> {
  label: string;
  hint?: string;
  error?: string;
}

// Zone de texte accessible, sur le modèle de Field.
export function TextAreaField({
  label,
  hint,
  error,
  required,
  className,
  id,
  ...textarea
}: TextAreaFieldProps) {
  const autoId = useId();
  const fieldId = id ?? autoId;
  const hintId = hint ? `${fieldId}-aide` : undefined;
  const errorId = error ? `${fieldId}-erreur` : undefined;
  return (
    <div className={cn("min-w-0", className)}>
      <label htmlFor={fieldId} className="mb-1 block font-bold">
        {label}
        {required ? <span className="font-normal text-muted"> (obligatoire)</span> : null}
      </label>
      <textarea
        id={fieldId}
        required={required}
        rows={3}
        aria-invalid={error ? true : undefined}
        aria-describedby={[hintId, errorId].filter(Boolean).join(" ") || undefined}
        className={cn(
          "w-full min-w-0 rounded-md border bg-surface px-3 py-2",
          error ? "border-danger" : "border-line-strong",
        )}
        {...textarea}
      />
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
