import { CircleAlert, Trash2 } from "lucide-react";
import { useId } from "react";
import { AssociationMark } from "~/shell/AssociationMark";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

// Logo de l'association : logo actuel (ou initiales), choix d'un fichier
// montré dans l'aperçu avant l'enregistrement, retrait (contracts/ui-routes.md 010).
export function LogoField({
  name,
  logoUrl,
  pendingName,
  error,
  onSelect,
  onRemove,
}: {
  name: string;
  logoUrl?: string | null;
  pendingName?: string;
  error?: string;
  onSelect: (file: File) => void;
  onRemove?: () => void;
}) {
  const id = useId();
  const hintId = `${id}-aide`;
  const errorId = error ? `${id}-erreur` : undefined;
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-4">
        <AssociationMark name={name} logoUrl={logoUrl} className="h-16 w-16 text-xl" />
        <p className="min-w-0 text-sm text-muted [overflow-wrap:anywhere]">
          {pendingName
            ? `Nouveau logo : ${pendingName} (enregistrez pour l'appliquer).`
            : logoUrl
              ? "Logo actuel."
              : "Aucun logo : les initiales de l'association sont affichées."}
        </p>
      </div>
      <div>
        <label htmlFor={id} className="mb-1 block font-bold">
          Logo
        </label>
        <input
          id={id}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          aria-invalid={error ? true : undefined}
          aria-describedby={[hintId, errorId].filter(Boolean).join(" ")}
          className="block w-full min-w-0 text-sm file:mr-3 file:min-h-10 file:cursor-pointer file:rounded-md file:border file:border-line-strong file:bg-surface file:px-4 file:font-bold file:text-ink"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) onSelect(f);
          }}
        />
        <div className="mt-2 flex flex-wrap gap-2">
          {onRemove ? (
            <Button type="button" variant="tertiary" onClick={onRemove}>
              <Icon icon={Trash2} className="h-5 w-5" />
              Retirer le logo
            </Button>
          ) : null}
        </div>
        <p id={hintId} className="mt-1 text-sm text-muted">
          Choisissez une image PNG, JPEG ou WebP, 1 Mo au plus.
        </p>
        {error ? (
          <p id={errorId} className="mt-1 flex items-start gap-1.5 text-sm font-bold text-danger">
            <Icon icon={CircleAlert} className="mt-0.5 h-4 w-4" />
            <span>{error}</span>
          </p>
        ) : null}
      </div>
    </div>
  );
}
