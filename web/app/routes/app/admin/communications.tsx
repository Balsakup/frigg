import { useQuery } from "@tanstack/react-query";
import { Mail, Plus } from "lucide-react";
import { Link, useSearchParams } from "react-router";
import { communicationsQuery } from "~/api/communications";
import { queryClient } from "~/api/query-client";
import { deliverySummary, deliveryTone } from "~/communications/DeliveryBadge";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { Pagination } from "~/ui/Pagination";
import { Table, TableCell, TableRow } from "~/ui/Table";
import type { Route } from "./+types/communications";

function pageOf(params: URLSearchParams) {
  const n = Number(params.get("page"));
  return Number.isInteger(n) && n > 0 ? n : 1;
}

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const page = pageOf(new URL(request.url).searchParams);
  await queryClient.ensureQueryData(communicationsQuery(page)).catch(() => undefined);
  return null;
}

const PAGE_SIZE = 25;
const dateTime = new Intl.DateTimeFormat("fr-FR", { dateStyle: "medium", timeStyle: "short" });

const newLink = (
  <Link to="/admin/communication/nouvelle" className={buttonClasses({})}>
    <Icon icon={Plus} className="h-4 w-4" />
    Nouvelle communication
  </Link>
);

// Historique des communications (contracts/ui-routes.md 009 : US4).
export default function Communications() {
  const [params] = useSearchParams();
  const page = pageOf(params);
  const list = useQuery(communicationsQuery(page));
  const items = list.data?.items ?? [];
  const total = list.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <Page
      title="Communication"
      intro="Écrivez aux adhérents d'une saison et retrouvez les messages envoyés."
      actions={newLink}
    >
      {list.isError ? (
        <Alert kind="danger" role="alert">
          Impossible de charger les communications. Rechargez la page.
        </Alert>
      ) : list.isPending ? null : total === 0 ? (
        <div className="flex max-w-prose items-start gap-3 rounded-lg border-2 border-dashed border-line-strong/60 bg-surface p-6">
          <span className="text-muted">
            <Icon icon={Mail} />
          </span>
          <div>
            <h2 className="font-display text-lg font-bold">Aucune communication pour l'instant</h2>
            <p className="mt-1 text-muted">
              Relancez les cotisations impayées, convoquez l'assemblée générale ou informez les
              adhérents d'une saison : chaque destinataire reçoit son propre e-mail, pièces jointes
              comprises.
            </p>
            <div className="mt-4">{newLink}</div>
          </div>
        </div>
      ) : (
        <section aria-labelledby="historique" className="rounded-lg border border-line bg-surface">
          <div className="px-5 pt-4">
            <h2 id="historique" className="font-display text-lg font-bold">
              Communications envoyées
            </h2>
            <p className="mt-1 text-sm text-muted">
              {total} communication{total > 1 ? "s" : ""}
            </p>
          </div>
          <div className="mt-3">
            <Table
              caption="Communications envoyées"
              columns={[
                { key: "date", label: "Date" },
                { key: "objet", label: "Objet" },
                { key: "cible", label: "Cible" },
                { key: "auteur", label: "Auteur" },
                { key: "destinataires", label: "Destinataires" },
                { key: "envoi", label: "Envoi" },
              ]}
            >
              {items.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="whitespace-nowrap">
                    {dateTime.format(new Date(c.createdAt))}
                  </TableCell>
                  <TableCell className="min-w-48">
                    <Link
                      to={`/admin/communication/${c.id}`}
                      className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
                    >
                      {c.subject}
                    </Link>
                  </TableCell>
                  <TableCell className="min-w-40">{c.targetLabel}</TableCell>
                  <TableCell className="[overflow-wrap:anywhere]">
                    {c.author ? (
                      c.author.email
                    ) : (
                      <span className="text-muted">Compte supprimé</span>
                    )}
                  </TableCell>
                  <TableCell>{c.recipientsCount}</TableCell>
                  <TableCell>
                    <Badge tone={deliveryTone(c.delivery)}>{deliverySummary(c.delivery)}</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </Table>
          </div>
          <div className="p-5">
            <Pagination page={page} pageCount={pageCount} hrefFor={(p) => `?page=${p}`} />
          </div>
        </section>
      )}
    </Page>
  );
}
