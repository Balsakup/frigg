import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { RotateCw, Send, Trash2, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useLocation, useSearchParams } from "react-router";
import { accountListQuery } from "~/api/accounts";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import {
  type AccountFilters,
  hasActiveFilters,
  hasSearchableTerm,
  parseFilters,
  type RoleFilter,
  SEARCH_MIN_LENGTH,
  type StatusFilter,
  toSearchParams,
} from "~/lib/account-filters";
import { formatDate, roleLabel } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Field } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { Pagination } from "~/ui/Pagination";
import { Select } from "~/ui/Select";
import { Table, TableCell, TableRow } from "~/ui/Table";
import type { Route } from "./+types/accounts";

// Préchargement ; une erreur (403 d'un membre, droits retirés) est affichée par
// la page ou le layout admin plutôt que par l'erreur globale de l'application.
export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const filters = parseFilters(new URL(request.url).searchParams);
  await queryClient.ensureQueryData(accountListQuery(filters)).catch(() => undefined);
  return null;
}

interface InviteForm {
  email: string;
  role: "member" | "admin";
}

// Rubrique « Comptes » : liste des comptes (002) et invitations par e-mail (001, US4).
export default function Accounts() {
  const queryClient = useQueryClient();
  const location = useLocation();
  // Message transmis par la fiche après une suppression.
  const [status, setStatus] = useState(
    () => (location.state as { flash?: string } | null)?.flash ?? "",
  );
  const list = $api.useQuery("get", "/admin/invitations");
  const listKey = $api.queryOptions("get", "/admin/invitations").queryKey;
  const create = $api.useMutation("post", "/admin/invitations");
  const resend = $api.useMutation("post", "/admin/invitations/{id}/resend");
  const cancel = $api.useMutation("delete", "/admin/invitations/{id}");
  const { register, handleSubmit, setError, reset, formState } = useForm<InviteForm>({
    defaultValues: { email: "", role: "member" },
  });

  const onSubmit = handleSubmit(async (body) => {
    setStatus("");
    try {
      const inv = await create.mutateAsync({ body });
      reset();
      setStatus(`Invitation envoyée à ${inv.email}.`);
      await queryClient.invalidateQueries({ queryKey: listKey });
    } catch (error) {
      if (isProblem(error)) applyProblemToForm(error, setError, ["email", "role"]);
    }
  });

  const onResend = async (id: number, email: string) => {
    setStatus("");
    await resend.mutateAsync({ params: { path: { id } } });
    setStatus(`Invitation renvoyée à ${email}.`);
    await queryClient.invalidateQueries({ queryKey: listKey });
  };

  const onCancel = async (id: number, email: string) => {
    setStatus("");
    await cancel.mutateAsync({ params: { path: { id } } });
    setStatus(`Invitation de ${email} annulée.`);
    await queryClient.invalidateQueries({ queryKey: listKey });
  };

  const items = list.data?.items ?? [];

  return (
    <Page
      title="Comptes"
      intro="Gérez les comptes de l'association et invitez de nouvelles personnes."
    >
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}

      <AccountList />

      <section
        aria-labelledby="inviter-titre"
        className="mt-6 max-w-3xl rounded-lg border border-line bg-surface p-5"
      >
        <h2 id="inviter-titre" className="font-display text-lg font-bold">
          Inviter une personne
        </h2>
        <p className="mt-1 text-muted">
          Elle recevra un e-mail pour choisir son mot de passe. Le lien est valable 7 jours.
        </p>
        {formState.errors.root?.server ? (
          <Alert kind="danger" className="mt-4" role="alert">
            {formState.errors.root.server.message}
          </Alert>
        ) : null}
        <form
          onSubmit={onSubmit}
          noValidate
          className="mt-4 grid gap-4 sm:grid-cols-[minmax(0,1fr)_12rem_auto] sm:items-end"
        >
          <Field
            label="Adresse électronique"
            type="email"
            autoComplete="off"
            required
            error={formState.errors.email?.message}
            {...register("email", {
              required: "Saisissez l'adresse électronique de la personne à inviter.",
            })}
          />
          <div>
            <label htmlFor="invite-role" className="mb-1 block font-bold">
              Rôle
            </label>
            <select
              id="invite-role"
              className="h-11 w-full rounded-md border border-line-strong bg-surface px-3"
              {...register("role")}
            >
              <option value="member">Membre</option>
              <option value="admin">Administrateur</option>
            </select>
          </div>
          <Button type="submit" aria-disabled={formState.isSubmitting} className="h-11">
            <Icon icon={Send} className="h-4 w-4" />
            Envoyer l'invitation
          </Button>
        </form>
      </section>

      <section
        aria-labelledby="invitations-titre"
        className="mt-6 rounded-lg border border-line bg-surface"
      >
        <div className="flex flex-wrap items-baseline justify-between gap-2 px-5 pb-2 pt-4">
          <h2 id="invitations-titre" className="font-display text-lg font-bold">
            Invitations en attente
          </h2>
          <p className="text-sm text-muted">
            {items.length} invitation{items.length > 1 ? "s" : ""}
          </p>
        </div>
        {list.isError ? (
          <Alert kind="danger" className="m-5">
            Impossible de charger les invitations. Rechargez la page.
          </Alert>
        ) : items.length === 0 ? (
          <p className="px-5 pb-5 text-muted">
            {list.isPending ? "Chargement…" : "Aucune invitation en attente."}
          </p>
        ) : (
          <Table
            caption="Invitations en attente"
            columns={[
              { key: "email", label: "Adresse" },
              { key: "role", label: "Rôle" },
              { key: "status", label: "Statut" },
              { key: "expires", label: "Valable jusqu'au" },
              { key: "by", label: "Invitée par" },
              {
                key: "actions",
                label: <span className="sr-only">Actions</span>,
                className: "text-right",
              },
            ]}
          >
            {items.map((inv) => (
              <TableRow key={inv.id}>
                <TableCell className="[overflow-wrap:anywhere]">{inv.email}</TableCell>
                <TableCell className="whitespace-nowrap">{roleLabel(inv.role)}</TableCell>
                <TableCell>
                  {inv.status === "expired" ? (
                    <Badge tone="warning">Expirée</Badge>
                  ) : (
                    <Badge tone="info">En attente</Badge>
                  )}
                </TableCell>
                <TableCell className="whitespace-nowrap">{formatDate(inv.expiresAt)}</TableCell>
                <TableCell className="text-muted [overflow-wrap:anywhere]">
                  {inv.invitedBy ?? "Compte supprimé"}
                </TableCell>
                <TableCell>
                  <div className="flex justify-end gap-2">
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => onResend(inv.id, inv.email)}
                    >
                      <Icon icon={RotateCw} className="h-4 w-4" />
                      Renvoyer<span className="sr-only"> l'invitation de {inv.email}</span>
                    </Button>
                    <ConfirmDialog
                      title="Annuler l'invitation ?"
                      confirmLabel="Annuler l'invitation"
                      onConfirm={() => onCancel(inv.id, inv.email)}
                      trigger={
                        <Button variant="danger-secondary" size="sm">
                          <Icon icon={Trash2} className="h-4 w-4" />
                          Annuler<span className="sr-only"> l'invitation de {inv.email}</span>
                        </Button>
                      }
                    >
                      <p>
                        Le lien envoyé à {inv.email} ne fonctionnera plus. Vous pourrez l'inviter à
                        nouveau.
                      </p>
                    </ConfirmDialog>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </Table>
        )}
      </section>
    </Page>
  );
}

const roleItems: { value: RoleFilter | "all"; label: string }[] = [
  { value: "all", label: "Tous" },
  { value: "member", label: "Membres" },
  { value: "admin", label: "Administrateurs" },
];

const statusItems: { value: StatusFilter | "all"; label: string }[] = [
  { value: "all", label: "Tous" },
  { value: "active", label: "Actifs" },
  { value: "disabled", label: "Désactivés" },
];

// Liste des comptes : recherche, filtres et pagination portés par l'URL.
function AccountList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const filters = parseFilters(searchParams);
  const [draft, setDraft] = useState(filters.q);
  const list = useQuery({ ...accountListQuery(filters), placeholderData: keepPreviousData });

  // Les filtres reprennent la saisie en cours, même si son délai n'est pas écoulé.
  const navigate = (next: AccountFilters, replace = false) => {
    setSearchParams(toSearchParams(next), { replace, preventScrollReset: true });
  };

  // La recherche suit la saisie après 300 ms sans frappe, sans remplir l'historique.
  useEffect(() => {
    if (draft.trim() === filters.q.trim()) return;
    const timer = setTimeout(() => navigate({ ...filters, q: draft, page: 1 }, true), 300);
    return () => clearTimeout(timer);
  });

  const clear = () => {
    setDraft("");
    navigate({ q: "", role: "", status: "", page: 1 });
  };

  const tooShort = draft.trim() !== "" && !hasSearchableTerm(draft);
  const items = list.data?.items ?? [];
  const total = list.data?.total ?? 0;
  const pageSize = list.data?.pageSize ?? 25;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const summary = list.isPending
    ? "Chargement…"
    : total === 0
      ? "Aucun compte ne correspond à ces critères."
      : `${total} compte${total > 1 ? "s" : ""}`;

  return (
    <section aria-labelledby="comptes-titre" className="rounded-lg border border-line bg-surface">
      <div className="px-5 pt-4">
        <h2 id="comptes-titre" className="font-display text-lg font-bold">
          Comptes de l'association
        </h2>
        <search className="mt-3 grid gap-4 sm:grid-cols-[minmax(0,1fr)_11rem_11rem] sm:items-start">
          <Field
            label="Rechercher une adresse"
            type="search"
            autoComplete="off"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            hint={tooShort ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères.` : undefined}
          />
          <Select
            label="Rôle"
            value={filters.role || "all"}
            items={roleItems}
            onValueChange={(v) =>
              navigate({ ...filters, q: draft, role: v === "all" ? "" : v, page: 1 })
            }
          />
          <Select
            label="État"
            value={filters.status || "all"}
            items={statusItems}
            onValueChange={(v) =>
              navigate({ ...filters, q: draft, status: v === "all" ? "" : v, page: 1 })
            }
          />
        </search>
        <div className="mt-3 flex min-h-10 flex-wrap items-center justify-between gap-2">
          <p role="status" className="text-sm text-muted">
            {summary}
          </p>
          {hasActiveFilters(filters) || draft !== "" ? (
            <Button variant="tertiary" size="sm" onClick={clear}>
              <Icon icon={X} className="h-4 w-4" />
              Effacer les critères
            </Button>
          ) : null}
        </div>
      </div>
      {list.isError ? (
        <Alert kind="danger" className="m-5">
          Impossible de charger les comptes. Rechargez la page.
        </Alert>
      ) : items.length > 0 ? (
        <Table
          caption="Comptes de l'association"
          columns={[
            { key: "email", label: "Adresse" },
            { key: "role", label: "Rôle" },
            { key: "status", label: "État" },
            { key: "verified", label: "Adresse confirmée" },
            { key: "created", label: "Créé le" },
            { key: "login", label: "Dernière connexion" },
          ]}
        >
          {items.map((a) => (
            <TableRow key={a.id}>
              <TableCell className="[overflow-wrap:anywhere]">
                <Link
                  to={`/admin/comptes/${a.id}`}
                  className="font-bold text-brand-ink underline underline-offset-2"
                >
                  {a.email}
                </Link>
                {a.isSelf ? <span className="text-muted"> (vous)</span> : null}
              </TableCell>
              <TableCell>
                <Badge tone={a.role === "admin" ? "brand" : "neutral"}>{roleLabel(a.role)}</Badge>
              </TableCell>
              <TableCell>
                {a.status === "active" ? (
                  <Badge tone="success">Actif</Badge>
                ) : (
                  <Badge tone="muted">Désactivé</Badge>
                )}
              </TableCell>
              <TableCell>{a.emailVerified ? "Oui" : "Non"}</TableCell>
              <TableCell className="whitespace-nowrap">{formatDate(a.createdAt)}</TableCell>
              <TableCell className="whitespace-nowrap">
                {a.lastLoginAt ? formatDate(a.lastLoginAt) : "Jamais"}
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
            const params = toSearchParams({ ...filters, page }).toString();
            return params ? `?${params}` : "?";
          }}
        />
      </div>
    </section>
  );
}
