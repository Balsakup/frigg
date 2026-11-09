import { useQuery } from "@tanstack/react-query";
import { CalendarDays, MapPin } from "lucide-react";
import { Link } from "react-router";
import { myEventListQuery } from "~/api/events";
import { queryClient } from "~/api/query-client";
import { dateBlock, formatEventRange } from "~/events/date";
import { EventBadges } from "~/events/EventBadges";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Icon } from "~/ui/Icon";

export async function clientLoader() {
  await queryClient.ensureQueryData(myEventListQuery()).catch(() => undefined);
  return null;
}

// « Événements à venir » (US2) : publiés ou annulés, du plus proche au plus
// lointain.
export default function MyEvents() {
  const list = useQuery(myEventListQuery());
  const items = list.data?.items ?? [];
  return (
    <Page title="Événements à venir" intro="Les activités proposées par l'association.">
      {list.isError ? (
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger les événements. Rechargez la page.
        </Alert>
      ) : list.isPending ? (
        <p className="text-muted">Chargement…</p>
      ) : items.length === 0 ? (
        <section className="max-w-3xl rounded-lg border border-line bg-surface p-6">
          <span className="grid h-11 w-11 place-items-center rounded-full bg-brand/10 text-brand-ink">
            <Icon icon={CalendarDays} />
          </span>
          <p className="mt-4 font-bold">Aucun événement à venir pour le moment.</p>
        </section>
      ) : (
        <ul className="flex max-w-3xl flex-col gap-4">
          {items.map((e) => {
            const [day, month] = dateBlock(e.startsAt);
            return (
              <li key={e.id} className="flex gap-4 rounded-lg border border-line bg-surface p-4">
                <div
                  aria-hidden="true"
                  className="grid h-16 w-16 shrink-0 place-content-center rounded-md bg-brand/10 text-center text-brand-ink"
                >
                  <span className="font-display text-2xl font-bold leading-none">{day}</span>
                  <span className="text-sm">{month}</span>
                </div>
                <div className="min-w-0 flex-1">
                  <h2 className="font-display text-lg font-bold [overflow-wrap:anywhere]">
                    <Link
                      to={`/evenements/${e.id}`}
                      className="text-brand-ink underline underline-offset-2"
                    >
                      {e.title}
                    </Link>
                  </h2>
                  <p className="mt-1">{formatEventRange(e.startsAt, e.endsAt)}</p>
                  <p className="mt-1 flex items-start gap-1.5 text-muted [overflow-wrap:anywhere]">
                    <Icon icon={MapPin} className="mt-0.5 h-4 w-4 shrink-0" />
                    {e.location}
                  </p>
                  <div className="mt-2 flex flex-wrap items-center gap-3">
                    <EventBadges event={e} showPhase={false} />
                    {e.placesLeft !== null &&
                    e.placesLeft !== undefined &&
                    e.placesLeft > 0 &&
                    e.phase === "upcoming" ? (
                      <span className="text-sm text-muted">
                        {e.placesLeft} place{e.placesLeft > 1 ? "s" : ""} restante
                        {e.placesLeft > 1 ? "s" : ""}
                      </span>
                    ) : null}
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </Page>
  );
}
