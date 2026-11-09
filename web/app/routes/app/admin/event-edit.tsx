import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router";
import { type AdminEvent, eventQuery, invalidateEvents } from "~/api/events";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { EventForm } from "~/events/EventForm";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { buttonClasses } from "~/ui/Button";
import NotFound from "../not-found";
import type { Route } from "./+types/event-edit";

export const handle = { crumb: "Modifier" };

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(eventQuery(id)).catch(() => null);
}

// Modification d'un événement (US1).
export default function EventEdit({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...eventQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <EventEditForm event={query.data} />;
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

function EventEditForm({ event }: { event: AdminEvent }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const update = $api.useMutation("put", "/admin/events/{id}");
  const back = `/admin/evenements/${event.id}`;
  return (
    <Page title={`Modifier « ${event.title} »`} documentTitle="Modifier l'événement">
      <section className="max-w-3xl rounded-lg border border-line bg-surface p-5">
        {!event.actions.updateDates ? (
          <Alert kind="info" className="mb-5">
            Cet événement est terminé : ses dates ne peuvent plus changer.
          </Alert>
        ) : null}
        <EventForm
          event={event}
          submitLabel="Enregistrer"
          cancel={
            <Link to={back} className={buttonClasses({ variant: "secondary" })}>
              Annuler
            </Link>
          }
          onSubmit={async (body) => {
            await update.mutateAsync({ params: { path: { id: event.id } }, body });
            await invalidateEvents(qc);
            navigate(back, { state: { flash: "Événement enregistré." } });
          }}
        />
      </section>
    </Page>
  );
}
