import { useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { type MyEvent, myEventQuery } from "~/api/events";
import { isProblem } from "~/api/problem";
import { queryClient } from "~/api/query-client";
import { formatEventMoment, formatEventRange } from "~/events/date";
import { EventBadges } from "~/events/EventBadges";
import { RegisterMembers } from "~/events/RegisterMembers";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import NotFound from "../not-found";
import type { Route } from "./+types/event-detail";

export const handle = {
  crumb: (data: unknown) => {
    const e = data as MyEvent | null;
    return e ? e.title : "Événement introuvable";
  },
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(myEventQuery(id)).catch(() => null);
}

// Page d'un événement et inscription des fiches du compte (US2, US3). Un
// brouillon donne « Page introuvable ».
export default function MyEventDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...myEventQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <EventView event={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger cet événement. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function EventView({ event: e }: { event: MyEvent }) {
  const [status, setStatus] = useState("");
  const heading = useRef<HTMLDivElement>(null);
  const closedReason =
    e.phase === "cancelled"
      ? null
      : e.window && new Date(e.window.opensAt).getTime() > Date.now()
        ? `Les inscriptions ouvriront le ${formatEventMoment(e.window.opensAt)}.`
        : "Les inscriptions sont closes.";

  return (
    <Page title={e.title} subtitle={formatEventRange(e.startsAt, e.endsAt)}>
      <div ref={heading} tabIndex={-1} className="outline-none">
        <p role="status" className="sr-only">
          {status}
        </p>
        {status ? (
          <Alert kind="success" className="mb-6 max-w-3xl">
            {status}
          </Alert>
        ) : null}
      </div>
      {e.phase === "cancelled" ? (
        <Alert kind="danger" className="mb-6 max-w-3xl" title="Événement annulé">
          {e.cancelMessage ?? "Cet événement n'aura pas lieu."}
        </Alert>
      ) : null}
      <div className="flex max-w-3xl flex-col gap-6">
        <section
          aria-labelledby="evenement-infos"
          className="rounded-lg border border-line bg-surface p-5"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 id="evenement-infos" className="font-display text-lg font-bold">
              Informations
            </h2>
            <EventBadges event={e} showPhase={false} />
          </div>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <div>
              <dt className="text-sm text-muted">Date</dt>
              <dd className="mt-0.5 font-bold">{formatEventRange(e.startsAt, e.endsAt)}</dd>
            </div>
            <div className="min-w-0">
              <dt className="text-sm text-muted">Lieu</dt>
              <dd className="mt-0.5 font-bold [overflow-wrap:anywhere]">{e.location}</dd>
            </div>
          </dl>
          {e.description ? (
            <p className="mt-4 whitespace-pre-line [overflow-wrap:anywhere]">{e.description}</p>
          ) : null}
        </section>
        {e.phase !== "cancelled" ? (
          <section
            aria-labelledby="evenement-inscriptions"
            className="rounded-lg border border-brand-ink/40 bg-surface p-5"
          >
            <h2 id="evenement-inscriptions" className="font-display text-lg font-bold">
              Inscriptions
            </h2>
            <p className="mt-2 text-muted">
              {e.openness === "all_members"
                ? "Ouvert à tous les membres."
                : `Réservé aux adhérents de la saison ${e.season.label}.`}{" "}
              {e.registrationOpen && e.window
                ? `Inscriptions ouvertes jusqu'au ${formatEventMoment(e.window.closesAt)}.`
                : closedReason}
            </p>
            <div className="mt-4">
              <RegisterMembers
                event={e}
                onStatus={(m) => {
                  setStatus(m);
                  heading.current?.focus();
                }}
              />
            </div>
          </section>
        ) : null}
      </div>
    </Page>
  );
}
