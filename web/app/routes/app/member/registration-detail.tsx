import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useLocation } from "react-router";
import { invalidateEvents } from "~/api/events";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { type MyRegistration, myRegistrationQuery } from "~/api/registrations";
import { formatEventRange } from "~/events/date";
import { RegistrationBadges } from "~/events/RegistrationBadges";
import { formatAmount, formatDate, formatDay, paymentMethodLabel } from "~/lib/format";
import { receivedSentence } from "~/payments/AmountDetails";
import { CardReturn, PaymentSection } from "~/payments/PaymentPanel";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import NotFound from "../not-found";
import type { Route } from "./+types/registration-detail";

// « de Lucas », « d'Alix » (élision devant une voyelle ou un h muet probable).
const of = (name: string) => (/^[aeiouyhàâéèêëîïôöûüœ]/i.test(name) ? `d'${name}` : `de ${name}`);

const title = (r: MyRegistration) =>
  `${r.event.title} — ${r.member.firstName} ${r.member.lastName}`;

export const handle = {
  crumb: (data: unknown) => {
    const r = data as MyRegistration | null;
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
  return queryClient.ensureQueryData(myRegistrationQuery(id)).catch(() => null);
}

// Une inscription : récapitulatif, paiement et annulation (US3, US4).
export default function MyRegistrationDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...myRegistrationQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <RegistrationView reg={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger cette inscription. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function RegistrationView({ reg: r }: { reg: MyRegistration }) {
  const qc = useQueryClient();
  const location = useLocation();
  const [status, setStatus] = useState(
    () => (location.state as { flash?: string } | null)?.flash ?? "",
  );
  const [error, setError] = useState("");
  const cancel = $api.useMutation("post", "/me/registrations/{id}/cancel");
  const announce = $api.useMutation("put", "/me/registrations/{id}/payment");
  const start = $api.useMutation("post", "/me/registrations/{id}/card-payment");
  const confirm = $api.useMutation("post", "/me/registrations/{id}/card-payment/confirm");
  const path = { params: { path: { id: r.id } } };
  const refresh = () => invalidateEvents(qc);
  const p = r.payment;

  const onCancel = async () => {
    setError("");
    try {
      await cancel.mutateAsync(path);
    } catch (e) {
      setError(isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.");
      await refresh();
      return;
    }
    await refresh();
    setStatus("Inscription annulée : la place est libérée.");
  };

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
        <CardReturn
          confirm={(attempt, outcome) =>
            confirm.mutateAsync({ ...path, body: { onlinePaymentId: attempt, outcome } })
          }
          refresh={refresh}
          successMessage={`Paiement reçu, merci. L'inscription ${of(r.member.firstName)} est réglée.`}
        />
        {r.event.phase === "cancelled" ? (
          <Alert kind="danger" title="Événement annulé">
            Cet événement n'aura pas lieu.
          </Alert>
        ) : null}
        {r.actions.pay && p ? (
          <PaymentSection
            amountCents={r.amountCents}
            payment={p}
            methods={r.paymentMethods ?? []}
            announce={(method) => announce.mutateAsync({ ...path, body: { method } })}
            startCard={() => start.mutateAsync(path)}
            refresh={refresh}
            subject="cette inscription"
          />
        ) : null}
        <section
          aria-labelledby="inscription-infos"
          className="rounded-lg border border-line bg-surface p-5"
        >
          <h2 id="inscription-infos" className="font-display text-lg font-bold">
            Inscription
          </h2>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <Info label="Événement">
              <Link
                to={`/evenements/${r.event.id}`}
                className="text-brand-ink underline underline-offset-2"
              >
                {r.event.title}
              </Link>
            </Info>
            <Info label="Date">{formatEventRange(r.event.startsAt, r.event.endsAt)}</Info>
            <Info label="Lieu">{r.event.location}</Info>
            <Info label="Membre">
              {r.member.firstName} {r.member.lastName}
            </Info>
            <Info label="Inscrit le">{formatDate(r.registeredAt)}</Info>
            <Info label="Montant">{formatAmount(r.amountCents)}</Info>
            {p ? <Info label="Référence">{p.reference}</Info> : null}
            <div>
              <dt className="text-sm text-muted">État</dt>
              <dd className="mt-0.5">
                <RegistrationBadges state={r.state} payment={p} />
              </dd>
            </div>
          </dl>
          {p?.status === "paid" && p.paidOn && p.paidMethod ? (
            <Alert kind="success" className="mt-4">
              Payée le {formatDay(p.paidOn)} ({paymentMethodLabel(p.paidMethod).toLowerCase()}).
              {receivedSentence(p.paidAmountCents, p.donationCents)}
              {p.online?.test ? " Mode test : aucun débit réel." : ""}
            </Alert>
          ) : null}
          {r.actions.cancel ? (
            <div className="mt-4">
              <ConfirmDialog
                title={`Annuler l'inscription de ${r.member.firstName} ?`}
                confirmLabel="Annuler l'inscription"
                onConfirm={onCancel}
                trigger={<Button variant="danger-secondary">Annuler mon inscription</Button>}
              >
                <p>
                  La place sera libérée. Vous pourrez vous réinscrire tant que les inscriptions sont
                  ouvertes.
                </p>
              </ConfirmDialog>
            </div>
          ) : r.state === "confirmed" &&
            (p?.status === "paid" || r.cardPayment?.status === "pending") ? (
            <Alert kind="info" className="mt-4">
              Pour annuler cette inscription, contactez l'association.
            </Alert>
          ) : null}
        </section>
      </div>
    </Page>
  );
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 font-bold [overflow-wrap:anywhere]">{children}</dd>
    </div>
  );
}
