import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router";
import { isProblem } from "~/api/problem";
import {
  invalidateTracking,
  type TrackedMembership,
  useAnnounceTrackedMembership,
  useCancelTrackedMembership,
  useConfirmTrackedMembershipCard,
  useStartTrackedMembershipCard,
  useTrackedMembership,
} from "~/api/tracking";
import {
  formatAmount,
  formatDate,
  formatDay,
  membershipKindLabel,
  paymentMethodLabel,
} from "~/lib/format";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import { receivedSentence } from "~/payments/AmountDetails";
import { CardReturn, PaymentSection } from "~/payments/PaymentPanel";
import { LinkExpired } from "~/public/LinkExpired";
import { useTrackingAccess } from "~/public/useTrackingAccess";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import type { Route } from "./+types/tracking-membership";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Mon adhésion · Frigg" }];
}

const linkClass = "font-bold text-brand-ink underline underline-offset-2";

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

// Page d'une adhésion suivie par lien personnel (016, US2 ; contracts/ui-routes.md
// › /suivi/adhesions/:id) : annulation de la demande en attente et paiement,
// avec les composants de l'espace membre. Aucun champ n'est modifiable.
export default function TrackingMembershipPage({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const access = useTrackingAccess();
  const query = useTrackedMembership(id ?? 0, id !== null && access.kind === "ready");

  if (access.kind === "gone" || query.expired) return <LinkExpired />;
  if (query.data) return <MembershipView membership={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Adhésion introuvable</h1>
        <p className="mt-4">Cette adhésion n'existe pas ou n'est plus liée à votre adresse.</p>
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
        <h1 className="font-display text-2xl font-bold">Mon adhésion</h1>
        <Alert kind="danger" role="alert" className="mt-4">
          {access.kind === "error"
            ? access.message
            : "Impossible de charger cette adhésion. Rechargez la page."}
        </Alert>
      </>
    );
  }
  return (
    <>
      <h1 className="font-display text-2xl font-bold">Mon adhésion</h1>
      <p role="status" className="mt-4">
        Chargement…
      </p>
    </>
  );
}

function MembershipView({ membership: m }: { membership: TrackedMembership }) {
  const qc = useQueryClient();
  const cancel = useCancelTrackedMembership();
  const announce = useAnnounceTrackedMembership();
  const start = useStartTrackedMembershipCard();
  const confirm = useConfirmTrackedMembershipCard();
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const path = { params: { path: { id: m.id } } };
  const refresh = () => invalidateTracking(qc);

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
    setStatus("Demande annulée.");
  };

  const kind = m.kind ?? m.declaredKind;
  return (
    <>
      <p className="mb-4">
        <Link to="/suivi" className={linkClass}>
          Mon suivi
        </Link>
      </p>
      <h1 className="font-display text-2xl font-bold [overflow-wrap:anywhere]">
        {m.member.firstName} {m.member.lastName} — {m.season.label}
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
          successMessage={`Paiement reçu, merci. L'adhésion de ${m.member.firstName} est réglée.`}
        />
        {m.actions.pay && m.payment ? (
          <PaymentSection
            amountCents={m.amountCents}
            payment={m.payment}
            methods={m.paymentMethods ?? []}
            announce={(method) => announce.mutateAsync({ ...path, body: { method } })}
            startCard={() => start.mutateAsync(path)}
            refresh={refresh}
            subject="cette adhésion"
          />
        ) : null}
        <section
          aria-labelledby="adhesion-infos"
          className="rounded-lg border border-line bg-surface p-5"
        >
          <h2 id="adhesion-infos" className="font-display text-lg font-bold">
            Adhésion
          </h2>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <Info label="Membre" value={`${m.member.firstName} ${m.member.lastName}`} />
            <Info label="Saison" value={m.season.label} />
            <Info label="Tarif" value={m.fee.label} />
            <Info label="Type" value={kind ? membershipKindLabel(kind) : "—"} />
            <Info label="Montant" value={formatAmount(m.amountCents)} />
            <Info label="Date de la demande" value={formatDate(m.requestedAt)} />
            <div>
              <dt className="text-sm text-muted">État</dt>
              <dd className="mt-0.5">
                <MembershipBadges
                  status={m.status}
                  amountCents={m.amountCents}
                  payment={m.payment}
                />
              </dd>
            </div>
            {m.reason ? <Info label="Motif" value={m.reason} /> : null}
          </dl>
          {m.actions.cancel ? (
            <div className="mt-4">
              <ConfirmDialog
                title={`Annuler la demande d'adhésion de ${m.member.firstName} ?`}
                confirmLabel="Annuler ma demande"
                onConfirm={onCancel}
                trigger={<Button variant="danger-secondary">Annuler ma demande</Button>}
              >
                <p>Vous pourrez déposer une nouvelle demande pour cette saison.</p>
              </ConfirmDialog>
            </div>
          ) : null}
          <p className="mt-4 text-sm text-muted">
            Pour corriger vos informations, contactez l'association.
          </p>
        </section>
        <PaidStatus membership={m} />
      </div>
    </>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 whitespace-pre-line font-bold [overflow-wrap:anywhere]">{value}</dd>
    </div>
  );
}

// Paiement terminé ou sans objet.
function PaidStatus({ membership: m }: { membership: TrackedMembership }) {
  if (m.status !== "validated" || m.actions.pay) return null;
  if (!m.payment) {
    return <Alert kind="success">Adhésion gratuite : aucun paiement n'est demandé.</Alert>;
  }
  if (m.payment.status === "paid" && m.payment.paidOn && m.payment.paidMethod) {
    return (
      <Alert kind="success">
        Payée le {formatDay(m.payment.paidOn)} (
        {paymentMethodLabel(m.payment.paidMethod).toLowerCase()}).
        {receivedSentence(m.payment.paidAmountCents, m.payment.donationCents)}
        {m.payment.online?.test ? " Mode test : aucun débit réel." : ""}
      </Alert>
    );
  }
  return null;
}
