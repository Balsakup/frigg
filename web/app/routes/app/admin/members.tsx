import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { Trash2, Upload, UserPlus, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router";
import { customFieldsQuery } from "~/api/custom-fields";
import {
  INACTIVE_DELETE_MAX,
  inactiveMemberIds,
  invalidateMembers,
  memberListQuery,
  useDeleteInactiveMembers,
} from "~/api/members";
import { invalidateMemberships } from "~/api/memberships";
import { queryClient } from "~/api/query-client";
import { CustomFieldFilters, filterableFields } from "~/customfields/CustomFieldFilters";
import { hasSearchableTerm, SEARCH_MIN_LENGTH } from "~/lib/account-filters";
import { formatAge } from "~/lib/format";
import {
  hasActiveMemberFilters,
  type MemberFilters,
  type MemberOrigin,
  parseMemberFilters,
  toMemberSearchParams,
} from "~/lib/member-filters";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button, buttonClasses } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
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

const originItems: { value: MemberOrigin; label: string }[] = [
  { value: "", label: "Toutes" },
  { value: "public", label: "Publique" },
  { value: "inactive", label: "Inactives à supprimer" },
];

// Rubrique « Membres » : toutes les fiches de l'association (005, US3).
export default function Members() {
  const location = useLocation();
  const [status, setStatus] = useState(
    () => (location.state as { flash?: string } | null)?.flash ?? "",
  );
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
          <search className="mt-3 grid gap-4 sm:grid-cols-[minmax(0,1fr)_11rem_11rem] sm:items-start">
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
              label="Origine"
              value={filters.origin ?? ""}
              items={originItems}
              onValueChange={(v) => navigate({ ...filters, q: draft, origin: v, page: 1 })}
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
              <div className="grid gap-4 sm:col-span-3 sm:grid-cols-2 lg:grid-cols-4">
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
            {filters.origin === "inactive" && total > 0 ? (
              <DeleteInactive total={total} onDone={setStatus} />
            ) : null}
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
                  {m.origin === "public" ? (
                    <span className="ml-2 inline-flex flex-wrap gap-1">
                      <Badge tone="info">Publique</Badge>
                      {m.inactive ? <Badge tone="warning">Inactive</Badge> : null}
                    </span>
                  ) : null}
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

// Suppression en lot des fiches publiques inactives listées (016, FR-041) :
// toujours sur décision d'un administrateur, après confirmation. Chaque fiche
// est revérifiée par le serveur ; celles redevenues actives sont ignorées.
function DeleteInactive({ total, onDone }: { total: number; onDone: (message: string) => void }) {
  const qc = useQueryClient();
  const remove = useDeleteInactiveMembers();
  const [error, setError] = useState("");
  const count = Math.min(total, INACTIVE_DELETE_MAX);
  const label = count > 1 ? `Supprimer les ${count} fiches` : "Supprimer la fiche";

  const onConfirm = async () => {
    setError("");
    try {
      const ids = await inactiveMemberIds();
      if (ids.length === 0) return;
      const res = await remove.mutateAsync({ body: { ids } });
      await Promise.all([invalidateMembers(qc), invalidateMemberships(qc)]);
      const parts = [
        `${res.deleted} fiche${res.deleted > 1 ? "s" : ""} supprimée${res.deleted > 1 ? "s" : ""}.`,
      ];
      if (res.skipped > 0) {
        parts.push(
          `${res.skipped} fiche${res.skipped > 1 ? "s" : ""} ignorée${res.skipped > 1 ? "s" : ""} : redevenue${res.skipped > 1 ? "s" : ""} active${res.skipped > 1 ? "s" : ""} ou déjà supprimée${res.skipped > 1 ? "s" : ""}.`,
        );
      }
      onDone(parts.join(" "));
    } catch {
      setError("La suppression a échoué. Réessayez.");
    }
  };

  return (
    <>
      <ConfirmDialog
        title={`${label} ?`}
        confirmLabel={label}
        onConfirm={onConfirm}
        trigger={
          <Button
            variant="danger-secondary"
            size="sm"
            type="button"
            aria-disabled={remove.isPending}
          >
            <Icon icon={Trash2} className="h-4 w-4" />
            {label}
          </Button>
        }
      >
        <p>
          {count > 1 ? `Les ${count} fiches publiques inactives` : "La fiche publique inactive"} et
          leurs adhésions et inscriptions seront effacées définitivement. Une fiche redevenue active
          entre-temps sera conservée.
        </p>
        {total > INACTIVE_DELETE_MAX ? (
          <p className="mt-2">
            Seules les {INACTIVE_DELETE_MAX} premières fiches seront supprimées ; recommencez pour
            les suivantes.
          </p>
        ) : null}
      </ConfirmDialog>
      {error ? (
        <p role="alert" className="w-full text-sm font-bold text-danger">
          {error}
        </p>
      ) : null}
    </>
  );
}
