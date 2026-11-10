import { Collapsible } from "@base-ui/react/collapsible";
import { ChevronDown, Download, FileSpreadsheet } from "lucide-react";
import { type DragEvent, useEffect, useId, useRef, useState } from "react";
import { cn } from "~/lib/cn";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

interface FileStepProps {
  busy: boolean;
  errors: string[];
  onFile: (file: File) => void;
  onTemplate: () => void;
  templateBusy: boolean;
}

// Étape 1 : explications, modèle et dépôt du fichier CSV.
export function FileStep({ busy, errors, onFile, onTemplate, templateBusy }: FileStepProps) {
  const inputId = useId();
  const errorId = `${inputId}-erreur`;
  const input = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);

  useEffect(() => {
    if (errors.length > 0) input.current?.focus();
  }, [errors]);

  const pick = (files: FileList | null) => {
    const file = files?.[0];
    if (file && !busy) onFile(file);
  };
  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragging(false);
    pick(e.dataTransfer.files);
  };

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <p>
          Déposez le fichier de vos membres, associez ses colonnes aux champs de Frigg, vérifiez
          l'aperçu puis confirmez. <strong>Seul le format CSV est accepté.</strong>
        </p>
        <Collapsible.Root>
          <Collapsible.Trigger className="group flex cursor-pointer items-center gap-2 font-bold">
            <Icon
              icon={ChevronDown}
              className="h-4 w-4 transition-transform motion-reduce:transition-none group-data-[panel-open]:rotate-180"
            />
            Comment enregistrer mon tableur en CSV ?
          </Collapsible.Trigger>
          <Collapsible.Panel>
            <ul className="mt-2 list-disc space-y-1 pl-6 text-sm">
              <li>Excel : Fichier › Enregistrer sous › CSV UTF-8 (délimité par des virgules).</li>
              <li>LibreOffice : Fichier › Enregistrer sous › Texte CSV.</li>
            </ul>
          </Collapsible.Panel>
        </Collapsible.Root>
        <ul className="list-disc space-y-1 pl-6 text-sm text-muted">
          <li>Chaque ligne doit indiquer le prénom, le nom et la date de naissance.</li>
          <li>Dates acceptées : 03/04/2012, 3-4-2012, 03.04.2012 ou 2012-04-03.</li>
          <li>Les fiches importées sont créées « Sans compte » et aucun e-mail n'est envoyé.</li>
          <li>Les personnes déjà présentes dans Frigg sont ignorées.</li>
        </ul>
        <Button variant="secondary" size="sm" onClick={onTemplate} aria-disabled={templateBusy}>
          <Icon icon={Download} className="h-4 w-4" />
          Télécharger le modèle
        </Button>
      </div>

      {/* biome-ignore lint/a11y/noStaticElementInteractions: le glisser-déposer complète le champ fichier, qui reste le moyen d'accès au clavier. */}
      {/* biome-ignore lint/a11y/noNoninteractiveElementInteractions: idem. */}
      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className={cn(
          "flex flex-col items-center gap-3 rounded-lg border-2 border-dashed border-line-strong bg-surface p-8 text-center",
          dragging && "border-brand bg-brand/5",
          errors.length > 0 && "border-danger",
        )}
      >
        <Icon icon={FileSpreadsheet} className="h-8 w-8 text-muted" />
        <label htmlFor={inputId} className="font-bold">
          Fichier CSV
        </label>
        <p className="text-sm text-muted">Glissez le fichier ici ou choisissez-le.</p>
        <input
          ref={input}
          id={inputId}
          type="file"
          accept=".csv,text/csv"
          aria-describedby={errors.length > 0 ? errorId : undefined}
          aria-invalid={errors.length > 0 ? true : undefined}
          aria-busy={busy}
          disabled={busy}
          onChange={(e) => {
            pick(e.target.files);
            e.target.value = "";
          }}
          className="max-w-full cursor-pointer text-sm file:mr-3 file:cursor-pointer file:rounded-md file:border file:border-line-strong file:bg-surface file:px-3 file:py-2 file:font-bold"
        />
        {busy ? <p role="status">Lecture du fichier…</p> : null}
      </div>
      {errors.length > 0 ? (
        <div id={errorId}>
          <Alert kind="danger">
            {errors.map((e) => (
              <p key={e}>{e}</p>
            ))}
          </Alert>
        </div>
      ) : null}
    </div>
  );
}
