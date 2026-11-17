import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router";
import { type TrackedItem, type Tracking, useCloseTracking, useTracking } from "~/api/tracking";
import { formatAmount, paymentStatusLabel } from "~/lib/format";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import { LinkExpired } from "~/public/LinkExpired";
import { useTrackingAccess } from "~/public/useTrackingAccess";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button, buttonClasses } from "~/ui/Button";
import type { Route } from "./+types/tracking";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Mon suivi · Frigg" }];
}

const linkClass = "font-bold text-brand-ink underline underline-offset-2";

// Espace de suivi par lien personnel (016, US2 ; contracts/ui-routes.md ›
// /suivi) : les adhésions et inscriptions des fiches sans compte de l'adresse
// du lien, par fiche.
export default function TrackingPage() {
  const access = useTrackingAccess();
  const tracking = useTracking(access.kind === "ready");
  const [closed, setClosed] = useState(false);

  if (closed) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Accès fermé</h1>
        <p role="status" className="mt-4">
          L'accès à votre suivi est fermé sur cet appareil.
        </p>
        <p className="mt-5">
          <Link to="/suivi/lien" className={linkClass}>
            Recevoir un nouveau lien
          </Link>
        </p>
      </>
    );
  }
  if (access.kind === "gone" || tracking.expired) return <LinkExpired />;
  if (access.kind === "error" || tracking.isError) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Mon suivi</h1>
        <Alert kind="danger" role="alert" className="mt-4">
          {access.kind === "error"
            ? access.message
            : "Impossible de charger votre suivi. Rechargez la page."}
        </Alert>
      </>
    );
  }
  if (!tracking.data) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Mon suivi</h1>
        <p role="status" className="mt-4">
          Chargement…
        </p>
      </>
    );
  }
  return <TrackingView tracking={tracking.data} onClosed={() => setClosed(true)} />;
}

function TrackingView({ tracking, onClosed }: { tracking: Tracking; onClosed: () => void }) {
  const members = tracking.members ?? [];
  const allItems = tracking.items ?? [];
  const qc = useQueryClient();
  const close = useCloseTracking();
  const [error, setError] = useState("");

  const onClose = async () => {
    setError("");
    try {
      await close.mutateAsync({});
    } catch {
      setError("Une erreur inattendue s'est produite.");
      return;
    }
    qc.removeQueries({ queryKey: ["get", "/tracking"] });
    qc.removeQueries({ queryKey: ["get", "/tracking/memberships/{id}"] });
    onClosed();
  };

  return (
    <>
      <h1 className="font-display text-2xl font-bold">Mon suivi</h1>
      <p className="mt-2 text-muted [overflow-wrap:anywhere]">
        Demandes liées à l'adresse <span className="font-bold text-ink">{tracking.email}</span>
      </p>
      {error ? (
        <Alert kind="danger" role="alert" className="mt-4">
          {error}
        </Alert>
      ) : null}
      {members.length === 0 ? (
        <p className="mt-6">Aucune demande n'est liée à cette adresse.</p>
      ) : (
        <div className="mt-6 flex flex-col gap-8">
          {members.map((m) => {
            const items = allItems.filter((it) => it.member.id === m.id);
            const headingId = `fiche-${m.id}`;
            return (
              <section key={m.id} aria-labelledby={headingId}>
                <h2 id={headingId} className="font-display text-lg font-bold">
                  {m.firstName} {m.lastName}
                </h2>
                {items.length === 0 ? (
                  <p className="mt-2 text-muted">Aucune demande en cours.</p>
                ) : (
                  <ul className="mt-3 flex flex-col gap-3">
                    {items.map((it) => (
                      <li key={`${it.kind}-${it.id}`}>
                        <ItemCard item={it} />
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            );
          })}
        </div>
      )}
      {tracking.canCreateAccount ? (
        <section
          aria-labelledby="creer-compte"
          className="mt-8 rounded-lg border border-line bg-canvas p-4"
        >
          <h2 id="creer-compte" className="font-display text-lg font-bold">
            Créer un compte
          </h2>
          <p className="mt-2">
            Avec un compte, vous retrouvez vos demandes et vos inscriptions dans votre espace
            membre, sans lien de suivi.
          </p>
          <p className="mt-3">
            <Link to="/suivi/compte" className={buttonClasses({ variant: "secondary" })}>
              Créer mon compte
            </Link>
          </p>
        </section>
      ) : null}
      <div className="mt-8 border-t border-line pt-5">
        <Button variant="secondary" onClick={onClose} aria-disabled={close.isPending}>
          Fermer l'accès sur cet appareil
        </Button>
      </div>
    </>
  );
}

// Carte d'une adhésion ou d'une inscription : libellé, état, montant et
// action « Payer » ou « Voir ».
export function ItemCard({ item }: { item: TrackedItem }) {
  const payment = item.paymentStatus ? { status: item.paymentStatus } : null;
  const href =
    item.kind === "membership" ? `/suivi/adhesions/${item.id}` : `/suivi/inscriptions/${item.id}`;
  return (
    <article className="rounded-lg border border-line bg-canvas p-4">
      <h3 className="font-bold [overflow-wrap:anywhere]">{item.label}</h3>
      <div className="mt-2">
        {item.kind === "membership" ? (
          <MembershipBadges
            status={item.status as "pending" | "validated" | "refused" | "cancelled"}
            amountCents={item.amountCents ?? 0}
            payment={payment}
          />
        ) : (
          <span className="inline-flex flex-wrap items-center gap-2">
            {item.status === "cancelled" ? (
              <Badge tone="muted">Annulée</Badge>
            ) : (
              <Badge tone="success">Confirmée</Badge>
            )}
            {payment ? (
              <Badge tone={payment.status === "paid" ? "success" : "warning"}>
                {paymentStatusLabel(payment)}
              </Badge>
            ) : null}
          </span>
        )}
      </div>
      {item.amountCents ? (
        <p className="mt-2">
          Montant : <span className="font-bold">{formatAmount(item.amountCents)}</span>
          {item.reference ? (
            <>
              {" "}
              · Référence : <span className="font-bold">{item.reference}</span>
            </>
          ) : null}
        </p>
      ) : null}
      {item.reason ? <p className="mt-2 whitespace-pre-line">Motif : {item.reason}</p> : null}
      <p className="mt-3">
        <Link
          to={href}
          className={item.actions.pay ? buttonClasses() : linkClass}
          aria-label={`${item.actions.pay ? "Payer" : "Voir"} : ${item.label}`}
        >
          {item.actions.pay ? "Payer" : "Voir"}
        </Link>
      </p>
    </article>
  );
}
