import { Dialog } from "@base-ui/react/dialog";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CalendarCheck, Copy, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useLocation, useNavigate } from "react-router";
import { membershipFeesQuery } from "~/api/membership-fees";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { invalidateSeasons, type Season, seasonQuery } from "~/api/seasons";
import { formatDate, formatDay, seasonStatusLabel, seasonStatusTone } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Icon } from "~/ui/Icon";
import NotFound from "../not-found";
import type { Route } from "./+types/season-detail";
import { SeasonDatesFields, type SeasonDatesForm } from "./SeasonDatesFields";
import { SeasonFees } from "./SeasonFees";

// Fil d'Ariane : Tableau de bord › Saisons › Saison 2026-2027.
export const handle = {
  crumb: (data: unknown) => {
    const season = data as Season | null;
    return season ? `Saison ${season.label}` : "Saison introuvable";
  },
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

// Préchargement ; les erreurs (saison inexistante, 403) sont affichées par la
// page ou le layout admin.
export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  // Les tarifs sont préchargés en parallèle ; leur erreur est affichée par la
  // section.
  queryClient.ensureQueryData(membershipFeesQuery(id)).catch(() => null);
  return queryClient.ensureQueryData(seasonQuery(id)).catch(() => null);
}

// Fiche d'une saison et actions autorisées par l'API (FR-022).
export default function SeasonDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...seasonQuery(id ?? 0), enabled: id !== null });
  const season = query.data;
  if (season) return <SeasonView key={season.id} season={season} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger cette saison. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

const unexpected = "Une erreur inattendue s'est produite.";

function SeasonView({ season }: { season: Season }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  // Message transmis après une création ou une duplication.
  const [status, setStatus] = useState(
    () => (location.state as { flash?: string } | null)?.flash ?? "",
  );
  const [error, setError] = useState("");
  const path = { params: { path: { id: season.id } } };
  const open = $api.useMutation("post", "/admin/seasons/{id}/open");
  const duplicate = $api.useMutation("post", "/admin/seasons/{id}/duplicate");
  const remove = $api.useMutation("delete", "/admin/seasons/{id}");
  const { actions } = season;

  const fail = async (e: unknown) => {
    setError(isProblem(e) && e.detail ? e.detail : unexpected);
    await invalidateSeasons(qc);
  };

  const onOpen = async () => {
    setStatus("");
    setError("");
    try {
      await open.mutateAsync(path);
      setStatus(`Saison ${season.label} ouverte.`);
      await invalidateSeasons(qc);
    } catch (e) {
      await fail(e);
    }
  };

  const onDuplicate = async () => {
    setStatus("");
    setError("");
    try {
      const created = await duplicate.mutateAsync(path);
      await invalidateSeasons(qc);
      navigate(`/admin/saisons/${created.id}`, {
        state: { flash: `Saison ${created.label} créée en brouillon.` },
      });
    } catch (e) {
      await fail(e);
    }
  };

  const onDelete = async () => {
    setStatus("");
    setError("");
    try {
      await remove.mutateAsync(path);
    } catch (e) {
      await fail(e);
      return;
    }
    qc.removeQueries({ queryKey: seasonQuery(season.id).queryKey });
    await invalidateSeasons(qc);
    navigate("/admin/saisons", { state: { flash: `Saison ${season.label} supprimée.` } });
  };

  const pageActions =
    actions.open || actions.editEndsOn || actions.duplicate || actions.delete ? (
      <>
        {actions.open ? (
          <ConfirmDialog
            tone="primary"
            title={`Ouvrir la saison ${season.label} ?`}
            confirmLabel="Ouvrir la saison"
            onConfirm={onOpen}
            trigger={
              <Button>
                <Icon icon={CalendarCheck} className="h-4 w-4" />
                Ouvrir la saison
              </Button>
            }
          >
            <p>
              Les adhésions et inscriptions pourront s'y rattacher. Une saison ouverte ne peut plus
              être supprimée et seule sa date de fin restera modifiable.
            </p>
          </ConfirmDialog>
        ) : null}
        {actions.editEndsOn ? (
          <EditDatesDialog
            season={season}
            onSaved={(message) => {
              setError("");
              setStatus(message);
            }}
          />
        ) : null}
        {actions.duplicate ? (
          <Button variant="secondary" onClick={onDuplicate} aria-disabled={duplicate.isPending}>
            <Icon icon={Copy} className="h-4 w-4" />
            Dupliquer
          </Button>
        ) : null}
        {actions.delete ? (
          <ConfirmDialog
            title={`Supprimer la saison ${season.label} ?`}
            confirmLabel="Supprimer"
            onConfirm={onDelete}
            trigger={
              <Button variant="danger-secondary">
                <Icon icon={Trash2} className="h-4 w-4" />
                Supprimer
              </Button>
            }
          >
            <p>Cette action est définitive.</p>
          </ConfirmDialog>
        ) : null}
      </>
    ) : undefined;

  return (
    <Page
      title={`Saison ${season.label}`}
      intro={`${seasonStatusLabel(season.status)}${season.isCurrent ? " · Saison courante" : ""}`}
      actions={pageActions}
    >
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
      {actions.openBlockedReason ? (
        <Alert kind="info" className="mb-6 max-w-3xl">
          {actions.openBlockedReason}
        </Alert>
      ) : null}

      <section
        aria-labelledby="saison-infos-titre"
        className="max-w-3xl rounded-lg border border-line bg-surface p-5"
      >
        <h2 id="saison-infos-titre" className="font-display text-lg font-bold">
          Informations
        </h2>
        <dl className="mt-3 grid gap-x-6 gap-y-3 sm:grid-cols-[12rem_minmax(0,1fr)]">
          <dt className="font-bold">Début</dt>
          <dd>{formatDay(season.startsOn)}</dd>
          <dt className="font-bold">Fin</dt>
          <dd>{formatDay(season.endsOn)}</dd>
          <dt className="font-bold">État</dt>
          <dd className="flex flex-wrap items-center gap-2">
            <Badge tone={seasonStatusTone[season.status]}>{seasonStatusLabel(season.status)}</Badge>
            {season.isCurrent ? <Badge tone="brand">Saison courante</Badge> : null}
          </dd>
          {season.openedAt ? (
            <>
              <dt className="font-bold">Ouverte le</dt>
              <dd>{formatDate(season.openedAt)}</dd>
            </>
          ) : null}
          <dt className="font-bold">Créée le</dt>
          <dd>{formatDate(season.createdAt)}</dd>
        </dl>
      </section>

      <SeasonFees season={season} onStatus={setStatus} onError={setError} />
    </Page>
  );
}

// Modification des dates (FR-015 à FR-017) : début seulement pour la première
// saison en brouillon, bornes de la fin fournies par l'API.
function EditDatesDialog({
  season,
  onSaved,
}: {
  season: Season;
  onSaved: (message: string) => void;
}) {
  const qc = useQueryClient();
  const [isOpen, setOpen] = useState(false);
  const update = $api.useMutation("patch", "/admin/seasons/{id}");
  const { register, handleSubmit, setError, reset, formState } = useForm<SeasonDatesForm>({
    defaultValues: { startsOn: season.startsOn, endsOn: season.endsOn },
  });

  const onSubmit = handleSubmit(async (form) => {
    try {
      await update.mutateAsync({
        params: { path: { id: season.id } },
        body: season.actions.editStartsOn
          ? { startsOn: form.startsOn, endsOn: form.endsOn }
          : { endsOn: form.endsOn },
      });
    } catch (error) {
      if (!isProblem(error)) throw error;
      applyProblemToForm(error, setError, ["startsOn", "endsOn"]);
      if (error.status === 409 || error.status === 404) await invalidateSeasons(qc);
      return;
    }
    setOpen(false);
    await invalidateSeasons(qc);
    onSaved(`Dates de la saison ${season.label} enregistrées.`);
  });

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={(o) => {
        if (o) reset({ startsOn: season.startsOn, endsOn: season.endsOn });
        setOpen(o);
      }}
    >
      <Dialog.Trigger
        render={
          <Button variant="secondary">
            <Icon icon={Pencil} className="h-4 w-4" />
            Modifier les dates
          </Button>
        }
      />
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 max-h-[calc(100vh-2rem)] w-[min(34rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <Dialog.Title className="font-display text-xl font-bold">
            Modifier les dates de la saison {season.label}
          </Dialog.Title>
          {formState.errors.root?.server ? (
            <Alert kind="danger" role="alert" className="mt-4">
              {formState.errors.root.server.message}
            </Alert>
          ) : null}
          <form onSubmit={onSubmit} noValidate className="mt-4 flex flex-col gap-5">
            <SeasonDatesFields
              register={register}
              errors={formState.errors}
              startsOn={season.actions.editStartsOn ? "editable" : "hidden"}
              endsOnMin={season.endsOnMin}
              endsOnMax={season.endsOnMax}
              endsOnHint={
                season.shiftsSeasonLabel
                  ? `Dernier jour inclus. La saison ${season.shiftsSeasonLabel} commencera le lendemain.`
                  : undefined
              }
            />
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
              <Button type="submit" aria-disabled={formState.isSubmitting}>
                Enregistrer
              </Button>
            </div>
          </form>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
