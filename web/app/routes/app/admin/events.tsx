import { useQuery } from "@tanstack/react-query";
import { CalendarPlus, Plus } from "lucide-react";
import { useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router";
import { type EventStatusFilter, type EventSummary, eventListQuery } from "~/api/events";
import { queryClient } from "~/api/query-client";
import { seasonListQuery } from "~/api/seasons";
import { formatEventRange } from "~/events/date";
import { EventBadges } from "~/events/EventBadges";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { Select } from "~/ui/Select";
import { Table, TableCell, TableRow } from "~/ui/Table";
import type { Route } from "./+types/events";

const statusItems: { value: EventStatusFilter | "all"; label: string }[] = [
  { value: "all", label: "Tous les états" },
  { value: "draft", label: "Brouillons" },
  { value: "published", label: "Publiés" },
  { value: "cancelled", label: "Annulés" },
  { value: "finished", label: "Terminés" },
];

function filtersOf(params: URLSearchParams) {
  const status = params.get("etat") as EventStatusFilter | null;
  return {
    season: params.get("saison") ?? undefined,
    status: statusItems.some((s) => s.value === status) && status ? status : undefined,
  };
}

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const filters = filtersOf(new URL(request.url).searchParams);
  await Promise.all([
    queryClient.ensureQueryData(eventListQuery(filters)).catch(() => undefined),
    queryClient.ensureQueryData(seasonListQuery()).catch(() => undefined),
  ]);
  return null;
}

// Liste des événements (US1) : à venir puis passés, filtres saison et état.
export default function Events() {
  const location = useLocation();
  const [status] = useState(() => (location.state as { flash?: string } | null)?.flash ?? "");
  const [params, setParams] = useSearchParams();
  const filters = filtersOf(params);
  const list = useQuery(eventListQuery(filters));
  const seasons = useQuery(seasonListQuery());
  const items = list.data?.items ?? [];
  const now = Date.now();
  const upcoming = items.filter((e) => new Date(e.endsAt).getTime() > now);
  const past = items.filter((e) => new Date(e.endsAt).getTime() <= now).reverse();
  const filtered = Boolean(filters.season || filters.status);

  const setFilter = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value === "" || value === "all") next.delete(key);
    else next.set(key, value);
    setParams(next, { replace: true });
  };

  const create = (
    <Link to="/admin/evenements/nouveau" className={buttonClasses()}>
      <Icon icon={Plus} className="h-4 w-4" />
      Créer un événement
    </Link>
  );

  return (
    <Page
      title="Événements"
      intro="Activités de l'association : assemblée générale, sorties, stages, tournois…"
      actions={items.length > 0 || filtered ? create : undefined}
    >
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}
      {items.length > 0 || filtered ? (
        <div className="mb-6 flex flex-wrap gap-4">
          <Select
            label="Saison"
            value={filters.season ?? ""}
            onValueChange={(v) => setFilter("saison", v)}
            items={[
              { value: "", label: "Saison courante" },
              { value: "all", label: "Toutes les saisons" },
              ...(seasons.data?.items ?? []).map((s) => ({
                value: String(s.id),
                label: `Saison ${s.label}`,
              })),
            ]}
            className="w-56"
          />
          <Select
            label="État"
            value={filters.status ?? "all"}
            onValueChange={(v) => setFilter("etat", v)}
            items={statusItems}
            className="w-56"
          />
        </div>
      ) : null}
      {list.isError ? (
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger les événements. Rechargez la page.
        </Alert>
      ) : list.isPending ? (
        <p className="text-muted">Chargement…</p>
      ) : items.length === 0 && !filtered ? (
        <section
          aria-labelledby="evenements-vide"
          className="max-w-3xl rounded-lg border border-line bg-surface p-6"
        >
          <span className="grid h-11 w-11 place-items-center rounded-full bg-brand/10 text-brand-ink">
            <Icon icon={CalendarPlus} />
          </span>
          <h2 id="evenements-vide" className="mt-4 font-display text-lg font-bold">
            Aucun événement pour le moment
          </h2>
          <p className="mt-1 max-w-prose text-muted">
            Un événement réunit les membres autour d'une activité. Publiez-le pour que les membres
            le voient et s'y inscrivent, gratuitement ou contre paiement.
          </p>
          <Link to="/admin/evenements/nouveau" className={`${buttonClasses()} mt-5`}>
            <Icon icon={Plus} className="h-4 w-4" />
            Créer un événement
          </Link>
        </section>
      ) : (
        <div className="flex flex-col gap-8">
          <EventTable title="À venir" events={upcoming} empty="Aucun événement à venir." />
          <EventTable title="Passés" events={past} empty="Aucun événement passé." />
        </div>
      )}
    </Page>
  );
}

function EventTable({
  title,
  events,
  empty,
}: {
  title: string;
  events: EventSummary[];
  empty: string;
}) {
  const id = `evenements-${title === "À venir" ? "a-venir" : "passes"}`;
  return (
    <section aria-labelledby={id}>
      <h2 id={id} className="mb-3 font-display text-lg font-bold">
        {title}
      </h2>
      {events.length === 0 ? (
        <p className="text-muted">{empty}</p>
      ) : (
        <div className="rounded-lg border border-line bg-surface">
          <Table
            caption={`Événements ${title.toLowerCase()}`}
            columns={[
              { key: "title", label: "Événement" },
              { key: "date", label: "Date" },
              { key: "location", label: "Lieu" },
              { key: "state", label: "État" },
              { key: "count", label: "Inscrits" },
            ]}
          >
            {events.map((e) => (
              <TableRow key={e.id}>
                <th scope="row" className="px-5 py-3 text-left font-normal">
                  <Link
                    to={`/admin/evenements/${e.id}`}
                    className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
                  >
                    {e.title}
                  </Link>
                </th>
                <TableCell className="whitespace-nowrap">
                  {formatEventRange(e.startsAt, e.endsAt)}
                </TableCell>
                <TableCell className="[overflow-wrap:anywhere]">{e.location}</TableCell>
                <TableCell>
                  <EventBadges event={e} />
                </TableCell>
                <TableCell className="whitespace-nowrap">
                  {e.confirmedCount}
                  {e.capacity ? ` / ${e.capacity}` : ""}
                </TableCell>
              </TableRow>
            ))}
          </Table>
        </div>
      )}
    </section>
  );
}
