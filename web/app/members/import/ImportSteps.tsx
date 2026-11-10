import { Check } from "lucide-react";
import { cn } from "~/lib/cn";
import { Icon } from "~/ui/Icon";

export const importSteps = ["Fichier", "Colonnes", "Aperçu", "Rapport"] as const;

// Étapes de l'assistant d'import ; l'étape courante porte aria-current.
export function ImportSteps({ current }: { current: number }) {
  return (
    <ol aria-label="Étapes de l'import" className="flex flex-wrap gap-x-4 gap-y-2 text-sm">
      {importSteps.map((label, i) => {
        const done = i < current;
        const active = i === current;
        return (
          <li
            key={label}
            aria-current={active ? "step" : undefined}
            className={cn(
              "flex items-center gap-2",
              active ? "font-bold text-brand-ink" : "text-muted",
            )}
          >
            <span
              className={cn(
                "grid h-6 w-6 place-items-center rounded-full border text-xs",
                active && "border-brand bg-brand text-on-brand",
                done && "border-ok text-ok",
                !active && !done && "border-line-strong",
              )}
            >
              {done ? <Icon icon={Check} className="h-3.5 w-3.5" /> : i + 1}
            </span>
            {label}
            {done ? <span className="sr-only"> (terminée)</span> : null}
          </li>
        );
      })}
    </ol>
  );
}
