import { Download } from "lucide-react";
import { Link } from "react-router";
import type { ImportReport } from "~/api/memberImport";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { missingText } from "./PreviewStep";
import { plural } from "./text";

interface ReportStepProps {
  report: ImportReport;
  rejectedBusy: boolean;
  onRejected: () => void;
  onRestart: () => void;
}

// Étape 4 : rapport de l'import confirmé.
export function ReportStep({ report, rejectedBusy, onRejected, onRestart }: ReportStepProps) {
  const rejected = report.errors + report.duplicates;
  return (
    <div className="space-y-5">
      <Alert kind="success">{plural(report.created, "membre importé", "membres importés")}.</Alert>
      <ul className="list-disc space-y-1 pl-6">
        <li>{plural(report.errors, "ligne en erreur", "lignes en erreur")}</li>
        <li>
          {plural(report.duplicates, "doublon")}
          {report.newDuplicates > 0
            ? `, dont ${plural(report.newDuplicates, "personne ajoutée", "personnes ajoutées")} entre-temps`
            : ""}
        </li>
        <li>{plural(report.incompleteCount, "fiche à compléter", "fiches à compléter")}</li>
      </ul>
      {report.incompleteCount > 0 ? (
        <p className="text-sm text-muted">{missingText(report)}</p>
      ) : null}
      <div className="flex flex-col gap-2 sm:flex-row">
        <Link to="/admin/membres?rattachement=sans-compte" className={buttonClasses({})}>
          Voir les membres sans compte
        </Link>
        {rejected > 0 ? (
          <Button variant="secondary" onClick={onRejected} aria-disabled={rejectedBusy}>
            <Icon icon={Download} className="h-4 w-4" />
            Télécharger les lignes rejetées
          </Button>
        ) : null}
        <Button variant="secondary" onClick={onRestart}>
          Importer un autre fichier
        </Button>
      </div>
    </div>
  );
}
