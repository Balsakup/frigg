import { Dialog } from "@base-ui/react/dialog";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink, RefreshCw } from "lucide-react";
import { type ReactElement, useState } from "react";
import { Link, useLocation } from "react-router";
import { type AdminMembership, invalidateMemberships, membershipQuery } from "~/api/memberships";
import { isProblem, type Problem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import {
  formatAmount,
  formatDate,
  formatDay,
  formatMoney,
  type ManualPaymentMethod,
  type MembershipKind,
  membershipKindLabel,
  paymentMethodLabel,
  previousSeasonLabel,
} from "~/lib/format";
import { AmountSummary, FeeChoice } from "~/memberships/FeeChoice";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Field, TextAreaField } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { RadioGroup } from "~/ui/RadioGroup";
import { Table } from "~/ui/Table";
import NotFound from "../not-found";
import type { Route } from "./+types/membership-detail";

const title = (m: AdminMembership) =>
  `${m.member.firstName} ${m.member.lastName} — ${m.season.label}`;

export const handle = {
  crumb: (data: unknown) => {
    const m = data as AdminMembership | null;
    return m ? title(m) : "Adhésion introuvable";
  },
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(membershipQuery(id)).catch(() => null);
}

const kindOptions: { value: MembershipKind; label: string }[] = [
  { value: "first", label: "Première adhésion" },
  { value: "renewal", label: "Renouvellement" },
];

const methodOptions: { value: ManualPaymentMethod; label: string }[] = [
  { value: "cash", label: "Espèces" },
  { value: "cheque", label: "Chèque" },
  { value: "transfer", label: "Virement" },
];

function errorMessage(e: unknown) {
  if (!isProblem(e)) return "Une erreur inattendue s'est produite.";
  return (
    e.errors?.map((x) => x.message).join(" ") || e.detail || "Une erreur inattendue s'est produite."
  );
}

function todayISO() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

// Détail d'une adhésion et décisions de l'administration (006, US2, US4, US7).
export default function AdminMembershipDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...membershipQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <MembershipView membership={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert">
        Impossible de charger cette adhésion. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 whitespace-pre-line font-bold [overflow-wrap:anywhere]">{children}</dd>
    </div>
  );
}

function MembershipView({ membership: m }: { membership: AdminMembership }) {
  const qc = useQueryClient();
  const location = useLocation();
  const [status, setStatus] = useState(
    () => (location.state as { flash?: string } | null)?.flash ?? "",
  );
  const [error, setError] = useState("");
  const path = { params: { path: { id: m.id } } };
  const cancelPayment = $api.useMutation("delete", "/admin/memberships/{id}/payment");

  const done = async (message: string) => {
    setError("");
    setStatus(message);
    await invalidateMemberships(qc);
  };
  const failed = async (e: unknown) => {
    setStatus("");
    setError(errorMessage(e));
    await invalidateMemberships(qc);
  };

  const by = (a?: { email: string } | null) => (a ? ` par ${a.email}` : "");
  const a = m.actions;

  return (
    <Page title={title(m)}>
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6">
          {status}
        </Alert>
      ) : null}
      {error ? (
        <Alert kind="danger" role="alert" className="mb-6">
          {error}
        </Alert>
      ) : null}

      <div className="flex flex-col gap-6">
        <section
          aria-labelledby="adhesion"
          className="rounded-lg border border-line bg-surface p-5"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 id="adhesion" className="font-display text-lg font-bold">
              Adhésion
            </h2>
            <MembershipBadges status={m.status} amountCents={m.amountCents} payment={m.payment} />
          </div>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <Info label="Membre">
              <Link
                to={`/admin/membres/${m.member.id}`}
                className="text-brand-ink underline underline-offset-2"
              >
                {m.member.firstName} {m.member.lastName}
              </Link>{" "}
              <span className="font-normal text-muted">({m.member.age} ans)</span>
            </Info>
            <Info label="Compte de rattachement">
              {m.account ? (
                <span className="inline-flex flex-wrap items-center gap-2">
                  <Link
                    to={`/admin/comptes/${m.account.id}`}
                    className="text-brand-ink underline underline-offset-2"
                  >
                    {m.account.email}
                  </Link>
                  {m.account.disabled ? <Badge tone="muted">désactivé</Badge> : null}
                </span>
              ) : (
                <span className="font-normal text-muted">Sans compte</span>
              )}
            </Info>
            <Info label="Saison">{m.season.label}</Info>
            <Info label="Tarif">{m.fee.label}</Info>
            <Info label="Type déclaré">
              {m.declaredKind ? membershipKindLabel(m.declaredKind) : "—"}
            </Info>
            <Info label="Type retenu">{m.kind ? membershipKindLabel(m.kind) : "—"}</Info>
            <Info label="Montant">
              {formatAmount(m.amountCents)}
              {!m.amountFixed ? (
                <span className="font-normal text-muted"> (calculé avec le tarif actuel)</span>
              ) : null}
            </Info>
            <Info label={m.createdBy ? "Créée" : "Demandée"}>
              <span className="font-normal">
                le {formatDate(m.requestedAt)}
                {by(m.createdBy ?? m.requestedBy)}
              </span>
            </Info>
            {m.decidedAt && !m.createdBy ? (
              <Info label="Décision">
                <span className="font-normal">
                  le {formatDate(m.decidedAt)}
                  {by(m.decidedBy)}
                </span>
              </Info>
            ) : null}
            {m.cancelledAt ? (
              <Info label="Annulée">
                <span className="font-normal">
                  le {formatDate(m.cancelledAt)}
                  {by(m.cancelledBy)}
                </span>
              </Info>
            ) : null}
            {m.reason ? <Info label="Motif">{m.reason}</Info> : null}
            <Info label="Adhérent la saison précédente dans Frigg">
              {previousSeasonLabel(m.previousSeason)}
              {m.previousSeason === "unknown" ? (
                <span className="block text-sm font-normal text-muted">
                  La saison précédente n'est pas gérée dans Frigg.
                </span>
              ) : null}
            </Info>
          </dl>
          {a.validate || a.refuse || a.update || a.cancel ? (
            <div className="mt-5 flex flex-wrap gap-2">
              {a.validate ? (
                <DecisionDialog
                  membership={m}
                  mode="validate"
                  trigger={<Button>Valider</Button>}
                  onDone={done}
                  onError={failed}
                />
              ) : null}
              {a.refuse ? (
                <ReasonDialog
                  membership={m}
                  mode="refuse"
                  trigger={<Button variant="danger-secondary">Refuser</Button>}
                  onDone={done}
                  onError={failed}
                />
              ) : null}
              {a.update ? (
                <DecisionDialog
                  membership={m}
                  mode="update"
                  trigger={<Button variant="secondary">Corriger</Button>}
                  onDone={done}
                  onError={failed}
                />
              ) : null}
              {a.cancel ? (
                <ReasonDialog
                  membership={m}
                  mode="cancel"
                  trigger={<Button variant="danger-secondary">Annuler l'adhésion</Button>}
                  onDone={done}
                  onError={failed}
                />
              ) : null}
            </div>
          ) : null}
        </section>

        {m.payment ? (
          <section
            aria-labelledby="paiement"
            className="rounded-lg border border-line bg-surface p-5"
          >
            <h2 id="paiement" className="font-display text-lg font-bold">
              Paiement
            </h2>
            <dl className="mt-3 grid gap-3 sm:grid-cols-2">
              <Info label="Référence">{m.payment.reference}</Info>
              <Info label="Moyen annoncé">
                {m.payment.announcedMethod
                  ? paymentMethodLabel(m.payment.announcedMethod)
                  : "Aucun"}
              </Info>
              {m.payment.status === "paid" && m.payment.paidMethod && m.payment.paidOn ? (
                <>
                  <Info label="Payée">
                    <span className="font-normal">
                      le {formatDay(m.payment.paidOn)} (
                      {paymentMethodLabel(m.payment.paidMethod).toLowerCase()}){by(m.paidBy)}
                    </span>
                    {m.payment.online?.test ? (
                      <span className="ml-2">
                        <Badge tone="warning">Mode test</Badge>
                      </span>
                    ) : null}
                  </Info>
                  {m.payment.paidMethod === "card" && m.payment.online ? (
                    <>
                      <Info label="Montant reçu">
                        {formatMoney(m.payment.paidAmountCents ?? 0)}
                        <span className="block text-sm font-normal text-muted">
                          hors contribution volontaire à HelloAsso, le{" "}
                          {formatDate(m.payment.online.paidAt)}
                        </span>
                      </Info>
                      <Info label="Commande HelloAsso">
                        <HelloAssoOrder
                          orderId={m.payment.online.orderId}
                          url={
                            m.onlinePayments?.find((o) => o.orderId === m.payment?.online?.orderId)
                              ?.helloassoUrl
                          }
                        />
                      </Info>
                    </>
                  ) : null}
                  {m.paymentNote ? <Info label="Commentaire">{m.paymentNote}</Info> : null}
                </>
              ) : null}
            </dl>
            <div className="mt-5 flex flex-wrap gap-2">
              {a.recordPayment ? (
                <PaymentDialog membership={m} onDone={done} onError={failed} />
              ) : null}
              {a.cancelPayment ? (
                <ConfirmDialog
                  title="Annuler la constatation du paiement ?"
                  confirmLabel="Annuler la constatation"
                  onConfirm={async () => {
                    try {
                      await cancelPayment.mutateAsync(path);
                    } catch (e) {
                      await failed(e);
                      return;
                    }
                    await done("Constatation du paiement annulée.");
                  }}
                  trigger={<Button variant="secondary">Annuler la constatation</Button>}
                >
                  <p>L'adhésion redeviendra à payer.</p>
                </ConfirmDialog>
              ) : null}
            </div>
          </section>
        ) : m.status === "validated" ? (
          <Alert kind="success">Adhésion gratuite : aucun paiement n'est demandé.</Alert>
        ) : null}

        <PaymentAlerts membership={m} onDone={done} onError={failed} />
        <OnlinePayments membership={m} onDone={done} onError={failed} />
      </div>
    </Page>
  );
}

interface DialogShellProps {
  title: string;
  trigger: ReactElement;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}

function DialogShell({ title, trigger, open, onOpenChange, children }: DialogShellProps) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Trigger render={trigger} />
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 max-h-[calc(100vh-2rem)] w-[min(34rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <Dialog.Title className="font-display text-xl font-bold">{title}</Dialog.Title>
          {children}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

interface ActionProps {
  membership: AdminMembership;
  onDone: (message: string) => Promise<void>;
  onError: (e: unknown) => Promise<void>;
}

// Validation (tarif et type retenus) ou correction d'une adhésion validée.
function DecisionDialog({
  membership: m,
  mode,
  trigger,
  onDone,
  onError,
}: ActionProps & { mode: "validate" | "update"; trigger: ReactElement }) {
  const [open, setOpen] = useState(false);
  const initialKind = (m.kind ?? m.declaredKind ?? "") as MembershipKind | "";
  const [feeId, setFeeId] = useState<number | null>(m.fee.id);
  const [kind, setKind] = useState<MembershipKind | "">(initialKind);
  const [errors, setErrors] = useState<{ feeId?: string; kind?: string; root?: string }>({});
  const validate = $api.useMutation("post", "/admin/memberships/{id}/validate");
  const update = $api.useMutation("put", "/admin/memberships/{id}");
  const fee = (m.seasonFees ?? []).find((f) => f.id === feeId);
  const isValidate = mode === "validate";

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fee || !kind) {
      setErrors({
        feeId: fee ? undefined : "Choisissez un tarif.",
        kind: kind ? undefined : "Choisissez le type d'adhésion.",
      });
      return;
    }
    const args = { params: { path: { id: m.id } }, body: { feeId: fee.id, kind } };
    try {
      if (isValidate) await validate.mutateAsync(args);
      else await update.mutateAsync(args);
    } catch (err) {
      const p = err as Problem;
      if (isProblem(p) && p.status === 422) {
        setErrors({
          feeId: p.errors?.find((x) => x.location === "body.feeId")?.message,
          root: p.detail,
        });
        return;
      }
      setOpen(false);
      await onError(err);
      return;
    }
    setOpen(false);
    await onDone(isValidate ? "Adhésion validée." : "Adhésion corrigée.");
  };

  return (
    <DialogShell
      title={isValidate ? "Valider l'adhésion" : "Corriger l'adhésion"}
      trigger={trigger}
      open={open}
      onOpenChange={(o) => {
        if (o) {
          setFeeId(m.fee.id);
          setKind(initialKind);
          setErrors({});
        }
        setOpen(o);
      }}
    >
      <form onSubmit={submit} noValidate className="mt-4 flex flex-col gap-5">
        {!isValidate && m.cardPaymentPending ? (
          <Alert kind="warning">
            Un paiement par carte est en cours. Le remboursement d'un éventuel écart se fait dans
            l'espace HelloAsso de l'association.
          </Alert>
        ) : null}
        <FeeChoice
          fees={m.seasonFees ?? []}
          value={feeId}
          onChange={setFeeId}
          error={errors.feeId}
        />
        <RadioGroup
          legend="Type d'adhésion"
          required
          hint={
            m.declaredKind
              ? `Déclaré : ${membershipKindLabel(m.declaredKind).toLowerCase()}.`
              : undefined
          }
          value={kind}
          onValueChange={setKind}
          error={errors.kind}
          options={kindOptions}
        />
        <AmountSummary fee={fee} kind={kind} />
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
          <Button type="submit" aria-disabled={validate.isPending || update.isPending}>
            {isValidate ? "Valider l'adhésion" : "Corriger l'adhésion"}
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}

// Refus d'une demande ou annulation d'une adhésion validée, avec motif.
function ReasonDialog({
  membership: m,
  mode,
  trigger,
  onDone,
  onError,
}: ActionProps & { mode: "refuse" | "cancel"; trigger: ReactElement }) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [fieldError, setFieldError] = useState("");
  const refuse = $api.useMutation("post", "/admin/memberships/{id}/refuse");
  const cancel = $api.useMutation("post", "/admin/memberships/{id}/cancel");
  const isRefuse = mode === "refuse";

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const args = { params: { path: { id: m.id } }, body: { reason } };
    try {
      if (isRefuse) await refuse.mutateAsync(args);
      else await cancel.mutateAsync(args);
    } catch (err) {
      if (isProblem(err) && err.status === 422) {
        setFieldError(err.errors?.[0]?.message ?? err.detail ?? "");
        return;
      }
      setOpen(false);
      await onError(err);
      return;
    }
    setOpen(false);
    await onDone(isRefuse ? "Demande refusée." : "Adhésion annulée.");
  };

  return (
    <DialogShell
      title={isRefuse ? "Refuser la demande" : "Annuler l'adhésion"}
      trigger={trigger}
      open={open}
      onOpenChange={(o) => {
        if (o) {
          setReason("");
          setFieldError("");
        }
        setOpen(o);
      }}
    >
      <form onSubmit={submit} noValidate className="mt-4 flex flex-col gap-5">
        <p className="text-muted">
          {isRefuse
            ? `La demande d'adhésion de ${m.member.firstName} ${m.member.lastName} sera refusée. Le motif est communiqué au compte qui l'a déposée.`
            : `L'adhésion de ${m.member.firstName} ${m.member.lastName} sera annulée. Le motif est communiqué au compte qui gère la fiche.`}
          {!isRefuse && (m.payment?.paidMethod === "card" || m.cardPaymentPending)
            ? " Le remboursement se fait dans l'espace HelloAsso de l'association."
            : !isRefuse && m.payment?.status === "paid"
              ? " Un éventuel remboursement se fait en dehors de Frigg."
              : ""}
        </p>
        <TextAreaField
          label="Motif"
          required
          maxLength={500}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          error={fieldError || undefined}
        />
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
          <Button
            type="submit"
            variant="danger"
            aria-disabled={refuse.isPending || cancel.isPending}
          >
            {isRefuse ? "Refuser la demande" : "Annuler l'adhésion"}
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}

// Constatation du paiement (US4 §9).
function PaymentDialog({ membership: m, onDone, onError }: ActionProps) {
  const [open, setOpen] = useState(false);
  const initialMethod = (m.payment?.announcedMethod ?? "") as ManualPaymentMethod | "";
  const [method, setMethod] = useState<ManualPaymentMethod | "">(initialMethod);
  const [receivedOn, setReceivedOn] = useState(todayISO());
  const [note, setNote] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const record = $api.useMutation("post", "/admin/memberships/{id}/payment");

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!method) {
      setErrors({ method: "Choisissez le moyen de paiement." });
      return;
    }
    try {
      await record.mutateAsync({
        params: { path: { id: m.id } },
        body: { method, receivedOn, ...(note.trim() ? { note } : {}) },
      });
    } catch (err) {
      if (isProblem(err) && err.status === 422) {
        const next: Record<string, string> = {};
        for (const x of err.errors ?? [])
          next[x.location?.replace("body.", "") ?? "root"] = x.message ?? "";
        setErrors(next);
        return;
      }
      setOpen(false);
      await onError(err);
      return;
    }
    setOpen(false);
    await onDone("Paiement enregistré.");
  };

  return (
    <DialogShell
      title="Marquer comme payée"
      trigger={<Button>Marquer comme payée</Button>}
      open={open}
      onOpenChange={(o) => {
        if (o) {
          setMethod(initialMethod);
          setReceivedOn(todayISO());
          setNote("");
          setErrors({});
        }
        setOpen(o);
      }}
    >
      <form onSubmit={submit} noValidate className="mt-4 flex flex-col gap-5">
        <p className="text-muted">
          {formatAmount(m.amountCents)} · référence {m.payment?.reference}
        </p>
        <RadioGroup
          legend="Moyen de paiement"
          required
          value={method}
          onValueChange={setMethod}
          error={errors.method}
          options={methodOptions}
        />
        <Field
          label="Date de réception"
          type="date"
          required
          value={receivedOn}
          onChange={(e) => setReceivedOn(e.target.value)}
          error={errors.receivedOn}
          className="max-w-56"
        />
        <TextAreaField
          label="Commentaire (facultatif)"
          maxLength={200}
          hint="Par exemple le numéro du chèque."
          value={note}
          onChange={(e) => setNote(e.target.value)}
          error={errors.note}
        />
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
          <Button type="submit" aria-disabled={record.isPending}>
            Enregistrer le paiement
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}

const onlineStatus: Record<
  string,
  { label: string; tone: "success" | "info" | "danger" | "muted" }
> = {
  pending: { label: "En cours", tone: "info" },
  succeeded: { label: "Réussi", tone: "success" },
  failed: { label: "Échoué", tone: "danger" },
  abandoned: { label: "Abandonné", tone: "muted" },
  expired: { label: "Expiré", tone: "muted" },
  refunded: { label: "Remboursé", tone: "info" },
};

const onlineErrors: Record<string, string> = {
  unavailable: "HelloAsso injoignable",
  unauthorized: "identifiants HelloAsso refusés",
  forbidden: "droits HelloAsso insuffisants",
  payment_error: "paiement refusé ou erreur chez HelloAsso",
  environment_changed: "changement d'environnement HelloAsso",
  no_checkout: "page de paiement non créée",
  redirect_host: "réponse inattendue de HelloAsso",
};

const alertLabels: Record<string, string> = {
  duplicate: "Paiement en double — remboursement à effectuer dans HelloAsso.",
  amount_mismatch: "Montant reçu différent du montant attendu.",
  refund_due: "Paiement par carte reçu — remboursement à effectuer dans HelloAsso.",
  unverified: "Paiement par carte à vérifier dans HelloAsso.",
};

function HelloAssoOrder({ orderId, url }: { orderId: number; url?: string | null }) {
  if (!url) return <>n° {orderId}</>;
  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-1 text-brand-ink underline underline-offset-2"
    >
      n° {orderId}
      <Icon icon={ExternalLink} className="h-4 w-4" />
      <span className="sr-only"> (espace HelloAsso, nouvel onglet)</span>
    </a>
  );
}

// Signalements de paiement à traiter (008, US3).
function PaymentAlerts({ membership: m, onDone, onError }: ActionProps) {
  const resolve = $api.useMutation("post", "/admin/payment-alerts/{id}/resolve");
  const open = (m.alerts ?? []).filter((a) => !a.resolvedAt);
  if (open.length === 0) return null;
  return (
    <div className="flex flex-col gap-3">
      {open.map((alert) => (
        <Alert
          key={alert.id}
          kind="warning"
          actions={
            <ConfirmDialog
              title="Marquer ce signalement comme traité ?"
              confirmLabel="Marquer comme traité"
              onConfirm={async () => {
                try {
                  await resolve.mutateAsync({ params: { path: { id: alert.id } } });
                } catch (e) {
                  await onError(e);
                  return;
                }
                await onDone("Signalement traité.");
              }}
              trigger={
                <Button variant="secondary" size="sm">
                  Marquer comme traité
                </Button>
              }
            >
              <p>{alertLabels[alert.kind]}</p>
              <p className="mt-2">Le signalement disparaîtra du tableau de bord.</p>
            </ConfirmDialog>
          }
        >
          {alertLabels[alert.kind]}{" "}
          <span className="text-muted">(signalé le {formatDate(alert.createdAt)})</span>
        </Alert>
      ))}
    </div>
  );
}

// Tentatives de paiement par carte et vérification auprès de HelloAsso (008, US3).
function OnlinePayments({ membership: m, onDone, onError }: ActionProps) {
  const verify = $api.useMutation("post", "/admin/memberships/{id}/card-payment/verify");
  const ops = m.onlinePayments ?? [];
  if (ops.length === 0) return null;
  const onVerify = async () => {
    if (verify.isPending) return;
    try {
      await verify.mutateAsync({ params: { path: { id: m.id } } });
    } catch (e) {
      await onError(e);
      return;
    }
    await onDone("Vérification auprès de HelloAsso effectuée.");
  };
  return (
    <section
      aria-labelledby="paiements-en-ligne"
      className="rounded-lg border border-line bg-surface"
    >
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 pt-5">
        <h2 id="paiements-en-ligne" className="font-display text-lg font-bold">
          Paiements en ligne
        </h2>
        {m.actions.verifyCardPayment ? (
          <Button variant="secondary" size="sm" onClick={onVerify} aria-disabled={verify.isPending}>
            <Icon icon={RefreshCw} className="h-4 w-4" />
            {verify.isPending ? "Vérification…" : "Vérifier auprès de HelloAsso"}
          </Button>
        ) : null}
      </div>
      {m.cardPaymentPending ? (
        <p className="px-5 pt-2 text-muted">Un paiement par carte est en cours.</p>
      ) : null}
      <div className="mt-3 pb-2">
        <Table
          caption="Tentatives de paiement par carte"
          columns={[
            { key: "date", label: "Date" },
            { key: "status", label: "État" },
            { key: "amount", label: "Montant" },
            { key: "order", label: "Commande HelloAsso" },
          ]}
        >
          {ops.map((o) => (
            <tr key={o.id} className="border-b border-line">
              <td className="whitespace-nowrap px-5 py-2.5">{formatDate(o.createdAt)}</td>
              <td className="px-5 py-2.5">
                <span className="inline-flex flex-wrap items-center gap-2">
                  <Badge tone={onlineStatus[o.status]?.tone ?? "muted"}>
                    {onlineStatus[o.status]?.label ?? o.status}
                  </Badge>
                  {o.environment === "sandbox" ? <Badge tone="warning">Mode test</Badge> : null}
                </span>
                {o.errorCode ? (
                  <span className="block text-sm text-muted">
                    {onlineErrors[o.errorCode] ?? o.errorCode}
                  </span>
                ) : null}
              </td>
              <td className="whitespace-nowrap px-5 py-2.5">
                {formatMoney(o.receivedAmountCents ?? o.amountCents)}
                {o.receivedAmountCents !== null &&
                o.receivedAmountCents !== undefined &&
                o.receivedAmountCents !== o.amountCents ? (
                  <span className="block text-sm text-muted">
                    attendu {formatMoney(o.amountCents)}
                  </span>
                ) : null}
              </td>
              <td className="px-5 py-2.5">
                {o.orderId ? <HelloAssoOrder orderId={o.orderId} url={o.helloassoUrl} /> : "—"}
              </td>
            </tr>
          ))}
        </Table>
      </div>
    </section>
  );
}
