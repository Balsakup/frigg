import { useQuery } from "@tanstack/react-query";
import { ClipboardCheck } from "lucide-react";
import { Link } from "react-router";
import { queryClient } from "~/api/query-client";
import { type MyRegistration, myRegistrationListQuery } from "~/api/registrations";
import { formatEventRange } from "~/events/date";
import { RegistrationBadges } from "~/events/RegistrationBadges";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { Table, TableCell, TableRow } from "~/ui/Table";

export async function clientLoader() {
  await queryClient.ensureQueryData(myRegistrationListQuery()).catch(() => undefined);
  return null;
}

// « Mes inscriptions » (US3) : à venir puis passées, pour toutes les fiches du
// compte.
export default function MyRegistrations() {
  const list = useQuery(myRegistrationListQuery());
  const items = list.data?.items ?? [];
  const now = Date.now();
  const upcoming = items.filter((r) => new Date(r.event.endsAt).getTime() > now);
  const past = items.filter((r) => new Date(r.event.endsAt).getTime() <= now);
  return (
    <Page
      title="Mes inscriptions"
      intro="Les inscriptions aux événements des membres que vous gérez."
    >
      {list.isError ? (
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger les inscriptions. Rechargez la page.
        </Alert>
      ) : list.isPending ? (
        <p className="text-muted">Chargement…</p>
      ) : items.length === 0 ? (
        <section className="max-w-3xl rounded-lg border border-line bg-surface p-6">
          <span className="grid h-11 w-11 place-items-center rounded-full bg-brand/10 text-brand-ink">
            <Icon icon={ClipboardCheck} />
          </span>
          <p className="mt-4 font-bold">Aucune inscription pour le moment.</p>
          <Link to="/evenements" className={`${buttonClasses()} mt-5`}>
            Voir les événements à venir
          </Link>
        </section>
      ) : (
        <div className="flex flex-col gap-8">
          <RegistrationTable title="À venir" items={upcoming} />
          <RegistrationTable title="Passées" items={past} />
        </div>
      )}
    </Page>
  );
}

function RegistrationTable({ title, items }: { title: string; items: MyRegistration[] }) {
  if (items.length === 0) return null;
  const id = `inscriptions-${title === "À venir" ? "a-venir" : "passees"}`;
  return (
    <section aria-labelledby={id}>
      <h2 id={id} className="mb-3 font-display text-lg font-bold">
        {title}
      </h2>
      <div className="rounded-lg border border-line bg-surface">
        <Table
          caption={`Inscriptions ${title.toLowerCase()}`}
          columns={[
            { key: "event", label: "Événement" },
            { key: "member", label: "Membre" },
            { key: "date", label: "Date" },
            { key: "state", label: "État" },
          ]}
        >
          {items.map((r) => (
            <TableRow key={r.id}>
              <th scope="row" className="px-5 py-3 text-left font-normal">
                <Link
                  to={`/mes-inscriptions/${r.id}`}
                  className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
                >
                  {r.event.title}
                </Link>
              </th>
              <TableCell>
                {r.member.firstName} {r.member.lastName}
              </TableCell>
              <TableCell className="whitespace-nowrap">
                {formatEventRange(r.event.startsAt, r.event.endsAt)}
              </TableCell>
              <TableCell>
                <span className="flex flex-wrap items-center gap-2">
                  <RegistrationBadges state={r.state} payment={r.payment} />
                  {r.actions.pay ? (
                    <Link
                      to={`/mes-inscriptions/${r.id}`}
                      className={buttonClasses({ size: "sm" })}
                    >
                      Payer
                    </Link>
                  ) : null}
                </span>
              </TableCell>
            </TableRow>
          ))}
        </Table>
      </div>
    </section>
  );
}
