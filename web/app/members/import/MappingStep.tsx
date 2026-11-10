import { useId } from "react";
import type { ImportAnalysis } from "~/api/memberImport";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Select, type SelectItem } from "~/ui/Select";
import { encodingLabels, plural, separatorLabels } from "./text";

const fieldTypeLabels: Record<string, string> = {
  short_text: "texte",
  long_text: "texte long",
  number: "nombre",
  date: "date",
  bool: "oui/non",
  choice: "liste",
};

interface MappingStepProps {
  fileName: string;
  analysis: ImportAnalysis;
  mapping: string[];
  onChange: (mapping: string[]) => void;
  /** Erreurs du serveur : clé « form.mapping » ou « form.mapping[i] ». */
  errors: Record<string, string[]>;
  busy: boolean;
  onBack: () => void;
  onPreview: () => void;
}

const NONE = "none";

// Étape 2 : une ligne par colonne du fichier et le champ Frigg associé. Le
// contrôle des champs exigés ne sert qu'à l'affichage du bouton : il s'appuie
// sur requiredForImport renvoyé par le serveur, qui reste l'autorité (422).
export function MappingStep({
  fileName,
  analysis,
  mapping,
  onChange,
  errors,
  busy,
  onBack,
  onPreview,
}: MappingStepProps) {
  const reasonId = useId();
  const columns = analysis.columns ?? [];
  const targets = analysis.targets ?? [];
  const labelOf = new Map(columns.map((c) => [c.index, c.label]));
  const takenBy = new Map<string, number>();
  mapping.forEach((t, i) => {
    if (t && !takenBy.has(t)) takenBy.set(t, i);
  });
  const missing = targets.filter((t) => t.requiredForImport && !takenBy.has(t.target));
  const associated = mapping.filter(Boolean).length;
  const global = errors["form.mapping"] ?? [];

  const itemsFor = (index: number): SelectItem<string>[] => [
    { value: NONE, label: "Ne pas importer" },
    ...targets.map((t) => {
      const owner = takenBy.get(t.target);
      const taken = owner !== undefined && owner !== index;
      const custom = t.fieldType != null;
      const notes = [
        custom ? `(${fieldTypeLabels[t.fieldType ?? ""] ?? t.fieldType})` : "",
        taken ? `(colonne « ${labelOf.get(owner) ?? ""} »)` : "",
      ].filter(Boolean);
      return {
        value: t.target,
        label: t.requiredForImport ? `${t.label} *` : t.label,
        disabled: taken,
        note: notes.join(" ") || undefined,
        group: custom ? "Informations complémentaires" : "Fiche",
      };
    }),
  ];

  return (
    <div className="space-y-5">
      <p className="text-sm text-muted">
        <span className="font-bold text-ink">{fileName}</span> ·{" "}
        {plural(analysis.file.rowCount, "ligne")} · encodage{" "}
        {encodingLabels[analysis.file.encoding] ?? analysis.file.encoding} · séparateur{" "}
        {separatorLabels[analysis.file.separator] ?? analysis.file.separator}
      </p>
      <p>
        {associated} colonne{associated > 1 ? "s" : ""} sur {columns.length} associée
        {associated > 1 ? "s" : ""}.
      </p>
      {missing.length > 0 || global.length > 0 ? (
        <div id={reasonId}>
          <Alert kind="warning">
            {missing.map((t) => (
              <p key={t.target}>Associez une colonne au champ « {t.label} ».</p>
            ))}
            {global
              .filter((g) => !missing.some((t) => g.includes(`« ${t.label} »`)))
              .map((g) => (
                <p key={g}>{g}</p>
              ))}
          </Alert>
        </div>
      ) : null}

      <ul className="divide-y divide-line rounded-lg border border-line bg-surface">
        {columns.map((c) => (
          <li
            key={c.index}
            className="grid gap-3 p-4 sm:grid-cols-[1fr_minmax(14rem,1fr)] sm:items-start"
          >
            <div className="min-w-0">
              <p className="font-bold [overflow-wrap:anywhere]">{c.label}</p>
              {(c.samples ?? []).length > 0 ? (
                <p className="text-sm text-muted [overflow-wrap:anywhere]">
                  {(c.samples ?? []).join(" · ")}
                </p>
              ) : (
                <p className="text-sm text-muted">Colonne vide</p>
              )}
            </div>
            <Select
              label={`Champ Frigg pour la colonne ${c.label}`}
              hideLabel
              value={mapping[c.index] || NONE}
              items={itemsFor(c.index)}
              error={(errors[`form.mapping[${c.index}]`] ?? []).join(" ") || undefined}
              onValueChange={(v) => {
                const next = [...mapping];
                next[c.index] = v === NONE ? "" : v;
                onChange(next);
              }}
            />
          </li>
        ))}
      </ul>

      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-between">
        <Button variant="secondary" onClick={onBack}>
          Changer de fichier
        </Button>
        <Button
          onClick={onPreview}
          aria-disabled={busy || missing.length > 0}
          aria-describedby={missing.length > 0 ? reasonId : undefined}
        >
          {busy ? "Calcul de l'aperçu…" : "Voir l'aperçu"}
        </Button>
      </div>
    </div>
  );
}
