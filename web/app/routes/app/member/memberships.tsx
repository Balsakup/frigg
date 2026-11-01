import { useQuery } from "@tanstack/react-query";
import { CreditCard } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router";
import { membershipOfferQuery, myMembershipListQuery } from "~/api/memberships";
import { queryClient } from "~/api/query-client";
import { formatAmount, membershipKindLabel } from "~/lib/format";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

export async function clientLoader() {
  await Promise.all([
    queryClient.ensureQueryData(membershipOfferQuery()).catch(() => undefined),
    queryClient.ensureQueryData(myMembershipListQuery()).catch(() => undefined),
  ]);
  return null;
}

// « Adhésions » : les adhésions des membres du compte, par saison (006, US1,
// US3, US4).
export default function Memberships() {
  const location = useLocation();
  const [status] = useState(() => (location.state as { flash?: string } | null)?.flash ?? "");
  const offer = useQuery(membershipOfferQuery());
  const list = useQuery(myMembershipListQuery());
  const seasons = list.data?.seasons ?? [];
  const canRequest = offer.data?.canRequest ?? false;

  return (
    <Page
      title="Adhésions"
      intro="Les adhésions de vos membres à l'association."
      actions={
        canRequest ? (
          <Link to="/adhesions/demande" className={buttonClasses({})}>
            <Icon icon={CreditCard} className="h-4 w-4" />
            Demander une adhésion
          </Link>
        ) : null
      }
    >
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}

      {offer.data && (offer.data.members ?? []).length === 0 ? (
        <Alert kind="info" className="mb-6 max-w-3xl">
          Ajoutez d'abord un membre pour demander son adhésion.{" "}
          <Link to="/mes-membres" className="font-bold text-brand-ink underline underline-offset-2">
            Mes membres
          </Link>
        </Alert>
      ) : offer.data && (offer.data.seasons ?? []).length === 0 ? (
        <Alert kind="info" className="mb-6 max-w-3xl">
          Les adhésions ne sont pas ouvertes pour le moment.
        </Alert>
      ) : null}

      {list.isError ? (
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger vos adhésions. Rechargez la page.
        </Alert>
      ) : list.isPending ? null : seasons.length === 0 ? (
        <p className="max-w-3xl text-muted">Aucune adhésion pour le moment.</p>
      ) : (
        <div className="flex max-w-5xl flex-col gap-8">
          {seasons.map((s) => (
            <section key={s.season.id} aria-labelledby={`saison-${s.season.id}`}>
              <h2 id={`saison-${s.season.id}`} className="font-display text-xl font-bold">
                Saison {s.season.label}
              </h2>
              <ul className="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {(s.items ?? []).map((m) => (
                  <li key={m.id} className="rounded-lg border border-line bg-surface p-5">
                    <h3 className="font-display text-lg font-bold [overflow-wrap:anywhere]">
                      <Link
                        to={`/adhesions/${m.id}`}
                        className="text-brand-ink underline underline-offset-2"
                      >
                        {m.member.firstName} {m.member.lastName}
                      </Link>
                    </h3>
                    <p className="mt-1 text-muted">
                      {m.fee.label} · {membershipKindLabel(m.kind ?? m.declaredKind ?? "first")} ·{" "}
                      {formatAmount(m.amountCents)}
                    </p>
                    <p className="mt-2">
                      <MembershipBadges
                        status={m.status}
                        amountCents={m.amountCents}
                        payment={m.payment}
                      />
                    </p>
                    {m.reason ? (
                      <p className="mt-2 whitespace-pre-line text-sm">
                        <span className="font-bold">Motif : </span>
                        {m.reason}
                      </p>
                    ) : null}
                    {m.actions.pay ? (
                      <Link
                        to={`/adhesions/${m.id}`}
                        className={`${buttonClasses({ size: "sm" })} mt-3`}
                      >
                        Payer
                      </Link>
                    ) : null}
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}
    </Page>
  );
}
