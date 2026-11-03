import { useQuery } from "@tanstack/react-query";
import { Download, Paperclip } from "lucide-react";
import { useState } from "react";
import { useLocation } from "react-router";
import {
  attachmentDownloadHref,
  type Communication,
  communicationQuery,
} from "~/api/communications";
import { isProblem } from "~/api/problem";
import { queryClient } from "~/api/query-client";
import { DeliveryBadge, deliverySummary } from "~/communications/DeliveryBadge";
import { formatBytes } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Icon } from "~/ui/Icon";
import { Table, TableCell, TableRow } from "~/ui/Table";
import NotFound from "../not-found";
import type { Route } from "./+types/communication-detail";

export const handle = { crumb: "Détail" };

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(communicationQuery(id)).catch(() => null);
}

const dateTime = new Intl.DateTimeFormat("fr-FR", { dateStyle: "long", timeStyle: "short" });

// Détail d'une communication envoyée (contracts/ui-routes.md 009 : US1, US2, US4).
export default function CommunicationDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({
    ...communicationQuery(id ?? 0),
    enabled: id !== null,
    // Suivi des envois tant qu'il en reste en attente.
    refetchInterval: (q) => ((q.state.data?.delivery.pending ?? 0) > 0 ? 15_000 : false),
  });
  if (query.data) return <CommunicationView communication={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert">
        Impossible de charger cette communication. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 font-bold [overflow-wrap:anywhere]">{children}</dd>
    </div>
  );
}

function CommunicationView({ communication: c }: { communication: Communication }) {
  const location = useLocation();
  const [status] = useState(() => (location.state as { flash?: string } | null)?.flash ?? "");
  const by = c.author ? ` par ${c.author.email}` : "";
  const attachments = c.attachments ?? [];
  const recipients = c.recipients ?? [];
  return (
    <Page title={c.subject} intro={`Envoyée le ${dateTime.format(new Date(c.createdAt))}${by}.`}>
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6">
          {status}
        </Alert>
      ) : null}
      <div className="flex flex-col gap-6">
        <section aria-labelledby="cible" className="rounded-lg border border-line bg-surface p-5">
          <h2 id="cible" className="font-display text-lg font-bold">
            Cible
          </h2>
          <dl className="mt-3 grid gap-4 sm:grid-cols-2">
            <Info label="Saison">{c.season.label}</Info>
            <Info label="Cible">{c.targetLabel}</Info>
            <Info label="Adhérents ciblés">{c.membersCount}</Info>
            <Info label="Non joignables">{c.unreachableCount}</Info>
            <Info label="Destinataires">{c.recipientsCount}</Info>
          </dl>
        </section>

        <section aria-labelledby="message" className="rounded-lg border border-line bg-surface p-5">
          <h2 id="message" className="font-display text-lg font-bold">
            Message
          </h2>
          <p className="mt-3 whitespace-pre-wrap [overflow-wrap:anywhere]">{c.body}</p>
        </section>

        {attachments.length > 0 ? (
          <section
            aria-labelledby="pieces"
            className="rounded-lg border border-line bg-surface p-5"
          >
            <h2 id="pieces" className="font-display text-lg font-bold">
              Pièces jointes
            </h2>
            <ul className="mt-3 flex flex-col gap-2">
              {attachments.map((a) => (
                <li key={a.id} className="flex items-center gap-2">
                  <Icon icon={Paperclip} className="h-4 w-4 text-muted" />
                  <a
                    href={attachmentDownloadHref(c.id, a.id)}
                    download={a.name}
                    className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
                  >
                    {a.name}
                    <span className="sr-only"> (télécharger)</span>
                  </a>
                  <span className="text-sm text-muted">{formatBytes(a.sizeBytes)}</span>
                  <Icon icon={Download} className="h-4 w-4 text-muted" />
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        <section
          aria-labelledby="destinataires"
          className="rounded-lg border border-line bg-surface"
        >
          <div className="px-5 pt-4">
            <h2 id="destinataires" className="font-display text-lg font-bold">
              Destinataires
            </h2>
            <p className="mt-1 text-sm text-muted">{deliverySummary(c.delivery)}</p>
          </div>
          <div className="mt-3">
            <Table
              caption="Destinataires"
              columns={[
                { key: "adresse", label: "Adresse" },
                { key: "adherents", label: "Adhérents" },
                { key: "etat", label: "État" },
                { key: "date", label: "Envoyé le" },
              ]}
            >
              {recipients.map((r, i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: une adresse effacée n'a pas d'identifiant.
                <TableRow key={`${r.email ?? "efface"}-${i}`}>
                  <TableCell className="[overflow-wrap:anywhere]">
                    {r.email ?? <span className="text-muted">Adresse supprimée</span>}
                  </TableCell>
                  <TableCell>{r.membersCount}</TableCell>
                  <TableCell>
                    <DeliveryBadge status={r.status} />
                  </TableCell>
                  <TableCell className="whitespace-nowrap">
                    {r.sentAt ? dateTime.format(new Date(r.sentAt)) : "—"}
                  </TableCell>
                </TableRow>
              ))}
            </Table>
          </div>
        </section>
      </div>
    </Page>
  );
}
