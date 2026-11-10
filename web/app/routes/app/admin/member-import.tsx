import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import {
  analyzeImport,
  downloadTemplate,
  errorsAt,
  type ImportAnalysis,
  type ImportPreview,
  type ImportReport,
  previewImport,
  rejectedImport,
  runImport,
} from "~/api/memberImport";
import { invalidateMembers } from "~/api/members";
import { isProblem, type Problem } from "~/api/problem";
import { downloadBlob } from "~/members/import/download";
import { FileStep } from "~/members/import/FileStep";
import { ImportSteps, importSteps } from "~/members/import/ImportSteps";
import { MappingStep } from "~/members/import/MappingStep";
import { PreviewStep } from "~/members/import/PreviewStep";
import { ReportStep } from "~/members/import/ReportStep";
import { plural } from "~/members/import/text";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";

export const handle = { crumb: "Importer" };

const UNEXPECTED = "Une erreur inattendue s'est produite.";

function asProblem(err: unknown): Problem {
  return isProblem(err) ? err : { detail: UNEXPECTED };
}

// Erreurs de correspondance regroupées par emplacement (« form.mapping[2] »).
function mappingErrors(problem: Problem | null): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const e of problem?.errors ?? []) {
    if (e.location?.startsWith("form.mapping") && e.message) {
      out[e.location] = [...(out[e.location] ?? []), e.message];
    }
  }
  return out;
}

// Assistant d'import des membres (013) : l'état vit dans la page, le fichier
// reste dans le navigateur et accompagne chaque appel.
export default function MemberImport() {
  const qc = useQueryClient();
  const [step, setStep] = useState(0);
  const [file, setFile] = useState<File | null>(null);
  const [analysis, setAnalysis] = useState<ImportAnalysis | null>(null);
  const [mapping, setMapping] = useState<string[]>([]);
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [report, setReport] = useState<ImportReport | null>(null);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [notice, setNotice] = useState("");
  const [status, setStatus] = useState("");
  const heading = useRef<HTMLHeadingElement>(null);
  const firstRender = useRef(true);

  // À chaque étape, le focus va sur son titre.
  // biome-ignore lint/correctness/useExhaustiveDependencies: déclenché par le changement d'étape.
  useEffect(() => {
    if (firstRender.current) {
      firstRender.current = false;
      return;
    }
    heading.current?.focus();
  }, [step]);

  const go = (next: number) => {
    setProblem(null);
    setStep(next);
  };

  const analyze = useMutation({
    mutationFn: analyzeImport,
    onSuccess: (data, f) => {
      setFile(f);
      setAnalysis(data);
      setMapping((data.suggested ?? []).slice());
      setPreview(null);
      go(1);
    },
    onError: (err) => setProblem(asProblem(err)),
  });

  const showPreview = useMutation({
    mutationFn: () => previewImport(file as File, mapping),
    onSuccess: (data) => {
      setPreview(data);
      if (step !== 2) go(2);
    },
    onError: (err) => {
      const p = asProblem(err);
      // Correspondance devenue invalide (champ archivé après un 409) : retour
      // à l'étape des colonnes, où l'erreur s'affiche sous la bonne colonne.
      if (step === 2 && Object.keys(mappingErrors(p)).length > 0) {
        setNotice("");
        setStep(1);
      }
      setProblem(p);
    },
  });

  const run = useMutation({
    mutationFn: () =>
      runImport(file as File, mapping, preview?.fieldsVersion ?? "", preview?.counts.ready ?? 0),
    onSuccess: async (data) => {
      setReport(data);
      setNotice("");
      go(3);
      setStatus(`${plural(data.created, "membre importé", "membres importés")}.`);
      await invalidateMembers(qc);
      await qc.invalidateQueries({ queryKey: ["get", "/admin/custom-fields"] });
    },
    onError: async (err) => {
      const p = asProblem(err);
      if (p.status === 409) {
        // Champs personnalisés changés : l'aperçu est recalculé sur place.
        setNotice(p.detail ?? "");
        showPreview.mutate();
        return;
      }
      if (Object.keys(mappingErrors(p)).length > 0) {
        // Champ associé archivé ou supprimé : la correspondance est à revoir.
        setNotice("");
        setProblem(p);
        setStep(1);
        return;
      }
      setNotice((p.errors ?? []).map((e) => e.message).join(" ") || p.detail || UNEXPECTED);
    },
  });

  const rejected = useMutation({
    mutationFn: () => rejectedImport(file as File, mapping),
    onSuccess: (blob) => downloadBlob(blob, "lignes-rejetees.csv"),
    onError: (err) => setNotice(asProblem(err).detail ?? UNEXPECTED),
  });

  const template = useMutation({
    mutationFn: downloadTemplate,
    onSuccess: (blob) => downloadBlob(blob, "modele-import-membres.csv"),
    onError: (err) => setProblem(asProblem(err)),
  });

  const restart = () => {
    setFile(null);
    setAnalysis(null);
    setMapping([]);
    setPreview(null);
    setReport(null);
    setNotice("");
    go(0);
  };

  const fileErrors = errorsAt(problem, "form.file");
  const otherError =
    problem && fileErrors.length === 0 && Object.keys(mappingErrors(problem)).length === 0
      ? problem.detail || UNEXPECTED
      : "";

  return (
    <Page
      title="Importer des membres"
      intro="Créez en une fois les fiches de vos membres à partir d'un fichier CSV."
    >
      <p role="status" className="sr-only">
        {status}
      </p>
      <div className="space-y-6">
        <ImportSteps current={step} />
        <section aria-labelledby="import-etape" className="space-y-5">
          <h2
            id="import-etape"
            ref={heading}
            tabIndex={-1}
            className="text-xl font-bold outline-none"
          >
            {step + 1}. {importSteps[step]}
          </h2>
          {otherError ? <Alert kind="danger">{otherError}</Alert> : null}

          {step === 0 ? (
            <FileStep
              busy={analyze.isPending}
              errors={fileErrors}
              onFile={(f) => {
                setProblem(null);
                analyze.mutate(f);
              }}
              onTemplate={() => template.mutate()}
              templateBusy={template.isPending}
            />
          ) : null}

          {step === 1 && analysis && file ? (
            <MappingStep
              fileName={file.name}
              analysis={analysis}
              mapping={mapping}
              onChange={setMapping}
              errors={mappingErrors(problem)}
              busy={showPreview.isPending}
              onBack={restart}
              onPreview={() => {
                setProblem(null);
                showPreview.mutate();
              }}
            />
          ) : null}

          {step === 2 && preview ? (
            <PreviewStep
              key={preview.fieldsVersion}
              preview={preview}
              busy={run.isPending || showPreview.isPending}
              rejectedBusy={rejected.isPending}
              notice={notice}
              onBack={() => {
                setNotice("");
                go(1);
              }}
              onRejected={() => rejected.mutate()}
              onImport={() => {
                setNotice("");
                run.mutate();
              }}
            />
          ) : null}

          {step === 3 && report ? (
            <ReportStep
              report={report}
              rejectedBusy={rejected.isPending}
              onRejected={() => rejected.mutate()}
              onRestart={restart}
            />
          ) : null}
        </section>
      </div>
    </Page>
  );
}
