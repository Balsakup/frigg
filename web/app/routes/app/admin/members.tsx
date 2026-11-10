import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Upload, UserPlus, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router";
import { customFieldsQuery } from "~/api/custom-fields";
import { memberListQuery } from "~/api/members";
import { queryClient } from "~/api/query-client";
import { CustomFieldFilters, filterableFields } from "~/customfields/CustomFieldFilters";
import { hasSearchableTerm, SEARCH_MIN_LENGTH } from "~/lib/account-filters";
import { formatAge } from "~/lib/format";
import {
  hasActiveMemberFilters,
  type MemberFilters,
  parseMemberFilters,
  toMemberSearchParams,
} from "~/lib/member-filters";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button, buttonClasses } from "~/ui/Button";
import { Field } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { Pagination } from "~/ui/Pagination";
import { Select } from "~/ui/Select";
import { Table, TableCell, TableRow } from "~/ui/Table";
import type { Route } from "./+types/members";

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const filters = parseMemberFilters(new URL(request.url).searchParams);
  await Promise.all([
    queryClient.ensureQueryData(memberListQuery(filters)).catch(() => undefined),
    queryClient.ensureQueryData(customFieldsQuery()).catch(() => undefined),
  ]);
  return null;
}

const accountItems: { value: "all" | "none"; label: string }[] = [
  { value: "all", label: "Tous" },
  { value: "none", label: "Sans compte" },
];

// Rubrique « Membres » : toutes les fiches de l'association (005, US3).
export default function Members() {
  const location = useLocation();
  const [status] = useState(() => (location.state as { flash?: string } | null)?.flash ?? "");
  const [searchParams, setSearchParams] = useSearchParams();
  const filters = parseMemberFilters(searchParams);
  const [draft, setDraft] = useState(filters.q);
  const list = useQuery({ ...memberListQuery(filters), placeholderData: keepPreviousData });
  const definitions = useQuery(customFieldsQuery());
  const filterable = filterableFields(definitions.data?.member);

  const navigate = (next: MemberFilters, replace = false) => {
    setSearchParams(toMemberSearchParams(next), { replace, preventScrollReset: true });
  };

  // La recherche suit la saisie après 300 ms sans frappe, sans remplir l'historique.
  useEffect(() => {
    if (draft.trim() === filters.q.trim()) return;
    const timer = setTimeout(() => navigate({ ...filters, q: draft, page: 1 }, true), 300);
    return () => clearTimeout(timer);
  });

  const clear = () => {
    setDraft("");
    navigate({ q: "", noAccount: false, fields: [], page: 1 });
  };

  const tooShort = draft.trim() !== "" && !hasSearchableTerm(draft);
  const items = list.data?.items ?? [];
  const total = list.data?.total ?? 0;
  const pageSize = list.data?.pageSize ?? 25;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const summary = list.isPending
    ? "Chargement…"
    : total === 0
      ? "Aucun membre ne correspond à ces critères."
      : `${total} membre${total > 1 ? "s" : ""}`;

  return (
    <Page
      title="Membres"
      intro="Toutes les personnes de l'association."
      actions={
        <>
          <Link to="/admin/membres/importer" className={buttonClasses({ variant: "secondary" })}>
            <Icon icon={Upload} className="h-4 w-4" />
            Importer
          </Link>
          <Link to="/admin/membres/nouveau" className={buttonClasses({})}>
            <Icon icon={UserPlus} className="h-4 w-4" />
            Ajouter un membre
          </Link>
        </>
      }
    >
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6">
          {status}
        </Alert>
      ) : null}

      <section aria-labelledby="membres-titre" className="rounded-lg border border-line bg-surface">
        <div className="px-5 pt-4">
          <h2 id="membres-titre" className="font-display text-lg font-bold">
            Membres de l'association
          </h2>
          <search className="mt-3 grid gap-4 sm:grid-cols-[minmax(0,1fr)_11rem] sm:items-start">
            <Field
              label="Rechercher un membre"
              type="search"
              autoComplete="off"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              hint={
                tooShort
                  ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères.`
                  : "Nom, prénom, adresse électronique, téléphone ou champ personnalisé texte."
              }
            />
            <Select
              label="Rattachement"
              value={filters.noAccount ? "none" : "all"}
              items={accountItems}
              onValueChange={(v) =>
                navigate({ ...filters, q: draft, noAccount: v === "none", page: 1 })
              }
            />
            {filterable.length ? (
              <div className="grid gap-4 sm:col-span-2 sm:grid-cols-2 lg:grid-cols-4">
                <CustomFieldFilters
                  fields={filterable}
                  value={filters.fields ?? []}
                  onChange={(fields) => navigate({ ...filters, q: draft, fields, page: 1 })}
                />
              </div>
            ) : null}
          </search>
          <div className="mt-3 flex min-h-10 flex-wrap items-center justify-between gap-2">
            <p role="status" className="text-sm text-muted">
              {summary}
            </p>
            {hasActiveMemberFilters(filters) || draft !== "" ? (
              <Button variant="tertiary" size="sm" onClick={clear}>
                <Icon icon={X} className="h-4 w-4" />
                Effacer la recherche
              </Button>
            ) : null}
          </div>
        </div>
        {list.isError ? (
          <Alert kind="danger" className="m-5">
            Impossible de charger les membres. Rechargez la page.
          </Alert>
        ) : items.length > 0 ? (
          <Table
            caption="Membres de l'association"
            columns={[
              { key: "name", label: "Nom" },
              { key: "age", label: "Âge" },
              { key: "account", label: "Compte" },
            ]}
          >
            {items.map((m) => (
              <TableRow key={m.id}>
                <th
                  scope="row"
                  className="px-5 py-3 text-left font-normal [overflow-wrap:anywhere]"
                >
                  <Link
                    to={`/admin/membres/${m.id}`}
                    className="font-bold text-brand-ink underline underline-offset-2"
                  >
                    {m.lastName} {m.firstName}
                  </Link>
                </th>
                <TableCell className="whitespace-nowrap">{formatAge(m.age)}</TableCell>
                <TableCell className="[overflow-wrap:anywhere]">
                  {m.account ? (
                    <span className="flex flex-wrap items-center gap-2">
                      {m.account.email}
                      {m.account.disabled ? <Badge tone="muted">désactivé</Badge> : null}
                    </span>
                  ) : (
                    <span className="text-muted">Sans compte</span>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </Table>
        ) : null}
        <div className="px-5 py-4 empty:hidden">
          <Pagination
            page={Math.min(filters.page, pageCount)}
            pageCount={pageCount}
            hrefFor={(page) => {
              const params = toMemberSearchParams({ ...filters, page }).toString();
              return params ? `?${params}` : "?";
            }}
          />
        </div>
      </section>
    </Page>
  );
}
