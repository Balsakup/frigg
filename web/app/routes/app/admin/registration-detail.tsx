import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router";
import { invalidateEvents } from "~/api/events";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { type AdminRegistration, registrationQuery } from "~/api/registrations";
import { formatEventRange } from "~/events/date";
import { RegistrationBadges } from "~/events/RegistrationBadges";
import { formatAmount, formatDate, formatDay, paymentMethodLabel } from "~/lib/format";
import {
  errorMessage,
  OnlinePaymentsPanel,
  PaymentAlertsPanel,
  RecordPaymentDialog,
} from "~/payments/AdminPayments";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import NotFound from "../not-found";
import type { Route } from "./+types/registration-detail";

const title = (r: AdminRegistration) =>
  `${r.member.firstName} ${r.member.lastName} — ${r.event.title}`;

export const handle = {
  crumb: (data: unknown) => {
    const r = data as AdminRegistration | null;
    return r ? title(r) : "Inscription introuvable";
  },
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(registrationQuery(id)).catch(() => null);
}

// Une inscription : paiement, tentatives par carte, signalements (US5).
export default function AdminRegistrationDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...registrationQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <RegistrationView reg={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert">
        Impossible de charger cette inscription. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 font-bold [overflow-wrap:anywhere]">{children}</dd>
    </div>
  );
}

function RegistrationView({ reg: r }: { reg: AdminRegistration }) {
  const qc = useQueryClient();
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const record = $api.useMutation("post", "/admin/registrations/{id}/payment");
  const cancelPayment = $api.useMutation("delete", "/admin/registrations/{id}/payment");
  const remove = $api.useMutation("post", "/admin/registrations/{id}/cancel");
  const verify = $api.useMutation("post", "/admin/registrations/{id}/card-payment/verify");
  const resolve = $api.useMutation("post", "/admin/registration-alerts/{id}/resolve");
  const path = { params: { path: { id: r.id } } };

  const done = async (message: string) => {
    setError("");
    setStatus(message);
    await invalidateEvents(qc);
  };
  const failed = async (e: unknown) => {
    setStatus("");
    setError(errorMessage(e));
    await invalidateEvents(qc);
  };
  const run = async (fn: () => Promise<unknown>, message: string) => {
    try {
      await fn();
    } catch (e) {
      await failed(e);
      return;
    }
    await done(message);
  };

  const p = r.payment;
  return (
    <Page title={title(r)}>
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}
      {error ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {error}
        </Alert>
      ) : null}
      <div className="flex max-w-3xl flex-col gap-6">
        <PaymentAlertsPanel
          alerts={r.alerts ?? []}
          resolve={(id) => resolve.mutateAsync({ params: { path: { id } } })}
          onDone={done}
          onError={failed}
        />
        <section
          aria-labelledby="inscription-infos"
          className="rounded-lg border border-line bg-surface p-5"
        >
          <h2 id="inscription-infos" className="font-display text-lg font-bold">
            Inscription
          </h2>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <Info label="Membre">
              {r.member.firstName} {r.member.lastName}
              {r.state === "confirmed" && !r.eligible ? (
                <span className="ml-2">
                  <Badge tone="warning">Non éligible</Badge>
                </span>
              ) : null}
            </Info>
            <Info label="Événement">
              <Link
                to={`/admin/evenements/${r.event.id}`}
                className="text-brand-ink underline underline-offset-2"
              >
                {r.event.title}
              </Link>
              <span className="block text-sm font-normal text-muted">
                {formatEventRange(r.event.startsAt, r.event.startsAt)}
              </span>
            </Info>
            <Info label="Adresse de contact">{r.contactEmail ?? "Non joignable par e-mail"}</Info>
            <Info label="Adhérent de la saison">{r.isSeasonMember ? "Oui" : "Non"}</Info>
            <Info label="Inscrit le">
              {formatDate(r.registeredAt)}
              <span className="block text-sm font-normal text-muted">
                par {r.registeredBy?.email ?? "—"}
              </span>
            </Info>
            <Info label="Montant">{formatAmount(r.amountCents)}</Info>
            <div>
              <dt className="text-sm text-muted">État</dt>
              <dd className="mt-0.5">
                <RegistrationBadges state={r.state} payment={p} />
              </dd>
            </div>
            {p ? <Info label="Référence">{p.reference}</Info> : null}
            {p?.status === "paid" && p.paidMethod && p.paidOn ? (
              <Info label="Paiement">
                {paymentMethodLabel(p.paidMethod)}, le {formatDay(p.paidOn)}
                {r.paymentNote ? (
                  <span className="block text-sm font-normal text-muted">{r.paymentNote}</span>
                ) : null}
              </Info>
            ) : null}
            {r.cancelledAt ? (
              <Info label="Annulée le">
                {formatDate(r.cancelledAt)}
                <span className="block text-sm font-normal text-muted">
                  par {r.cancelledBy?.email ?? "le compte ou HelloAsso"}
                </span>
              </Info>
            ) : null}
          </dl>
          <div className="mt-5 flex flex-wrap gap-2">
            {r.actions.recordPayment ? (
              <RecordPaymentDialog
                amountCents={r.amountCents}
                reference={p?.reference}
                announcedMethod={p?.announcedMethod}
                record={(body) => record.mutateAsync({ ...path, body })}
                onDone={done}
                onError={failed}
              />
            ) : null}
            {r.actions.cancelPayment ? (
              <ConfirmDialog
                title="Annuler la constatation du paiement ?"
                confirmLabel="Annuler la constatation"
                onConfirm={() =>
                  run(() => cancelPayment.mutateAsync(path), "Constatation annulée.")
                }
                trigger={<Button variant="secondary">Annuler la constatation</Button>}
              >
                <p>L'inscription reviendra à l'état « À payer » ou « Paiement annoncé ».</p>
              </ConfirmDialog>
            ) : null}
            {r.actions.remove ? (
              <ConfirmDialog
                title={`Retirer l'inscription de ${r.member.firstName} ${r.member.lastName} ?`}
                confirmLabel="Retirer l'inscription"
                onConfirm={() => run(() => remove.mutateAsync(path), "Inscription retirée.")}
                trigger={<Button variant="danger-secondary">Retirer l'inscription</Button>}
              >
                <p>
                  La place sera libérée et la personne prévenue par e-mail si elle est joignable.
                </p>
                {p?.status === "paid" || r.cardPaymentPending ? (
                  <p className="mt-2 font-bold">
                    Le remboursement se fait hors de Frigg, ou dans l'espace HelloAsso pour un
                    paiement par carte.
                  </p>
                ) : null}
              </ConfirmDialog>
            ) : null}
          </div>
        </section>
        <OnlinePaymentsPanel
          ops={r.onlinePayments ?? []}
          pending={r.cardPaymentPending}
          canVerify={r.actions.verifyCardPayment}
          verify={() => verify.mutateAsync(path)}
          onDone={done}
          onError={failed}
        />
      </div>
    </Page>
  );
}
