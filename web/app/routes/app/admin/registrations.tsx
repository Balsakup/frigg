import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { eventListQuery } from "~/api/events";
import { queryClient } from "~/api/query-client";
import { registrationPageQuery } from "~/api/registrations";
import { formatEventRange } from "~/events/date";
import { RegistrationBadges } from "~/events/RegistrationBadges";
import { hasSearchableTerm, SEARCH_MIN_LENGTH } from "~/lib/account-filters";
import { formatAmount, formatDate } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Checkbox } from "~/ui/Checkbox";
import { Field } from "~/ui/Field";
import { Pagination } from "~/ui/Pagination";
import { Select } from "~/ui/Select";
import { Table, TableCell, TableRow } from "~/ui/Table";
import type { Route } from "./+types/registrations";

type State = "" | "confirmed" | "cancelled";
type PaymentFilter = "" | "due" | "announced" | "paid" | "free";
const pageSize = 25;

function filtersOf(p: URLSearchParams) {
  const state = (p.get("etat") ?? "") as State;
  const payment = (p.get("paiement") ?? "") as PaymentFilter;
  const page = Number(p.get("page") ?? "1");
  return {
    event: Number(p.get("evenement")) || undefined,
    state: (["confirmed", "cancelled"].includes(state) ? state : "") as State,
    payment: (["due", "announced", "paid", "free"].includes(payment)
      ? payment
      : "") as PaymentFilter,
    alerts: p.get("alerts") === "open",
    q: p.get("q") ?? "",
    page: Number.isInteger(page) && page > 0 ? page : 1,
  };
}

function apiQuery(f: ReturnType<typeof filtersOf>) {
  return registrationPageQuery({
    event: f.event,
    state: f.state || undefined,
    payment: f.payment || undefined,
    alerts: f.alerts ? "open" : undefined,
    q: hasSearchableTerm(f.q) ? f.q : undefined,
    page: f.page,
    pageSize,
  });
}

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const f = filtersOf(new URL(request.url).searchParams);
  await Promise.all([
    queryClient.ensureQueryData(apiQuery(f)).catch(() => undefined),
    queryClient.ensureQueryData(eventListQuery({ season: "all" })).catch(() => undefined),
  ]);
  return null;
}

// Toutes les inscriptions (US5 §7) : filtres, recherche par nom ou référence.
export default function Registrations() {
  const [params, setParams] = useSearchParams();
  const f = filtersOf(params);
  const [draft, setDraft] = useState(f.q);
  const page = useQuery(apiQuery(f));
  const events = useQuery(eventListQuery({ season: "all" }));
  const tooShort = draft.trim() !== "" && !hasSearchableTerm(draft);

  const update = (changes: Record<string, string | undefined>) => {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(changes)) {
      if (!v) next.delete(k);
      else next.set(k, v);
    }
    next.delete("page");
    setParams(next, { replace: true });
  };

  useEffect(() => {
    const timer = setTimeout(() => {
      if (draft !== f.q && (draft.trim() === "" || hasSearchableTerm(draft)))
        update({ q: draft.trim() });
    }, 300);
    return () => clearTimeout(timer);
  });

  const data = page.data;
  const pageCount = data ? Math.max(1, Math.ceil(data.total / pageSize)) : 1;

  return (
    <Page
      title="Inscriptions"
      intro="Les inscriptions à tous les événements, les plus récentes d'abord."
    >
      <section
        aria-labelledby="inscriptions-titre"
        className="rounded-lg border border-line bg-surface"
      >
        <div className="px-5 pt-4">
          <h2 id="inscriptions-titre" className="font-display text-lg font-bold">
            Inscriptions
          </h2>
          <search className="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-[minmax(0,1fr)_14rem_10rem_11rem] lg:items-start">
            <Field
              label="Nom ou référence"
              type="search"
              autoComplete="off"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              hint={
                tooShort
                  ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères.`
                  : "Membre ou référence de paiement."
              }
            />
            <Select
              label="Événement"
              value={f.event ? String(f.event) : ""}
              onValueChange={(v) => update({ evenement: v })}
              items={[
                { value: "", label: "Tous les événements" },
                ...(events.data?.items ?? []).map((e) => ({ value: String(e.id), label: e.title })),
              ]}
            />
            <Select
              label="État"
              value={f.state}
              onValueChange={(v) => update({ etat: v })}
              items={[
                { value: "", label: "Tous" },
                { value: "confirmed", label: "Confirmées" },
                { value: "cancelled", label: "Annulées" },
              ]}
            />
            <Select
              label="Paiement"
              value={f.payment}
              onValueChange={(v) => update({ paiement: v })}
              items={[
                { value: "", label: "Tous" },
                { value: "due", label: "À payer" },
                { value: "announced", label: "Paiement annoncé" },
                { value: "paid", label: "Payées" },
                { value: "free", label: "Gratuites" },
              ]}
            />
          </search>
          <div className="mt-3">
            <Checkbox
              checked={f.alerts}
              onCheckedChange={(c) => update({ alerts: c ? "open" : undefined })}
            >
              Signalements de paiement à traiter
            </Checkbox>
          </div>
        </div>
        {page.isError ? (
          <Alert kind="danger" role="alert" className="m-5">
            Impossible de charger les inscriptions. Rechargez la page.
          </Alert>
        ) : !data ? (
          <p className="px-5 py-5 text-muted">Chargement…</p>
        ) : (data.items ?? []).length === 0 ? (
          <p className="px-5 py-5 text-muted">Aucune inscription ne correspond.</p>
        ) : (
          <div className="mt-4">
            <p className="px-5 text-sm text-muted" role="status">
              {data.total} inscription{data.total > 1 ? "s" : ""}
            </p>
            <Table
              caption="Inscriptions"
              columns={[
                { key: "member", label: "Membre" },
                { key: "event", label: "Événement" },
                { key: "date", label: "Inscription" },
                { key: "amount", label: "Montant" },
                { key: "state", label: "État" },
              ]}
            >
              {(data.items ?? []).map((r) => (
                <TableRow key={r.id}>
                  <th scope="row" className="px-5 py-3 text-left font-normal">
                    <Link
                      to={`/admin/inscriptions/${r.id}`}
                      className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
                    >
                      {r.member.lastName} {r.member.firstName}
                    </Link>
                    {r.openAlerts > 0 ? (
                      <span className="mt-1 block">
                        <Badge tone="danger">
                          {r.openAlerts} signalement{r.openAlerts > 1 ? "s" : ""}
                        </Badge>
                      </span>
                    ) : null}
                  </th>
                  <TableCell>
                    <Link
                      to={`/admin/evenements/${r.event.id}`}
                      className="underline underline-offset-2 [overflow-wrap:anywhere]"
                    >
                      {r.event.title}
                    </Link>
                    <span className="block text-sm text-muted">
                      {formatEventRange(r.event.startsAt, r.event.startsAt)}
                    </span>
                  </TableCell>
                  <TableCell className="whitespace-nowrap">{formatDate(r.registeredAt)}</TableCell>
                  <TableCell className="whitespace-nowrap">
                    {formatAmount(r.amountCents)}
                    {r.payment ? (
                      <span className="block text-sm text-muted">{r.payment.reference}</span>
                    ) : null}
                  </TableCell>
                  <TableCell>
                    <RegistrationBadges state={r.state} payment={r.payment} />
                  </TableCell>
                </TableRow>
              ))}
            </Table>
            <div className="px-5 py-4 empty:hidden">
              <Pagination
                page={Math.min(f.page, pageCount)}
                pageCount={pageCount}
                hrefFor={(p) => {
                  const next = new URLSearchParams(params);
                  next.set("page", String(p));
                  return `?${next.toString()}`;
                }}
              />
            </div>
          </div>
        )}
      </section>
    </Page>
  );
}
