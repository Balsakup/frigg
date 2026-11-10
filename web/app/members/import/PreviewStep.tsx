import { ChevronLeft, ChevronRight, Download } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import type { ImportPreview, ImportRow } from "~/api/memberImport";
import { Alert } from "~/ui/Alert";
import { Badge, type BadgeTone } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { RadioGroup } from "~/ui/RadioGroup";
import { Table, TableCell, TableRow } from "~/ui/Table";
import { plural } from "./text";

type Filter = "all" | "ready" | "error" | "duplicate";

const PAGE_SIZE = 50;

const statusBadges: Record<ImportRow["status"], { label: string; tone: BadgeTone }> = {
  ready: { label: "À importer", tone: "success" },
  error: { label: "En erreur", tone: "danger" },
  duplicate: { label: "Doublon", tone: "warning" },
};

interface PreviewStepProps {
  preview: ImportPreview;
  busy: boolean;
  rejectedBusy: boolean;
  notice: string;
  onBack: () => void;
  onRejected: () => void;
  onImport: () => void;
}

export function missingText(preview: Pick<ImportPreview, "missingRequired" | "incompleteCount">) {
  const detail = (preview.missingRequired ?? []).map((m) => `${m.label} (${m.count})`).join(", ");
  return `${plural(preview.incompleteCount, "fiche importée", "fiches importées")} n'${
    preview.incompleteCount > 1 ? "auront" : "aura"
  } pas de valeur pour un champ obligatoire : ${detail}.`;
}

export function RowDetail({ row }: { row: ImportRow }) {
  if (row.duplicateOfMember) {
    return (
      <>
        Déjà dans Frigg :{" "}
        <Link
          to={`/admin/membres/${row.duplicateOfMember.id}`}
          className="text-brand-ink underline"
        >
          {row.duplicateOfMember.label}
        </Link>
      </>
    );
  }
  if (row.duplicateOfLine) return <>Même personne qu'à la ligne {row.duplicateOfLine}</>;
  const errors = row.errors ?? [];
  if (errors.length === 0) return null;
  return (
    <ul className="space-y-0.5">
      {errors.map((e) => (
        <li key={`${e.column}-${e.message}`}>
          {e.columnLabel ? <span className="font-bold">{e.columnLabel} : </span> : null}
          {e.message}
        </li>
      ))}
    </ul>
  );
}

// Étape 3 : compteurs, filtre, lignes et confirmation.
export function PreviewStep({
  preview,
  busy,
  rejectedBusy,
  notice,
  onBack,
  onRejected,
  onImport,
}: PreviewStepProps) {
  const { counts } = preview;
  const rows = preview.rows ?? [];
  const [filter, setFilter] = useState<Filter>(counts.errors > 0 ? "error" : "all");
  const [page, setPage] = useState(1);
  const shown = filter === "all" ? rows : rows.filter((r) => r.status === filter);
  const pageCount = Math.max(1, Math.ceil(shown.length / PAGE_SIZE));
  const current = Math.min(page, pageCount);
  const pageRows = shown.slice((current - 1) * PAGE_SIZE, current * PAGE_SIZE);
  const rejected = counts.errors + counts.duplicates;

  return (
    <div className="space-y-5">
      {notice ? (
        <Alert kind="warning" role="alert">
          {notice}
        </Alert>
      ) : null}
      <div aria-live="polite" className="flex flex-wrap gap-2">
        <Badge tone="success">{counts.ready.toLocaleString("fr-FR")} à importer</Badge>
        <Badge tone="danger">{counts.errors.toLocaleString("fr-FR")} en erreur</Badge>
        <Badge tone="warning">{plural(counts.duplicates, "doublon")}</Badge>
      </div>
      {preview.incompleteCount > 0 ? (
        <Alert kind="info">{missingText(preview)} Vous pourrez les compléter ensuite.</Alert>
      ) : null}

      <RadioGroup<Filter>
        legend="Afficher"
        value={filter}
        onValueChange={(v) => {
          setFilter(v);
          setPage(1);
        }}
        options={[
          { value: "all", label: `Toutes (${rows.length})` },
          { value: "ready", label: `À importer (${counts.ready})` },
          { value: "error", label: `En erreur (${counts.errors})` },
          { value: "duplicate", label: `Doublons (${counts.duplicates})` },
        ]}
      />

      <div className="rounded-lg border border-line bg-surface">
        <Table
          caption="Lignes du fichier"
          columns={[
            { key: "line", label: "Ligne" },
            { key: "last", label: "Nom" },
            { key: "first", label: "Prénom" },
            { key: "birth", label: "Naissance" },
            { key: "status", label: "État" },
            { key: "detail", label: "Détail" },
          ]}
        >
          {pageRows.map((r) => (
            <TableRow key={r.line}>
              <TableCell>{r.line}</TableCell>
              <TableCell className="[overflow-wrap:anywhere]">{r.lastName}</TableCell>
              <TableCell className="[overflow-wrap:anywhere]">{r.firstName}</TableCell>
              <TableCell className="whitespace-nowrap">{r.birthDate}</TableCell>
              <TableCell>
                <Badge tone={statusBadges[r.status].tone}>{statusBadges[r.status].label}</Badge>
              </TableCell>
              <TableCell className="min-w-56 text-sm">
                <RowDetail row={r} />
              </TableCell>
            </TableRow>
          ))}
        </Table>
        {shown.length === 0 ? <p className="px-5 py-4 text-muted">Aucune ligne.</p> : null}
        {pageCount > 1 ? (
          <nav
            aria-label="Pagination de l'aperçu"
            className="flex flex-wrap items-center justify-between gap-3 border-t border-line px-5 py-3"
          >
            <p aria-current="page">
              Page {current} sur {pageCount}
            </p>
            <div className="flex gap-2">
              {current > 1 ? (
                <Button variant="secondary" size="sm" onClick={() => setPage(current - 1)}>
                  <Icon icon={ChevronLeft} className="h-4 w-4" />
                  Précédente
                </Button>
              ) : null}
              {current < pageCount ? (
                <Button variant="secondary" size="sm" onClick={() => setPage(current + 1)}>
                  Suivante
                  <Icon icon={ChevronRight} className="h-4 w-4" />
                </Button>
              ) : null}
            </div>
          </nav>
        ) : null}
      </div>

      {counts.ready === 0 ? (
        <Alert kind="warning">Aucune ligne à importer : corrigez le fichier ou les colonnes.</Alert>
      ) : null}
      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-between">
        <div className="flex flex-col-reverse gap-2 sm:flex-row">
          <Button variant="secondary" onClick={onBack}>
            Modifier les colonnes
          </Button>
          {rejected > 0 ? (
            <Button variant="secondary" onClick={onRejected} aria-disabled={rejectedBusy}>
              <Icon icon={Download} className="h-4 w-4" />
              Télécharger les lignes rejetées
            </Button>
          ) : null}
        </div>
        {counts.ready > 0 ? (
          <Button onClick={onImport} aria-disabled={busy}>
            {busy ? "Import en cours…" : `Importer ${plural(counts.ready, "membre")}`}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
