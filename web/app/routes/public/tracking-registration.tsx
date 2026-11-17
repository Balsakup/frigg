import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router";
import { isProblem } from "~/api/problem";
import {
  invalidateTracking,
  type TrackedRegistration,
  useAnnounceTrackedRegistration,
  useCancelTrackedRegistration,
  useConfirmTrackedRegistrationCard,
  useStartTrackedRegistrationCard,
  useTrackedRegistration,
} from "~/api/tracking";
import { formatEventRange } from "~/events/date";
import { RegistrationBadges } from "~/events/RegistrationBadges";
import { formatAmount, formatDate, formatDay, paymentMethodLabel } from "~/lib/format";
import { receivedSentence } from "~/payments/AmountDetails";
import { CardReturn, PaymentSection } from "~/payments/PaymentPanel";
import { LinkExpired } from "~/public/LinkExpired";
import { useTrackingAccess } from "~/public/useTrackingAccess";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import type { Route } from "./+types/tracking-registration";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Mon inscription · Frigg" }];
}

const linkClass = "font-bold text-brand-ink underline underline-offset-2";

// « de Lucas », « d'Alix » (élision devant une voyelle ou un h muet probable).
const of = (name: string) => (/^[aeiouyhàâéèêëîïôöûüœ]/i.test(name) ? `d'${name}` : `de ${name}`);

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

// Page d'une inscription suivie par lien personnel (016, US3 ;
// contracts/ui-routes.md › /suivi/inscriptions/:id) : paiement et annulation
// selon 012 FR-013, avec les composants de l'espace membre. Aucun champ n'est
// modifiable.
export default function TrackingRegistrationPage({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const access = useTrackingAccess();
  const query = useTrackedRegistration(id ?? 0, id !== null && access.kind === "ready");

  if (access.kind === "gone" || query.expired) return <LinkExpired />;
  if (query.data) return <RegistrationView reg={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Inscription introuvable</h1>
        <p className="mt-4">Cette inscription n'existe pas ou n'est plus liée à votre adresse.</p>
        <p className="mt-5">
          <Link to="/suivi" className={linkClass}>
            Revenir à mon suivi
          </Link>
        </p>
      </>
    );
  }
  if (access.kind === "error" || query.isError) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Mon inscription</h1>
        <Alert kind="danger" role="alert" className="mt-4">
          {access.kind === "error"
            ? access.message
            : "Impossible de charger cette inscription. Rechargez la page."}
        </Alert>
      </>
    );
  }
  return (
    <>
      <h1 className="font-display text-2xl font-bold">Mon inscription</h1>
      <p role="status" className="mt-4">
        Chargement…
      </p>
    </>
  );
}

function RegistrationView({ reg: r }: { reg: TrackedRegistration }) {
  const qc = useQueryClient();
  const cancel = useCancelTrackedRegistration();
  const announce = useAnnounceTrackedRegistration();
  const start = useStartTrackedRegistrationCard();
  const confirm = useConfirmTrackedRegistrationCard();
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const path = { params: { path: { id: r.id } } };
  const refresh = () => invalidateTracking(qc);
  const p = r.payment;

  const onCancel = async () => {
    setError("");
    setStatus("");
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
    <>
      <p className="mb-4">
        <Link to="/suivi" className={linkClass}>
          Mon suivi
        </Link>
      </p>
      <h1 className="font-display text-2xl font-bold [overflow-wrap:anywhere]">
        {r.event.title} — {r.member.firstName} {r.member.lastName}
      </h1>
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mt-4">
          {status}
        </Alert>
      ) : null}
      {error ? (
        <Alert kind="danger" role="alert" className="mt-4">
          {error}
        </Alert>
      ) : null}
      <div className="mt-6 flex flex-col gap-6">
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
            <Info label="Événement">{r.event.title}</Info>
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
                title={`Annuler l'inscription ${of(r.member.firstName)} ?`}
                confirmLabel="Annuler mon inscription"
                onConfirm={onCancel}
                trigger={<Button variant="danger-secondary">Annuler mon inscription</Button>}
              >
                <p>
                  La place sera libérée. Vous pourrez vous réinscrire depuis le lien de l'événement
                  tant que les inscriptions sont ouvertes.
                </p>
              </ConfirmDialog>
            </div>
          ) : r.state === "confirmed" &&
            (p?.status === "paid" || r.cardPayment?.status === "pending") ? (
            <Alert kind="info" className="mt-4">
              Pour annuler cette inscription, contactez l'association.
            </Alert>
          ) : null}
          <p className="mt-4 text-sm text-muted">
            Pour corriger vos informations, contactez l'association.
          </p>
        </section>
      </div>
    </>
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
