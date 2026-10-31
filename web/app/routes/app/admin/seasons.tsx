import { useQuery } from "@tanstack/react-query";
import { CalendarRange, Plus } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router";
import { queryClient } from "~/api/query-client";
import { seasonListQuery } from "~/api/seasons";
import { formatDay, seasonStatusLabel, seasonStatusTone } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { Table, TableCell, TableRow } from "~/ui/Table";

// Préchargement ; une erreur (403) est affichée par le layout admin.
export async function clientLoader() {
  await queryClient.ensureQueryData(seasonListQuery()).catch(() => undefined);
  return null;
}

// Liste des saisons, de la plus récente à la plus ancienne (FR-020), ou état
// vide (FR-021).
export default function Seasons() {
  const location = useLocation();
  // Message transmis par la fiche après une suppression.
  const [status] = useState(() => (location.state as { flash?: string } | null)?.flash ?? "");
  const list = useQuery(seasonListQuery());
  const items = list.data?.items ?? [];
  const newSeason = (
    <Link to="/admin/saisons/nouvelle" className={buttonClasses()}>
      <Icon icon={Plus} className="h-4 w-4" />
      Nouvelle saison
    </Link>
  );

  return (
    <Page
      title="Saisons"
      intro="Périodes d'activité de l'association, qui servent aussi d'exercice comptable."
      actions={items.length > 0 ? newSeason : undefined}
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
          Impossible de charger les saisons. Rechargez la page.
        </Alert>
      ) : list.isPending ? (
        <p className="text-muted">Chargement…</p>
      ) : items.length === 0 ? (
        <section
          aria-labelledby="saisons-vide-titre"
          className="max-w-3xl rounded-lg border border-line bg-surface p-6"
        >
          <span className="grid h-11 w-11 place-items-center rounded-full bg-brand/10 text-brand-ink">
            <Icon icon={CalendarRange} />
          </span>
          <h2 id="saisons-vide-titre" className="mt-4 font-display text-lg font-bold">
            Aucune saison pour le moment
          </h2>
          <p className="mt-1 max-w-prose text-muted">
            Une saison délimite une période d'activité : adhésions, événements et comptes de l'année
            s'y rattachent. Créez la première pour mettre en place Frigg.
          </p>
          <Link to="/admin/saisons/nouvelle" className={`${buttonClasses()} mt-5`}>
            <Icon icon={Plus} className="h-4 w-4" />
            Créer la première saison
          </Link>
        </section>
      ) : (
        <section className="max-w-4xl rounded-lg border border-line bg-surface">
          <Table
            caption="Saisons de l'association, de la plus récente à la plus ancienne"
            columns={[
              { key: "label", label: "Saison" },
              { key: "start", label: "Début" },
              { key: "end", label: "Fin" },
              { key: "status", label: "État" },
            ]}
          >
            {items.map((s) => (
              <TableRow key={s.id}>
                <th scope="row" className="px-5 py-3 text-left font-normal">
                  <span className="flex flex-wrap items-center gap-2">
                    <Link
                      to={`/admin/saisons/${s.id}`}
                      className="whitespace-nowrap font-bold text-brand-ink underline underline-offset-2"
                    >
                      Saison {s.label}
                    </Link>
                    {s.isCurrent ? <Badge tone="brand">Saison courante</Badge> : null}
                  </span>
                </th>
                <TableCell className="whitespace-nowrap">{formatDay(s.startsOn)}</TableCell>
                <TableCell className="whitespace-nowrap">{formatDay(s.endsOn)}</TableCell>
                <TableCell>
                  <Badge tone={seasonStatusTone[s.status]}>{seasonStatusLabel(s.status)}</Badge>
                </TableCell>
              </TableRow>
            ))}
          </Table>
        </section>
      )}
    </Page>
  );
}
