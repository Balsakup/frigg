import { useQuery } from "@tanstack/react-query";
import { UserPlus } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router";
import { myMemberListQuery } from "~/api/members";
import { queryClient } from "~/api/query-client";
import { formatAge } from "~/lib/format";
import { fullName } from "~/members/form";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

export async function clientLoader() {
  await queryClient.ensureQueryData(myMemberListQuery()).catch(() => undefined);
  return null;
}

const MAX_NOTE = "20 membres au plus par compte. Contactez le bureau pour en ajouter.";

// « Mes membres » : les fiches gérées par le compte connecté (005, US1).
export default function MyMembers() {
  const location = useLocation();
  const [status] = useState(() => (location.state as { flash?: string } | null)?.flash ?? "");
  const list = useQuery(myMemberListQuery());
  const items = list.data?.items ?? [];
  const canAdd = list.data?.actions.add ?? false;

  const addLink = (primary: boolean) => (
    <Link
      to="/mes-membres/nouveau"
      className={buttonClasses({ variant: primary ? "primary" : "secondary" })}
    >
      <Icon icon={UserPlus} className="h-4 w-4" />
      Ajouter un membre
    </Link>
  );

  return (
    <Page
      title="Mes membres"
      intro="Les personnes que vous inscrivez à l'association : vous-même, vos enfants…"
      actions={items.length > 0 && canAdd ? addLink(false) : null}
    >
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}

      {list.isError ? (
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger vos membres. Rechargez la page.
        </Alert>
      ) : list.isPending ? null : items.length === 0 ? (
        <section className="max-w-3xl rounded-lg border border-line bg-surface p-5">
          <p>
            Un membre est une personne que vous inscrivez à l'association. Commencez par vous
            ajouter, ou ajoutez vos enfants.
          </p>
          <div className="mt-4">{addLink(true)}</div>
        </section>
      ) : (
        <>
          <ul aria-label="Mes membres" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {items.map((m) => (
              <li key={m.id} className="rounded-lg border border-line bg-surface p-5">
                <h2 className="font-display text-lg font-bold [overflow-wrap:anywhere]">
                  <Link
                    to={`/mes-membres/${m.id}`}
                    className="text-brand-ink underline underline-offset-2"
                  >
                    {fullName(m)}
                  </Link>
                </h2>
                <p className="mt-1 flex flex-wrap items-center gap-2 text-muted">
                  {formatAge(m.age)}
                  {m.isSelf ? <Badge tone="brand">Vous</Badge> : null}
                </p>
              </li>
            ))}
          </ul>
          {!canAdd ? <p className="mt-4 text-muted">{MAX_NOTE}</p> : null}
        </>
      )}
    </Page>
  );
}
