import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { Eye, Send } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useBlocker, useNavigate } from "react-router";
import {
  type Attachment,
  type Audience,
  audienceQuery,
  changedAudience,
  invalidateCommunications,
  normalizeTarget,
  type Preset,
  presetOptions,
  type Situation,
  situationLabels,
} from "~/api/communications";
import { membershipFeesQuery } from "~/api/membership-fees";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { seasonListQuery } from "~/api/seasons";
import { AttachmentsField } from "~/communications/AttachmentsField";
import { AudienceSummary } from "~/communications/AudienceSummary";
import { MessagePreview, type PreviewContent } from "~/communications/MessagePreview";
import { formatBytes } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { CheckboxGroup } from "~/ui/Checkbox";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Field, TextAreaField } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { RadioGroup } from "~/ui/RadioGroup";
import { Select } from "~/ui/Select";

export const handle = { crumb: "Nouvelle communication" };

export async function clientLoader() {
  await queryClient.ensureQueryData(seasonListQuery()).catch(() => undefined);
  return null;
}

const SUBJECT_MAX = 150;
const BODY_MAX = 10000;

type Errors = Record<string, string | undefined>;

const situationOptions = (Object.keys(situationLabels) as Situation[]).map((value) => ({
  value,
  label: situationLabels[value],
}));

function newDraftKey() {
  return crypto.randomUUID();
}

// Préparation et envoi d'une communication aux adhérents d'une saison
// (contracts/ui-routes.md 009 : US1 à US3).
export default function CommunicationNew() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [draftKey] = useState(newDraftKey);
  const seasons = useQuery(seasonListQuery());
  const eligible = (seasons.data?.items ?? []).filter((s) => s.status !== "draft");
  const [seasonId, setSeasonId] = useState<number | null>(null);
  const season =
    eligible.find((s) => s.id === seasonId) ?? eligible.find((s) => s.isCurrent) ?? eligible[0];
  const fees = useQuery({ ...membershipFeesQuery(season?.id ?? 0), enabled: season !== undefined });

  const [preset, setPreset] = useState<Preset>("unpaid");
  const [custom, setCustom] = useState<Situation[]>(["announced", "due"]);
  const [feeIds, setFeeIds] = useState<string[]>([]);
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [uploading, setUploading] = useState(false);
  const [errors, setErrors] = useState<Errors>({});
  const [confirming, setConfirming] = useState(false);
  const [changed, setChanged] = useState<{ before: number; after: number } | null>(null);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [preview, setPreview] = useState<PreviewContent | null>(null);
  const [sent, setSent] = useState(false);
  // Lu par le garde-fou de navigation dans la même tâche que l'envoi, avant
  // le rendu suivant.
  const sentRef = useRef(false);

  const situations =
    preset === "custom"
      ? custom
      : (presetOptions.find((p) => p.value === preset)?.situations ?? []);
  const target = season
    ? normalizeTarget({
        seasonId: season.id,
        situations,
        feeIds: preset === "custom" ? feeIds.map(Number) : [],
      })
    : null;
  const audience = useQuery({
    ...audienceQuery(target ?? { seasonId: 0, situations: [], feeIds: [] }),
    enabled: target !== null && target.situations.length > 0,
    placeholderData: keepPreviousData,
  });
  const current: Audience | undefined = target?.situations.length ? audience.data : undefined;
  const recipients = current?.recipientsCount ?? 0;

  const send = $api.useMutation("post", "/admin/communications");
  const previewMutation = $api.useMutation("post", "/admin/communications/preview");

  const dirty = !sent && (subject !== "" || body !== "" || attachments.length > 0);
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      dirty && !sentRef.current && currentLocation.pathname !== nextLocation.pathname,
  );
  useEffect(() => {
    if (!dirty) return;
    const warn = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  const onAttachments = useCallback((list: Attachment[], busy: boolean) => {
    setAttachments(list);
    setUploading(busy);
  }, []);

  const checkMessage = () => {
    const next: Errors = {
      subject: subject.trim() ? undefined : "Saisissez l'objet du message.",
      body: body.trim() ? undefined : "Saisissez le message.",
      situations:
        preset === "custom" && custom.length === 0
          ? "Choisissez au moins une situation de paiement."
          : undefined,
    };
    setErrors(next);
    return !Object.values(next).some(Boolean);
  };

  const openPreview = async () => {
    if (!season || !checkMessage()) return;
    setPreview(null);
    setPreviewOpen(true);
    try {
      const p = await previewMutation.mutateAsync({ body: { seasonId: season.id, subject, body } });
      setPreview({ subject: p.subject, html: p.html });
    } catch (error) {
      setPreviewOpen(false);
      if (!isProblem(error)) throw error;
      applyErrors(error.errors ?? [], error.detail);
    }
  };

  const applyErrors = (list: { location?: string; message?: string }[], detail?: string) => {
    const next: Errors = {};
    for (const e of list) {
      const loc = e.location ?? "";
      if (loc === "body.subject") next.subject = e.message;
      else if (loc === "body.body") next.body = e.message;
      else if (loc.startsWith("body.target")) next.target = e.message;
      else if (loc === "body.draftKey") next.attachments = e.message;
      else next.root = e.message;
    }
    if (Object.keys(next).length === 0)
      next.root = detail ?? "Une erreur inattendue s'est produite.";
    setErrors(next);
  };

  const askConfirm = () => {
    if (!checkMessage()) return;
    if (recipients === 0) {
      setErrors({ target: "Aucun destinataire ne correspond à cette cible." });
      return;
    }
    setChanged(null);
    setConfirming(true);
  };

  const confirm = async () => {
    if (!target) return;
    try {
      const c = await send.mutateAsync({
        body: { draftKey, target, subject, body, expectedRecipients: recipients },
      });
      sentRef.current = true;
      setSent(true);
      await invalidateCommunications(qc);
      qc.setQueryData(["get", "/admin/communications/{id}", { params: { path: { id: c.id } } }], c);
      navigate(`/admin/communication/${c.id}`, {
        state: {
          flash: `Communication envoyée à ${c.recipientsCount} destinataire${c.recipientsCount > 1 ? "s" : ""}.`,
        },
      });
    } catch (error) {
      const next = changedAudience(error);
      if (next) {
        qc.setQueryData(audienceQuery(target).queryKey, next);
        setChanged({ before: recipients, after: next.recipientsCount });
        setConfirming(next.recipientsCount > 0);
        if (next.recipientsCount === 0) {
          setErrors({ target: "Aucun destinataire ne correspond désormais à cette cible." });
        }
        return;
      }
      if (!isProblem(error)) throw error;
      applyErrors(error.errors ?? [], error.detail);
    }
  };

  if (seasons.isPending) return null;
  if (!season) {
    return (
      <Page title="Nouvelle communication">
        <Alert kind="info" className="max-w-3xl">
          Aucune saison n'est ouverte ou close : les communications s'adressent aux adhérents d'une
          saison.
        </Alert>
      </Page>
    );
  }

  const feeOptions = (fees.data?.items ?? []).map((f) => ({ value: String(f.id), label: f.label }));
  const plural = recipients > 1 ? "s" : "";

  return (
    <Page
      title="Nouvelle communication"
      intro="Écrivez aux adhérents d'une saison. Chaque destinataire reçoit son propre e-mail."
    >
      <section className="max-w-3xl rounded-lg border border-line bg-surface p-5">
        {errors.root ? (
          <Alert kind="danger" role="alert" className="mb-4">
            {errors.root}
          </Alert>
        ) : null}
        <form
          noValidate
          onSubmit={(e) => {
            e.preventDefault();
            askConfirm();
          }}
          className="flex flex-col gap-8"
        >
          <section aria-labelledby="destinataires" className="flex flex-col gap-5">
            <h2 id="destinataires" className="font-display text-lg font-bold">
              Destinataires
            </h2>
            <Select
              label="Saison"
              value={String(season.id)}
              items={eligible.map((s) => ({
                value: String(s.id),
                label: s.isCurrent ? `${s.label} (saison courante)` : s.label,
              }))}
              onValueChange={(v) => {
                setSeasonId(Number(v));
                setFeeIds([]);
              }}
              className="max-w-xs"
            />
            <RadioGroup
              legend="Cible"
              value={preset}
              onValueChange={(v) => {
                if (v === "custom" && preset !== "custom") setCustom(situations);
                setPreset(v);
              }}
              error={errors.target}
              options={[
                ...presetOptions.map((p) => ({ value: p.value, label: p.label })),
                {
                  value: "custom" as const,
                  label: "Personnalisée",
                  description: "Situations de paiement et tarifs au choix.",
                },
              ]}
            />
            {preset === "custom" ? (
              <div className="flex flex-col gap-5 rounded-lg border border-line bg-canvas p-4">
                <CheckboxGroup
                  legend="Situation de paiement"
                  value={custom}
                  onValueChange={setCustom}
                  options={situationOptions}
                  error={errors.situations}
                />
                {feeOptions.length > 0 ? (
                  <CheckboxGroup
                    legend="Tarifs"
                    hint="Aucun tarif coché : tous les tarifs de la saison."
                    value={feeIds}
                    onValueChange={setFeeIds}
                    options={feeOptions}
                  />
                ) : null}
              </div>
            ) : null}
            <AudienceSummary audience={current} loading={audience.isFetching} />
            {audience.isError ? (
              <Alert kind="danger">
                Impossible de calculer les destinataires. Rechargez la page.
              </Alert>
            ) : null}
          </section>

          <section aria-labelledby="message" className="flex flex-col gap-5">
            <h2 id="message" className="font-display text-lg font-bold">
              Message
            </h2>
            <Field
              label="Objet"
              required
              maxLength={SUBJECT_MAX}
              autoComplete="off"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              hint={`${subject.length} / ${SUBJECT_MAX} caractères`}
              error={errors.subject}
            />
            <TextAreaField
              label="Message"
              required
              rows={10}
              maxLength={BODY_MAX}
              value={body}
              onChange={(e) => setBody(e.target.value)}
              hint={`Texte simple. Les liens commençant par https:// deviennent cliquables. ${body.length} / ${BODY_MAX} caractères`}
              error={errors.body}
            />
          </section>

          <section aria-labelledby="pieces" className="flex flex-col gap-3">
            <h2 id="pieces" className="sr-only">
              Pièces jointes
            </h2>
            <AttachmentsField draftKey={draftKey} onChange={onAttachments} />
            {errors.attachments ? (
              <Alert kind="danger" role="alert">
                {errors.attachments}
              </Alert>
            ) : null}
          </section>

          <div className="flex flex-col-reverse gap-2 sm:flex-row">
            <Link to="/admin/communication" className={buttonClasses({ variant: "secondary" })}>
              Annuler
            </Link>
            <Button variant="secondary" onClick={() => void openPreview()}>
              <Icon icon={Eye} className="h-4 w-4" />
              Aperçu
            </Button>
            <Button type="submit" aria-disabled={uploading || send.isPending}>
              <Icon icon={Send} className="h-4 w-4" />
              Envoyer
            </Button>
          </div>
          {uploading ? (
            <p className="text-sm text-muted">
              L'envoi sera possible une fois les fichiers déposés.
            </p>
          ) : null}
        </form>
      </section>

      <MessagePreview
        open={previewOpen}
        onOpenChange={setPreviewOpen}
        preview={preview}
        attachments={attachments}
      />

      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        tone="primary"
        title="Envoyer cette communication ?"
        confirmLabel={`Envoyer à ${recipients} destinataire${plural}`}
        confirmDisabled={uploading || send.isPending}
        onConfirm={() => void confirm()}
        content={
          <div className="flex flex-col gap-3">
            {changed ? (
              <Alert kind="warning" role="alert">
                Le nombre de destinataires a changé : {changed.after} au lieu de {changed.before}.
              </Alert>
            ) : null}
            <dl className="grid gap-1 text-sm">
              <dt className="font-bold">Objet</dt>
              <dd className="[overflow-wrap:anywhere]">{subject}</dd>
              <dt className="mt-1 font-bold">Cible</dt>
              <dd>{current?.targetLabel}</dd>
              <dt className="mt-1 font-bold">Destinataires</dt>
              <dd>
                {recipients} destinataire{plural}
              </dd>
              {attachments.length > 0 ? (
                <>
                  <dt className="mt-1 font-bold">Pièces jointes</dt>
                  <dd>
                    <ul>
                      {attachments.map((a) => (
                        <li key={a.id} className="[overflow-wrap:anywhere]">
                          {a.name} ({formatBytes(a.sizeBytes)})
                        </li>
                      ))}
                    </ul>
                  </dd>
                </>
              ) : null}
            </dl>
          </div>
        }
      >
        Chaque destinataire recevra un e-mail individuel. Une communication envoyée ne peut plus
        être modifiée.
      </ConfirmDialog>

      <ConfirmDialog
        open={blocker.state === "blocked"}
        onOpenChange={(o) => {
          if (!o && blocker.state === "blocked") blocker.reset();
        }}
        title="Abandonner cette communication ?"
        confirmLabel="Abandonner"
        onConfirm={() => blocker.proceed?.()}
      >
        Le message saisi sera perdu. Les fichiers déjà déposés seront supprimés automatiquement.
      </ConfirmDialog>
    </Page>
  );
}
