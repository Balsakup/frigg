import { Dialog } from "@base-ui/react/dialog";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { Link, useLocation } from "react-router";
import { type AdminMembership, invalidateMemberships, membershipQuery } from "~/api/memberships";
import { isProblem, type Problem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { CustomFieldInputs } from "~/customfields/CustomFieldInputs";
import { CustomFieldValues } from "~/customfields/CustomFieldValues";
import { useCustomFieldsForm } from "~/customfields/useCustomFieldsForm";
import {
  formatAmount,
  formatDate,
  formatDay,
  formatMoney,
  type MembershipKind,
  membershipKindLabel,
  paymentMethodLabel,
  previousSeasonLabel,
} from "~/lib/format";
import { FeeChoice } from "~/memberships/FeeChoice";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import {
  DialogShell,
  errorMessage,
  HelloAssoOrder,
  OnlinePaymentsPanel,
  PaymentAlertsPanel,
  RecordPaymentDialog,
} from "~/payments/AdminPayments";
import { AdjustmentNote, ReceivedAmount } from "~/payments/AmountDetails";
import { AmountToPayField } from "~/payments/AmountToPayField";
import { useTrackedAmount } from "~/payments/useTrackedAmount";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { TextAreaField } from "~/ui/Field";
import { RadioGroup } from "~/ui/RadioGroup";
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
            <Info label={m.amountFixed ? "Montant à payer" : "Montant"}>
              {formatAmount(m.amountCents)}
              {!m.amountFixed ? (
                <span className="font-normal text-muted"> (calculé avec le tarif actuel)</span>
              ) : null}
              <AdjustmentNote
                baseAmountCents={m.baseAmountCents}
                adjustmentCents={m.adjustmentCents}
                reason={m.adjustmentReason}
              />
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

        <MembershipCustomFields membership={m} onDone={done} onError={failed} />

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
                  {m.payment.paidMethod !== "card" ? (
                    <Info label="Montant reçu">
                      <ReceivedAmount
                        paidAmountCents={m.payment.paidAmountCents}
                        donationCents={m.payment.donationCents}
                      />
                    </Info>
                  ) : null}
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
  const [errors, setErrors] = useState<{
    feeId?: string;
    kind?: string;
    amount?: string;
    adjustmentReason?: string;
    root?: string;
  }>({});
  const validate = $api.useMutation("post", "/admin/memberships/{id}/validate");
  const update = $api.useMutation("put", "/admin/memberships/{id}");
  const fee = (m.seasonFees ?? []).find((f) => f.id === feeId);
  const isValidate = mode === "validate";
  // Montant à payer (014) : en correction, une adhésion ajustée garde son
  // montant saisi.
  const initialAmount = isValidate
    ? null
    : { amountCents: m.amountCents, adjustmentReason: m.adjustmentReason };
  const theoretical = fee && kind ? fee.amounts[kind] : undefined;
  const amount = useTrackedAmount(theoretical, initialAmount);
  // Les champs personnalisés font partie de la validation (011, US3).
  const fields = isValidate ? (m.customFields ?? []) : [];
  const custom = useCustomFieldsForm(fields);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fee || !kind) {
      setErrors({
        feeId: fee ? undefined : "Choisissez un tarif.",
        kind: kind ? undefined : "Choisissez le type d'adhésion.",
      });
      return;
    }
    const args = {
      params: { path: { id: m.id } },
      body: { feeId: fee.id, kind, ...amount.payload() },
    };
    custom.clearErrors();
    try {
      if (isValidate)
        await validate.mutateAsync({
          ...args,
          body: { ...args.body, customFields: custom.inputs() },
        });
      else await update.mutateAsync(args);
    } catch (err) {
      const p = err as Problem;
      if (isProblem(p) && p.status === 422) {
        const placed = custom.applyProblem(p);
        const at = (loc: string) => p.errors?.find((x) => x.location === loc)?.message;
        const amountErrors = {
          amount: at("body.amount"),
          adjustmentReason: at("body.adjustmentReason"),
        };
        const own = at("body.feeId") ?? amountErrors.amount ?? amountErrors.adjustmentReason;
        setErrors({
          feeId: at("body.feeId"),
          ...amountErrors,
          root: placed || own ? undefined : p.detail,
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
          amount.restart(initialAmount);
          setErrors({});
          custom.reset(fields);
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
        <AmountToPayField
          amount={amount}
          theoreticalCents={theoretical}
          errors={{ amount: errors.amount, adjustmentReason: errors.adjustmentReason }}
        />
        <CustomFieldInputs fields={fields} control={custom.control} errors={custom.errors} />
        {errors.root ? <Alert kind="danger">{errors.root}</Alert> : null}
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

// Constatation du paiement (US4 §9) : dialogue partagé (payments/).
function PaymentDialog({ membership: m, onDone, onError }: ActionProps) {
  const record = $api.useMutation("post", "/admin/memberships/{id}/payment");
  return (
    <RecordPaymentDialog
      amountCents={m.amountCents}
      reference={m.payment?.reference}
      announcedMethod={m.payment?.announcedMethod}
      record={(body) => record.mutateAsync({ params: { path: { id: m.id } }, body })}
      onDone={onDone}
      onError={onError}
    />
  );
}

// Signalements de paiement à traiter (008, US3).
function PaymentAlerts({ membership: m, onDone, onError }: ActionProps) {
  const resolve = $api.useMutation("post", "/admin/payment-alerts/{id}/resolve");
  return (
    <PaymentAlertsPanel
      alerts={m.alerts ?? []}
      resolve={(id) => resolve.mutateAsync({ params: { path: { id } } })}
      onDone={onDone}
      onError={onError}
    />
  );
}

// Tentatives de paiement par carte et vérification auprès de HelloAsso (008, US3).
function OnlinePayments({ membership: m, onDone, onError }: ActionProps) {
  const verify = $api.useMutation("post", "/admin/memberships/{id}/card-payment/verify");
  return (
    <OnlinePaymentsPanel
      ops={m.onlinePayments ?? []}
      pending={m.cardPaymentPending}
      canVerify={m.actions.verifyCardPayment}
      verify={() => verify.mutateAsync({ params: { path: { id: m.id } } })}
      onDone={onDone}
      onError={onError}
    />
  );
}

// Champs personnalisés d'adhésion (011, US3) : modifiables tant que
// l'adhésion est en attente ou validée, payée ou non ; en lecture sinon.
function MembershipCustomFields({ membership: m, onDone, onError }: ActionProps) {
  const fields = m.customFields ?? [];
  const writable = fields.some((f) => f.editable);
  const custom = useCustomFieldsForm(fields);
  const save = $api.useMutation("put", "/admin/memberships/{id}/fields");
  if (fields.length === 0) return null;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    custom.clearErrors();
    try {
      const saved = await save.mutateAsync({
        params: { path: { id: m.id } },
        body: { customFields: custom.inputs() },
      });
      custom.reset(saved.customFields ?? []);
    } catch (err) {
      if (isProblem(err) && err.status === 422 && custom.applyProblem(err)) return;
      await onError(err);
      return;
    }
    await onDone("Informations complémentaires enregistrées.");
  };

  return (
    <section
      aria-labelledby="informations-complementaires"
      className="rounded-lg border border-line bg-surface p-5"
    >
      <h2 id="informations-complementaires" className="font-display text-lg font-bold">
        Informations complémentaires
      </h2>
      {writable ? (
        <form onSubmit={submit} noValidate className="mt-3 flex flex-col gap-5">
          <CustomFieldInputs
            fields={fields}
            control={custom.control}
            errors={custom.errors}
            legend="Champs de l'adhésion"
          />
          <div>
            <Button type="submit" variant="secondary" aria-disabled={save.isPending}>
              Enregistrer les informations
            </Button>
          </div>
        </form>
      ) : (
        <div className="mt-3">
          <CustomFieldValues fields={fields} />
        </div>
      )}
    </section>
  );
}
