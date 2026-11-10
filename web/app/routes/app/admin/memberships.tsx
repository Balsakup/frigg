import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { CreditCard, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router";
import { customFieldsQuery } from "~/api/custom-fields";
import {
  type MembershipStats,
  membershipDashboardQuery,
  membershipPageQuery,
  membershipStatsQuery,
} from "~/api/memberships";
import { queryClient } from "~/api/query-client";
import { seasonListQuery } from "~/api/seasons";
import { CustomFieldFilters, filterableFields } from "~/customfields/CustomFieldFilters";
import { hasSearchableTerm, SEARCH_MIN_LENGTH } from "~/lib/account-filters";
import { formatAmount, formatMoney, membershipKindLabel } from "~/lib/format";
import {
  hasActiveMembershipFilters,
  type MembershipFilters,
  parseMembershipFilters,
  toMembershipApiQuery,
  toMembershipSearchParams,
} from "~/lib/membership-filters";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button, buttonClasses } from "~/ui/Button";
import { Field } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { Pagination } from "~/ui/Pagination";
import { Select } from "~/ui/Select";
import { Table, TableCell, TableRow } from "~/ui/Table";
import type { Route } from "./+types/memberships";

export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  const filters = parseMembershipFilters(new URL(request.url).searchParams);
  await Promise.all([
    queryClient.ensureQueryData(seasonListQuery()).catch(() => undefined),
    queryClient
      .ensureQueryData(membershipPageQuery(toMembershipApiQuery(filters)))
      .catch(() => undefined),
    queryClient.ensureQueryData(membershipDashboardQuery()).catch(() => undefined),
    queryClient.ensureQueryData(customFieldsQuery()).catch(() => undefined),
  ]);
  return null;
}

const statusItems = [
  { value: "", label: "Tous" },
  { value: "pending", label: "En attente" },
  { value: "validated", label: "Validées" },
  { value: "refused", label: "Refusées" },
  { value: "cancelled", label: "Annulées" },
];

const methodItems = [
  { value: "", label: "Tous" },
  { value: "card", label: "Carte bancaire" },
  { value: "cash", label: "Espèces" },
  { value: "cheque", label: "Chèque" },
  { value: "transfer", label: "Virement" },
];

const alertItems = [
  { value: "", label: "Toutes" },
  { value: "open", label: "Avec signalement" },
];

const paymentItems = [
  { value: "", label: "Tous" },
  { value: "due", label: "À payer" },
  { value: "announced", label: "Paiement annoncé" },
  { value: "paid", label: "Payée" },
  { value: "free", label: "Gratuite" },
];

// Rubrique « Adhésions » de l'administration : chiffres de la saison, filtres
// et liste (006, US2, US8).
export default function AdminMemberships() {
  const location = useLocation();
  const [status] = useState(() => (location.state as { flash?: string } | null)?.flash ?? "");
  const [searchParams, setSearchParams] = useSearchParams();
  const filters = parseMembershipFilters(searchParams);
  const definitions = useQuery(customFieldsQuery());
  const filterable = filterableFields(definitions.data?.membership);
  const [draft, setDraft] = useState(filters.q);
  const seasons = useQuery(seasonListQuery());
  const dashboard = useQuery(membershipDashboardQuery());
  const list = useQuery({
    ...membershipPageQuery(toMembershipApiQuery(filters)),
    placeholderData: keepPreviousData,
  });
  const seasonId = list.data?.seasonId ?? null;
  const stats = useQuery({ ...membershipStatsQuery(seasonId ?? 0), enabled: seasonId !== null });

  const navigate = (next: MembershipFilters, replace = false) => {
    setSearchParams(toMembershipSearchParams(next), { replace, preventScrollReset: true });
  };

  useEffect(() => {
    if (draft.trim() === filters.q.trim()) return;
    const timer = setTimeout(() => navigate({ ...filters, q: draft, page: 1 }, true), 300);
    return () => clearTimeout(timer);
  });

  const clear = () => {
    setDraft("");
    navigate({
      season: filters.season,
      status: "",
      payment: "",
      paidMethod: "",
      alerts: "",
      fields: [],
      q: "",
      page: 1,
    });
  };

  const tooShort = draft.trim() !== "" && !hasSearchableTerm(draft);
  const items = list.data?.items ?? [];
  const total = list.data?.total ?? 0;
  const pageSize = list.data?.pageSize ?? 25;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const pending = dashboard.data?.pendingCount ?? 0;
  const showSeason = seasonId === null;
  const seasonItems = [
    ...(seasons.data?.items ?? []).map((s) => ({ value: String(s.id), label: s.label })),
    { value: "all", label: "Toutes" },
  ];
  const summary = list.isPending
    ? "Chargement…"
    : total === 0
      ? "Aucune adhésion ne correspond à ces critères."
      : `${total} adhésion${total > 1 ? "s" : ""}`;

  return (
    <Page
      title="Adhésions"
      intro="Les adhésions des membres, saison par saison."
      actions={
        <Link to="/admin/adhesions/nouvelle" className={buttonClasses({})}>
          <Icon icon={CreditCard} className="h-4 w-4" />
          Créer une adhésion
        </Link>
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
      {pending > 0 ? (
        <Alert
          kind="info"
          className="mb-6"
          actions={
            <Link
              to="?season=all&status=pending"
              className={buttonClasses({ variant: "secondary", size: "sm" })}
            >
              Afficher les demandes en attente
            </Link>
          }
        >
          {pending} demande{pending > 1 ? "s" : ""} en attente.
        </Alert>
      ) : null}

      {stats.data && seasonId !== null ? <Stats stats={stats.data} /> : null}

      <section
        aria-labelledby="adhesions-titre"
        className="rounded-lg border border-line bg-surface"
      >
        <div className="px-5 pt-4">
          <h2 id="adhesions-titre" className="font-display text-lg font-bold">
            Adhésions
          </h2>
          <search className="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-[minmax(0,1fr)_10rem_11rem_11rem] lg:items-start">
            <Field
              label="Nom ou référence"
              type="search"
              autoComplete="off"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              hint={
                tooShort
                  ? `Saisissez au moins ${SEARCH_MIN_LENGTH} caractères.`
                  : "Membre, référence de paiement ou n° de commande HelloAsso."
              }
            />
            <Select
              label="Saison"
              value={filters.season || (seasonId !== null ? String(seasonId) : "all")}
              items={seasonItems}
              onValueChange={(v) => navigate({ ...filters, q: draft, season: v, page: 1 })}
            />
            <Select
              label="État"
              value={filters.status}
              items={statusItems}
              onValueChange={(v) => navigate({ ...filters, q: draft, status: v, page: 1 })}
            />
            <Select
              label="Paiement"
              value={filters.payment}
              items={paymentItems}
              onValueChange={(v) => navigate({ ...filters, q: draft, payment: v, page: 1 })}
            />
            <Select
              label="Payée par"
              value={filters.paidMethod ?? ""}
              items={methodItems}
              onValueChange={(v) => navigate({ ...filters, q: draft, paidMethod: v, page: 1 })}
            />
            <Select
              label="Signalements"
              value={filters.alerts ?? ""}
              items={alertItems}
              onValueChange={(v) => navigate({ ...filters, q: draft, alerts: v, page: 1 })}
            />
            <CustomFieldFilters
              fields={filterable}
              value={filters.fields ?? []}
              onChange={(fields) => navigate({ ...filters, q: draft, fields, page: 1 })}
            />
          </search>
          <div className="mt-3 flex min-h-10 flex-wrap items-center justify-between gap-2">
            <p role="status" className="text-sm text-muted">
              {summary}
            </p>
            {hasActiveMembershipFilters(filters) || draft !== "" ? (
              <Button variant="tertiary" size="sm" onClick={clear}>
                <Icon icon={X} className="h-4 w-4" />
                Effacer la recherche
              </Button>
            ) : null}
          </div>
        </div>
        {list.isError ? (
          <Alert kind="danger" className="m-5">
            Impossible de charger les adhésions. Rechargez la page.
          </Alert>
        ) : items.length > 0 ? (
          <Table
            caption="Adhésions"
            columns={[
              { key: "member", label: "Membre" },
              ...(showSeason ? [{ key: "season", label: "Saison" }] : []),
              { key: "fee", label: "Tarif" },
              { key: "kind", label: "Type" },
              { key: "amount", label: "Montant" },
              { key: "status", label: "État" },
              { key: "reference", label: "Référence" },
            ]}
          >
            {items.map((m) => (
              <TableRow key={m.id}>
                <th
                  scope="row"
                  className="px-5 py-3 text-left font-normal [overflow-wrap:anywhere]"
                >
                  <Link
                    to={`/admin/adhesions/${m.id}`}
                    className="font-bold text-brand-ink underline underline-offset-2"
                  >
                    {m.member.lastName} {m.member.firstName}
                  </Link>
                </th>
                {showSeason ? (
                  <TableCell className="whitespace-nowrap">{m.season.label}</TableCell>
                ) : null}
                <TableCell>{m.fee.label}</TableCell>
                <TableCell>
                  {m.kind ? (
                    membershipKindLabel(m.kind)
                  ) : m.declaredKind ? (
                    <>
                      {membershipKindLabel(m.declaredKind)}{" "}
                      <span className="text-muted">(déclaré)</span>
                    </>
                  ) : (
                    "—"
                  )}
                </TableCell>
                <TableCell className="whitespace-nowrap">
                  {m.amountCents != null ? formatAmount(m.amountCents) : "—"}
                </TableCell>
                <TableCell>
                  <MembershipBadges
                    status={m.status}
                    amountCents={m.amountCents}
                    payment={m.payment}
                  />
                  {m.openAlerts > 0 ? (
                    <span className="mt-1 block">
                      <Badge tone="danger">
                        {m.openAlerts} signalement{m.openAlerts > 1 ? "s" : ""}
                      </Badge>
                    </span>
                  ) : null}
                </TableCell>
                <TableCell className="whitespace-nowrap">{m.payment?.reference ?? "—"}</TableCell>
              </TableRow>
            ))}
          </Table>
        ) : null}
        <div className="px-5 py-4 empty:hidden">
          <Pagination
            page={Math.min(filters.page, pageCount)}
            pageCount={pageCount}
            hrefFor={(page) => {
              const params = toMembershipSearchParams({ ...filters, page }).toString();
              return params ? `?${params}` : "?";
            }}
          />
        </div>
      </section>
    </Page>
  );
}

// Chiffres de la saison (FR-018) : seules les adhésions validées comptent.
function Stats({ stats }: { stats: MembershipStats }) {
  const tile = (label: string, value: string, detail?: string) => (
    <div className="rounded-lg border border-line bg-surface p-4">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="font-display text-2xl font-bold">{value}</dd>
      {detail ? <dd className="text-sm text-muted">{detail}</dd> : null}
    </div>
  );
  const d = stats.dueByMethod;
  return (
    <section aria-labelledby="chiffres" className="mb-6">
      <h2 id="chiffres" className="font-display text-lg font-bold">
        Chiffres de la saison
      </h2>
      <dl className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        {tile(
          "Adhésions validées",
          String(stats.validated),
          `${stats.first} première${stats.first > 1 ? "s" : ""} adhésion${stats.first > 1 ? "s" : ""}, ${stats.renewal} renouvellement${stats.renewal > 1 ? "s" : ""}`,
        )}
        {tile("Demandes en attente", String(stats.pending))}
        {tile(
          "Total à payer",
          formatMoney(stats.totalCents),
          stats.discountCents || stats.surchargeCents
            ? [
                `théorique ${formatMoney(stats.theoreticalCents)}`,
                stats.discountCents ? `remises ${formatMoney(stats.discountCents)}` : "",
                stats.surchargeCents ? `majorations ${formatMoney(stats.surchargeCents)}` : "",
              ]
                .filter(Boolean)
                .join(" · ")
            : undefined,
        )}
        {tile(
          "Encaissé",
          formatMoney(stats.paidCents),
          [
            stats.collectedByMethod.card
              ? `carte ${formatMoney(stats.collectedByMethod.card)}`
              : "",
            stats.collectedByMethod.cash
              ? `espèces ${formatMoney(stats.collectedByMethod.cash)}`
              : "",
            stats.collectedByMethod.cheque
              ? `chèque ${formatMoney(stats.collectedByMethod.cheque)}`
              : "",
            stats.collectedByMethod.transfer
              ? `virement ${formatMoney(stats.collectedByMethod.transfer)}`
              : "",
            stats.donationCents ? `dont dons ${formatMoney(stats.donationCents)}` : "",
            stats.collectedTest
              ? `dont paiements de test ${formatMoney(stats.collectedTest)} (hors total)`
              : "",
          ]
            .filter(Boolean)
            .join(" · ") || undefined,
        )}
        {tile(
          "Restant à percevoir",
          formatMoney(stats.dueCents),
          stats.dueCents > 0
            ? [
                d.none ? `non choisi ${formatMoney(d.none)}` : "",
                d.cash ? `espèces ${formatMoney(d.cash)}` : "",
                d.cheque ? `chèque ${formatMoney(d.cheque)}` : "",
                d.transfer ? `virement ${formatMoney(d.transfer)}` : "",
              ]
                .filter(Boolean)
                .join(" · ")
            : undefined,
        )}
      </dl>
      {(stats.byFee ?? []).length > 0 ? (
        <div className="mt-3 max-w-2xl rounded-lg border border-line bg-surface">
          <Table
            caption="Répartition par tarif"
            captionVisible
            columns={[
              { key: "fee", label: "Tarif" },
              { key: "count", label: "Adhésions" },
              { key: "amount", label: "Montant" },
            ]}
          >
            {(stats.byFee ?? []).map((f) => (
              <TableRow key={f.feeId}>
                <th scope="row" className="px-5 py-3 text-left font-normal">
                  {f.label}
                </th>
                <TableCell>{f.count}</TableCell>
                <TableCell className="whitespace-nowrap">{formatMoney(f.amountCents)}</TableCell>
              </TableRow>
            ))}
          </Table>
        </div>
      ) : null}
    </section>
  );
}
