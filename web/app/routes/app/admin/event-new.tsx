import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router";
import { invalidateEvents } from "~/api/events";
import { $api } from "~/api/queries";
import { EventForm } from "~/events/EventForm";
import { Page } from "~/shell/Page";
import { buttonClasses } from "~/ui/Button";

export const handle = { crumb: "Nouvel événement" };

// Création d'un événement en brouillon (US1).
export default function EventNew() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const create = $api.useMutation("post", "/admin/events");
  return (
    <Page
      title="Nouvel événement"
      intro="L'événement est créé en brouillon : les membres ne le voient qu'une fois publié."
    >
      <section className="max-w-3xl rounded-lg border border-line bg-surface p-5">
        <EventForm
          submitLabel="Enregistrer le brouillon"
          cancel={
            <Link to="/admin/evenements" className={buttonClasses({ variant: "secondary" })}>
              Annuler
            </Link>
          }
          onSubmit={async (body) => {
            const created = await create.mutateAsync({ body });
            await invalidateEvents(qc);
            navigate(`/admin/evenements/${created.id}`, {
              state: { flash: "Événement enregistré en brouillon." },
            });
          }}
        />
      </section>
    </Page>
  );
}
