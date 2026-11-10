import { Menu } from "@base-ui/react/menu";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Ban, Download, Ellipsis, Pencil, RotateCcw, Send, Trash2 } from "lucide-react";
import { useState } from "react";
import { Link, useLocation, useNavigate } from "react-router";
import { client } from "~/api/client";
import { type AdminEvent, eventQuery, invalidateEvents } from "~/api/events";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { type AdminRegistration, eventRegistrationsQuery } from "~/api/registrations";
import { formatEventMoment, formatEventRange } from "~/events/date";
import { EventBadges, priceLabel } from "~/events/EventBadges";
import { RegisterMemberDialog } from "~/events/RegisterMemberDialog";
import { RegistrationBadges } from "~/events/RegistrationBadges";
import { formatDate, formatMoney } from "~/lib/format";
import { downloadBlob } from "~/members/import/download";
import { errorMessage, RecordPaymentDialog } from "~/payments/AdminPayments";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button, buttonClasses } from "~/ui/Button";
import { Checkbox } from "~/ui/Checkbox";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { TextAreaField } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { Table, TableCell, TableRow } from "~/ui/Table";
import NotFound from "../not-found";
import type { Route } from "./+types/event-detail";

export const handle = {
  crumb: (data: unknown) => {
    const e = data as AdminEvent | null;
    return e ? e.title : "Événement introuvable";
  },
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(eventQuery(id)).catch(() => null);
}

// Détail d'un événement : informations, chiffres, inscrits et actions
// (US1, US5, US6).
export default function AdminEventDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...eventQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <EventView event={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert">
        Impossible de charger cet événement. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 whitespace-pre-line font-bold [overflow-wrap:anywhere]">{children}</dd>
    </div>
  );
}

const itemClass =
  "flex cursor-pointer items-center gap-2 px-3 py-2 outline-none data-[highlighted]:bg-brand/10";

function EventView({ event: e }: { event: AdminEvent }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  const [status, setStatus] = useState(
    () => (location.state as { flash?: string } | null)?.flash ?? "",
  );
  const [error, setError] = useState("");
  const publish = $api.useMutation("post", "/admin/events/{id}/publish");
  const restore = $api.useMutation("post", "/admin/events/{id}/restore");
  const remove = $api.useMutation("delete", "/admin/events/{id}");
  const [cancelOpen, setCancelOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const path = { params: { path: { id: e.id } } };

  const done = async (message: string) => {
    setError("");
    setStatus(message);
    await invalidateEvents(qc);
  };
  const failed = async (err: unknown) => {
    setStatus("");
    setError(errorMessage(err));
    await invalidateEvents(qc);
  };
  const run = async (fn: () => Promise<unknown>, message: string) => {
    try {
      await fn();
    } catch (err) {
      await failed(err);
      return;
    }
    await done(message);
  };

  const onDelete = async () => {
    try {
      await remove.mutateAsync(path);
    } catch (err) {
      await failed(err);
      return;
    }
    await invalidateEvents(qc);
    navigate("/admin/evenements", { state: { flash: `Événement « ${e.title} » supprimé.` } });
  };

  const hasMenu = e.actions.cancel || e.actions.restore || e.actions.delete;
  const actions = (
    <div className="flex flex-wrap items-center gap-2">
      {e.actions.publish ? (
        <Button
          onClick={() =>
            run(() => publish.mutateAsync(path), "Événement publié : les membres le voient.")
          }
        >
          <Icon icon={Send} className="h-4 w-4" />
          Publier
        </Button>
      ) : null}
      {e.actions.update ? (
        <Link
          to={`/admin/evenements/${e.id}/modifier`}
          className={buttonClasses({ variant: "secondary" })}
        >
          <Icon icon={Pencil} className="h-4 w-4" />
          Modifier
        </Link>
      ) : null}
      {hasMenu ? (
        <Menu.Root>
          <Menu.Trigger
            className="grid h-11 w-11 place-items-center rounded-md text-ink hover:bg-ink/5"
            aria-label="Plus d'actions sur l'événement"
          >
            <Icon icon={Ellipsis} />
          </Menu.Trigger>
          <Menu.Portal>
            <Menu.Positioner align="end" sideOffset={4} className="z-50 outline-none">
              <Menu.Popup className="w-60 rounded-lg border border-line bg-surface py-1 text-ink shadow-lg outline-none">
                {e.actions.cancel ? (
                  <Menu.Item
                    onClick={() => setCancelOpen(true)}
                    className={`${itemClass} text-danger`}
                  >
                    <Icon icon={Ban} className="h-4 w-4" />
                    Annuler l'événement
                  </Menu.Item>
                ) : null}
                {e.actions.restore ? (
                  <Menu.Item
                    onClick={() =>
                      run(
                        () => restore.mutateAsync(path),
                        "Événement rétabli : les inscrits sont prévenus.",
                      )
                    }
                    className={itemClass}
                  >
                    <Icon icon={RotateCcw} className="h-4 w-4" />
                    Rétablir l'événement
                  </Menu.Item>
                ) : null}
                {e.actions.delete ? (
                  <Menu.Item
                    onClick={() => setDeleteOpen(true)}
                    className={`${itemClass} text-danger`}
                  >
                    <Icon icon={Trash2} className="h-4 w-4" />
                    Supprimer
                  </Menu.Item>
                ) : null}
              </Menu.Popup>
            </Menu.Positioner>
          </Menu.Portal>
        </Menu.Root>
      ) : null}
    </div>
  );

  return (
    <Page title={e.title} subtitle={formatEventRange(e.startsAt, e.endsAt)} actions={actions}>
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}
      {error ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {error}
        </Alert>
      ) : null}
      {e.phase === "cancelled" ? (
        <Alert kind="danger" className="mb-6 max-w-3xl" title="Événement annulé">
          {e.cancelMessage ?? "Aucun message n'a été envoyé avec l'annulation."}
        </Alert>
      ) : null}
      {e.phase === "draft" ? (
        <Alert kind="info" className="mb-6 max-w-3xl">
          Brouillon : les membres ne voient pas cet événement tant qu'il n'est pas publié.
        </Alert>
      ) : null}
      <div className="flex flex-col gap-6">
        <section
          aria-labelledby="evenement-infos"
          className="max-w-3xl rounded-lg border border-line bg-surface p-5"
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 id="evenement-infos" className="font-display text-lg font-bold">
              Informations
            </h2>
            <EventBadges event={e} />
          </div>
          <dl className="mt-3 grid gap-3 sm:grid-cols-2">
            <Info label="Date">{formatEventRange(e.startsAt, e.endsAt)}</Info>
            <Info label="Lieu">{e.location}</Info>
            <Info label="Saison">{e.season.label}</Info>
            <Info label="Qui peut s'inscrire">
              {e.openness === "all_members" ? "Toutes les fiches" : "Adhérents de la saison"}
            </Info>
            <Info label="Capacité">{e.capacity ? `${e.capacity} places` : "Illimitée"}</Info>
            <Info label="Prix">{priceLabel(e.priceCents)}</Info>
            {e.window ? (
              <Info label="Inscriptions">
                du {formatEventMoment(e.window.opensAt)} au {formatEventMoment(e.window.closesAt)}
              </Info>
            ) : null}
            {e.description ? (
              <div className="min-w-0 sm:col-span-2">
                <dt className="text-sm text-muted">Description</dt>
                <dd className="mt-0.5 whitespace-pre-line [overflow-wrap:anywhere]">
                  {e.description}
                </dd>
              </div>
            ) : null}
          </dl>
        </section>
        {e.status !== "draft" ? <Figures event={e} /> : null}
        {e.status !== "draft" ? <Registrations event={e} onDone={done} onError={failed} /> : null}
      </div>
      <CancelEventDialog
        event={e}
        open={cancelOpen}
        onOpenChange={setCancelOpen}
        onDone={done}
        onError={failed}
      />
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={`Supprimer « ${e.title} » ?`}
        confirmLabel="Supprimer définitivement"
        onConfirm={onDelete}
      >
        <p>L'événement sera supprimé définitivement. Cette action est irréversible.</p>
      </ConfirmDialog>
    </Page>
  );
}

// Chiffres des inscriptions confirmées (FR-026).
function Figures({ event: e }: { event: AdminEvent }) {
  const t = e.totals;
  return (
    <section
      aria-labelledby="evenement-chiffres"
      className="max-w-3xl rounded-lg border border-line bg-surface p-5"
    >
      <h2 id="evenement-chiffres" className="font-display text-lg font-bold">
        Chiffres
      </h2>
      <dl className="mt-3 grid gap-3 sm:grid-cols-3">
        <Info label="Inscrits">
          {e.confirmedCount}
          {e.capacity ? ` / ${e.capacity}` : ""}
        </Info>
        {e.priceCents > 0 || t.dueCents > 0 ? (
          <>
            <Info label="Total dû">{formatMoney(t.dueCents)}</Info>
            <Info label="Encaissé">
              {formatMoney(t.collectedCents)}
              {t.collectedTestCents > 0 ? (
                <span className="block text-sm font-normal text-muted">
                  dont {formatMoney(t.collectedTestCents)} en mode test (non compté)
                </span>
              ) : null}
            </Info>
          </>
        ) : null}
      </dl>
    </section>
  );
}

interface ActionProps {
  onDone: (message: string) => Promise<void>;
  onError: (e: unknown) => Promise<void>;
}

// Inscrits d'un événement (US5).
function Registrations({ event: e, onDone, onError }: { event: AdminEvent } & ActionProps) {
  const [all, setAll] = useState(false);
  const list = useQuery(eventRegistrationsQuery(e.id, all ? "all" : "confirmed"));
  const items = list.data?.items ?? [];
  const [exporting, setExporting] = useState(false);

  const onExport = async () => {
    setExporting(true);
    try {
      const { data, error } = await client.GET("/admin/events/{id}/registrations.csv", {
        params: { path: { id: e.id } },
        parseAs: "blob",
      });
      if (error || !data) throw error ?? new Error("export");
      downloadBlob(data as Blob, `inscrits-${e.id}.csv`);
    } catch (err) {
      await onError(err);
    } finally {
      setExporting(false);
    }
  };

  return (
    <section
      aria-labelledby="evenement-inscrits"
      className="rounded-lg border border-line bg-surface"
    >
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 pt-5">
        <h2 id="evenement-inscrits" className="font-display text-lg font-bold">
          Inscrits
        </h2>
        <div className="flex flex-wrap items-center gap-2">
          {e.actions.register ? (
            <RegisterMemberDialog
              eventId={e.id}
              priceCents={e.priceCents}
              onDone={onDone}
              onError={onError}
            />
          ) : null}
          <Button
            variant="secondary"
            size="sm"
            onClick={onExport}
            aria-disabled={exporting || items.length === 0}
          >
            <Icon icon={Download} className="h-4 w-4" />
            Exporter (CSV)
          </Button>
        </div>
      </div>
      <div className="px-5 pt-3">
        <Checkbox checked={all} onCheckedChange={setAll}>
          Afficher les inscriptions annulées
        </Checkbox>
      </div>
      {list.isError ? (
        <Alert kind="danger" role="alert" className="m-5">
          Impossible de charger les inscrits. Rechargez la page.
        </Alert>
      ) : items.length === 0 ? (
        <p className="px-5 py-5 text-muted">Aucune inscription pour le moment.</p>
      ) : (
        <div className="mt-3 pb-2">
          <Table
            caption={`Inscrits à ${e.title}`}
            columns={[
              { key: "member", label: "Membre" },
              { key: "member-state", label: "Adhérent" },
              { key: "date", label: "Inscription" },
              { key: "payment", label: "État" },
              { key: "actions", label: <span className="sr-only">Actions</span> },
            ]}
          >
            {items.map((r) => (
              <RegistrationRow key={r.id} reg={r} onDone={onDone} onError={onError} />
            ))}
          </Table>
        </div>
      )}
    </section>
  );
}

function RegistrationRow({ reg: r, onDone, onError }: { reg: AdminRegistration } & ActionProps) {
  const record = $api.useMutation("post", "/admin/registrations/{id}/payment");
  const remove = $api.useMutation("post", "/admin/registrations/{id}/cancel");
  const name = `${r.member.lastName} ${r.member.firstName}`;
  return (
    <TableRow>
      <th scope="row" className="px-5 py-3 text-left font-normal">
        <Link
          to={`/admin/inscriptions/${r.id}`}
          className="font-bold text-brand-ink underline underline-offset-2 [overflow-wrap:anywhere]"
        >
          {name}
        </Link>
        {r.contactEmail ? null : (
          <span className="block text-sm text-muted">Non joignable par e-mail</span>
        )}
      </th>
      <TableCell>
        <span className="inline-flex flex-wrap items-center gap-2">
          {r.isSeasonMember ? "Oui" : "Non"}
          {r.state === "confirmed" && !r.eligible ? (
            <Badge tone="warning">Non éligible</Badge>
          ) : null}
        </span>
      </TableCell>
      <TableCell className="whitespace-nowrap">
        {formatDate(r.registeredAt)}
        <span className="block text-sm text-muted">
          {r.registeredBy ? r.registeredBy.email : "—"}
        </span>
      </TableCell>
      <TableCell>
        <RegistrationBadges state={r.state} payment={r.payment} />
      </TableCell>
      <TableCell>
        <span className="flex flex-wrap justify-end gap-2">
          {r.actions.recordPayment ? (
            <RecordPaymentDialog
              amountCents={r.amountCents}
              reference={r.payment?.reference}
              announcedMethod={r.payment?.announcedMethod}
              record={(body) => record.mutateAsync({ params: { path: { id: r.id } }, body })}
              onDone={onDone}
              onError={onError}
              trigger={
                <Button variant="secondary" size="sm">
                  Constater le paiement
                </Button>
              }
            />
          ) : null}
          {r.actions.remove ? (
            <ConfirmDialog
              title={`Retirer l'inscription de ${r.member.firstName} ${r.member.lastName} ?`}
              confirmLabel="Retirer l'inscription"
              onConfirm={async () => {
                try {
                  await remove.mutateAsync({ params: { path: { id: r.id } } });
                } catch (err) {
                  await onError(err);
                  return;
                }
                await onDone("Inscription retirée.");
              }}
              trigger={
                <Button variant="danger-secondary" size="sm">
                  Retirer
                </Button>
              }
            >
              <p>La place sera libérée et la personne prévenue par e-mail si elle est joignable.</p>
              {r.payment?.status === "paid" ? (
                <p className="mt-2 font-bold">
                  Cette inscription est payée : le remboursement se fait hors de Frigg, ou dans
                  l'espace HelloAsso pour un paiement par carte.
                </p>
              ) : null}
            </ConfirmDialog>
          ) : null}
        </span>
      </TableCell>
    </TableRow>
  );
}

interface CancelProps extends ActionProps {
  event: AdminEvent;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// Annulation avec message aux inscrits (US6).
function CancelEventDialog({ event: e, open, onOpenChange, onDone, onError }: CancelProps) {
  const cancel = $api.useMutation("post", "/admin/events/{id}/cancel");
  const [message, setMessage] = useState("");
  const paid = e.totals.paidCount;
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(o) => {
        if (o) setMessage("");
        onOpenChange(o);
      }}
      title={`Annuler « ${e.title} » ?`}
      confirmLabel="Annuler l'événement"
      onConfirm={async () => {
        try {
          await cancel.mutateAsync({
            params: { path: { id: e.id } },
            body: { message: message.trim() ? message : null },
          });
        } catch (err) {
          await onError(err);
          return;
        }
        await onDone("Événement annulé : les inscrits sont prévenus.");
      }}
      content={
        <TextAreaField
          label="Message aux inscrits (facultatif)"
          maxLength={1000}
          rows={4}
          value={message}
          onChange={(ev) => setMessage(ev.target.value)}
          className="mt-4"
        />
      }
    >
      <p>
        {e.confirmedCount === 0
          ? "Aucun inscrit à prévenir."
          : `${e.confirmedCount} inscrit(s) seront prévenus par e-mail. Les inscriptions sont conservées.`}
      </p>
      {paid > 0 ? (
        <p className="mt-2 font-bold">
          {paid} inscription(s) payée(s) (
          {formatMoney(e.totals.collectedCents + e.totals.collectedTestCents)}) : le remboursement
          se fait hors de Frigg, ou dans l'espace HelloAsso pour un paiement par carte.
        </p>
      ) : null}
    </ConfirmDialog>
  );
}
