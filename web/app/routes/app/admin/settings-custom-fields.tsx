import { Collapsible } from "@base-ui/react/collapsible";
import { Menu } from "@base-ui/react/menu";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Archive,
  ArchiveRestore,
  ArrowDown,
  ArrowUp,
  ChevronDown,
  Ellipsis,
  ListPlus,
  Pencil,
  Trash2,
} from "lucide-react";
import { useRef, useState } from "react";
import {
  customFieldsQuery,
  type FieldDefinition,
  type FieldScope,
  invalidateCustomFields,
  typeLabels,
  visibilityLabels,
} from "~/api/custom-fields";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { FieldDialog } from "~/customfields/FieldDialog";
import { Alert } from "~/ui/Alert";
import { Badge, type BadgeTone } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Icon } from "~/ui/Icon";
import { SettingsPage } from "./SettingsPage";

export const handle = { crumb: "Champs personnalisés" };

export async function clientLoader() {
  await queryClient.ensureQueryData(customFieldsQuery()).catch(() => undefined);
  return null;
}

const MAX_ACTIVE = 30;

const visibilityTones: Record<FieldDefinition["visibility"], BadgeTone> = {
  admin: "muted",
  visible: "info",
  editable: "brand",
};

const scopes: { scope: FieldScope; title: string; intro: string; example: string }[] = [
  {
    scope: "member",
    title: "Champs des membres",
    intro: "Informations sur la personne, conservées d'une saison à l'autre.",
    example: "Exemple : numéro de licence fédérale.",
  },
  {
    scope: "membership",
    title: "Champs des adhésions",
    intro: "Informations propres à une saison, à renseigner pour chaque adhésion.",
    example: "Exemple : a payé sa licence fédérale.",
  },
];

// Paramètres › Champs personnalisés (011, US1 et US5) : définitions des
// champs des fiches et des adhésions, ordre, archivage et suppression.
export default function SettingsCustomFields() {
  const list = useQuery(customFieldsQuery());
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");

  return (
    <SettingsPage section="Champs personnalisés">
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
      {list.isError ? (
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger les champs personnalisés. Rechargez la page.
        </Alert>
      ) : null}
      {list.data ? (
        <div className="flex max-w-3xl flex-col gap-10">
          {scopes.map((s) => (
            <ScopeSection
              key={s.scope}
              {...s}
              fields={list.data[s.scope] ?? []}
              onStatus={(m) => {
                setError("");
                setStatus(m);
              }}
              onError={(m) => {
                setStatus("");
                setError(m);
              }}
            />
          ))}
        </div>
      ) : null}
    </SettingsPage>
  );
}

interface ScopeSectionProps {
  scope: FieldScope;
  title: string;
  intro: string;
  example: string;
  fields: FieldDefinition[];
  onStatus: (message: string) => void;
  onError: (message: string) => void;
}

function ScopeSection({
  scope,
  title,
  intro,
  example,
  fields,
  onStatus,
  onError,
}: ScopeSectionProps) {
  const qc = useQueryClient();
  const reorder = $api.useMutation("put", "/admin/custom-fields/order");
  const active = fields.filter((f) => !f.archived);
  const archived = fields.filter((f) => f.archived);
  const full = active.length >= MAX_ACTIVE;
  const headingId = `champs-${scope}`;
  const moveRefs = useRef(new Map<string, HTMLButtonElement | null>());

  const move = async (index: number, delta: -1 | 1) => {
    const ids = active.map((f) => f.id);
    const target = index + delta;
    [ids[index], ids[target]] = [ids[target], ids[index]];
    const field = active[index];
    try {
      await reorder.mutateAsync({ body: { scope, ids } });
    } catch (e) {
      onError(isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.");
      await invalidateCustomFields(qc);
      return;
    }
    await invalidateCustomFields(qc);
    onStatus(`${field.label} déplacé en position ${target + 1} sur ${ids.length}.`);
    // Le focus reste sur le bouton actionné, sauf s'il devient inactif.
    const key = `${field.id}-${delta}`;
    const other = `${field.id}-${-delta}`;
    requestAnimationFrame(() => {
      const button = moveRefs.current.get(key);
      (button?.getAttribute("aria-disabled") === "true"
        ? moveRefs.current.get(other)
        : button
      )?.focus();
    });
  };

  const addButton = (
    <FieldDialog
      scope={scope}
      onDone={onStatus}
      trigger={
        <Button variant={active.length ? "secondary" : "primary"} aria-disabled={full}>
          <Icon icon={ListPlus} />
          Ajouter un champ
        </Button>
      }
    />
  );

  return (
    <section aria-labelledby={headingId}>
      <h2 id={headingId} className="font-display text-xl font-bold">
        {title}
      </h2>
      <p className="mt-1 text-muted">{intro}</p>
      {active.length === 0 ? (
        <div className="mt-4 rounded-lg border border-dashed border-line-strong bg-surface p-6">
          <p>
            Aucun champ pour l'instant. {example} Chaque champ s'ajoute aux formulaires, avec une
            visibilité au choix : réservé au bureau, visible ou rempli par le membre.
          </p>
          <div className="mt-4">{addButton}</div>
        </div>
      ) : (
        <>
          <ol className="mt-4 flex flex-col gap-2">
            {active.map((f, i) => (
              <li
                key={f.id}
                className="flex flex-col gap-3 rounded-lg border border-line bg-surface p-4 sm:flex-row sm:items-center"
              >
                <div className="min-w-0 flex-1">
                  <p className="font-bold [overflow-wrap:anywhere]">{f.label}</p>
                  <div className="mt-1 flex flex-wrap items-center gap-2 text-sm">
                    <span className="text-muted">{typeLabels[f.type]}</span>
                    <Badge tone={visibilityTones[f.visibility]}>
                      {visibilityLabels[f.visibility]}
                    </Badge>
                    {f.required ? <Badge tone="neutral">Obligatoire</Badge> : null}
                  </div>
                  {f.missingCount ? (
                    <p className="mt-2 text-sm font-bold text-warn">
                      {missingLabel(scope, f.missingCount)}
                    </p>
                  ) : null}
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="tertiary"
                    iconOnly
                    aria-disabled={i === 0}
                    ref={(el) => {
                      moveRefs.current.set(`${f.id}--1`, el);
                    }}
                    onClick={() => (i === 0 || reorder.isPending ? undefined : move(i, -1))}
                  >
                    <Icon icon={ArrowUp} label={`Monter ${f.label}`} />
                  </Button>
                  <Button
                    variant="tertiary"
                    iconOnly
                    aria-disabled={i === active.length - 1}
                    ref={(el) => {
                      moveRefs.current.set(`${f.id}-1`, el);
                    }}
                    onClick={() =>
                      i === active.length - 1 || reorder.isPending ? undefined : move(i, 1)
                    }
                  >
                    <Icon icon={ArrowDown} label={`Descendre ${f.label}`} />
                  </Button>
                  <FieldDialog
                    scope={scope}
                    field={f}
                    onDone={onStatus}
                    trigger={
                      <Button variant="tertiary" iconOnly>
                        <Icon icon={Pencil} label={`Modifier ${f.label}`} />
                      </Button>
                    }
                  />
                  <FieldMenu field={f} onStatus={onStatus} onError={onError} />
                </div>
              </li>
            ))}
          </ol>
          <div className="mt-4 flex flex-col gap-2">
            {full ? (
              <p className="text-sm text-muted">
                30 champs actifs au plus : archivez un champ inutile pour en ajouter un.
              </p>
            ) : (
              <div>{addButton}</div>
            )}
          </div>
        </>
      )}
      {archived.length ? (
        <Collapsible.Root className="mt-6">
          <Collapsible.Trigger className="group flex items-center gap-2 font-bold">
            <Icon
              icon={ChevronDown}
              className="h-4 w-4 transition-transform group-data-[panel-open]:rotate-180"
            />
            Champs archivés ({archived.length})
          </Collapsible.Trigger>
          <Collapsible.Panel>
            <ul className="mt-3 flex flex-col gap-2">
              {archived.map((f) => (
                <li
                  key={f.id}
                  className="flex flex-col gap-3 rounded-lg border border-line bg-canvas p-4 sm:flex-row sm:items-center"
                >
                  <div className="min-w-0 flex-1">
                    <p className="font-bold [overflow-wrap:anywhere]">{f.label}</p>
                    <p className="mt-1 text-sm text-muted">
                      {typeLabels[f.type]} · {valuesLabel(f.valueCount)}
                    </p>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <RestoreButton field={f} onStatus={onStatus} onError={onError} />
                    <DeleteDialog field={f} onStatus={onStatus} onError={onError} />
                  </div>
                </li>
              ))}
            </ul>
          </Collapsible.Panel>
        </Collapsible.Root>
      ) : null}
    </section>
  );
}

function missingLabel(scope: FieldScope, n: number) {
  if (scope === "member") return n > 1 ? `${n} fiches sans valeur` : "1 fiche sans valeur";
  return n > 1
    ? `${n} adhésions de la saison en cours sans valeur`
    : "1 adhésion de la saison en cours sans valeur";
}

function valuesLabel(n: number) {
  if (n === 0) return "aucune valeur";
  return n > 1 ? `${n} valeurs` : "1 valeur";
}

interface ActionProps {
  field: FieldDefinition;
  onStatus: (message: string) => void;
  onError: (message: string) => void;
}

function problemMessage(e: unknown) {
  return isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.";
}

function useArchive({ field, onStatus, onError }: ActionProps) {
  const qc = useQueryClient();
  const archive = $api.useMutation("post", "/admin/custom-fields/{id}/archive");
  return async () => {
    try {
      await archive.mutateAsync({ params: { path: { id: field.id } } });
    } catch (e) {
      onError(problemMessage(e));
      await invalidateCustomFields(qc);
      return;
    }
    await invalidateCustomFields(qc);
    onStatus(`Champ ${field.label} archivé. Ses valeurs sont conservées.`);
  };
}

const itemClass =
  "flex cursor-pointer items-center gap-2 px-3 py-2 outline-none data-[highlighted]:bg-brand/10";

// Menu « Plus » d'un champ actif : archiver, supprimer.
function FieldMenu(props: ActionProps) {
  const { field } = props;
  const archive = useArchive(props);
  const [confirmDelete, setConfirmDelete] = useState(false);
  return (
    <>
      <Menu.Root>
        <Menu.Trigger
          className="grid h-11 w-11 place-items-center rounded-md text-ink hover:bg-ink/5"
          aria-label={`Plus d'actions pour ${field.label}`}
        >
          <Icon icon={Ellipsis} />
        </Menu.Trigger>
        <Menu.Portal>
          <Menu.Positioner align="end" sideOffset={4} className="z-50 outline-none">
            <Menu.Popup className="w-56 rounded-lg border border-line bg-surface py-1 text-ink shadow-lg outline-none">
              <Menu.Item onClick={archive} className={itemClass}>
                <Icon icon={Archive} className="h-4 w-4" />
                Archiver
              </Menu.Item>
              <Menu.Item
                onClick={() => setConfirmDelete(true)}
                className={`${itemClass} text-danger`}
              >
                <Icon icon={Trash2} className="h-4 w-4" />
                Supprimer
              </Menu.Item>
            </Menu.Popup>
          </Menu.Positioner>
        </Menu.Portal>
      </Menu.Root>
      <DeleteDialog
        {...props}
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        onArchive={archive}
      />
    </>
  );
}

function RestoreButton({ field, onStatus, onError }: ActionProps) {
  const qc = useQueryClient();
  const restore = $api.useMutation("post", "/admin/custom-fields/{id}/restore");
  const onClick = async () => {
    if (restore.isPending) return;
    try {
      await restore.mutateAsync({ params: { path: { id: field.id } } });
    } catch (e) {
      onError(problemMessage(e));
      await invalidateCustomFields(qc);
      return;
    }
    await invalidateCustomFields(qc);
    onStatus(`Champ ${field.label} rétabli.`);
  };
  return (
    <Button variant="secondary" size="sm" onClick={onClick} aria-disabled={restore.isPending}>
      <Icon icon={ArchiveRestore} className="h-4 w-4" />
      Rétablir<span className="sr-only"> {field.label}</span>
    </Button>
  );
}

// Suppression définitive (US5) : le dialogue annonce les valeurs perdues et
// propose l'archivage pour un champ actif.
function DeleteDialog({
  field,
  onStatus,
  onError,
  open,
  onOpenChange,
  onArchive,
}: ActionProps & {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onArchive?: () => Promise<void>;
}) {
  const qc = useQueryClient();
  const remove = $api.useMutation("delete", "/admin/custom-fields/{id}");
  const onConfirm = async () => {
    try {
      await remove.mutateAsync({ params: { path: { id: field.id } } });
    } catch (e) {
      onError(problemMessage(e));
      await invalidateCustomFields(qc);
      return;
    }
    await invalidateCustomFields(qc);
    onStatus(`Champ ${field.label} supprimé.`);
  };
  const controlled = open !== undefined;
  return (
    <ConfirmDialog
      trigger={
        controlled ? undefined : (
          <Button variant="danger-secondary" size="sm">
            <Icon icon={Trash2} className="h-4 w-4" />
            Supprimer<span className="sr-only"> {field.label}</span>
          </Button>
        )
      }
      open={open}
      onOpenChange={onOpenChange}
      title={`Supprimer le champ « ${field.label} » ?`}
      confirmLabel="Supprimer"
      onConfirm={onConfirm}
      extraActions={
        onArchive && field.valueCount > 0 ? (
          <Button
            variant="secondary"
            onClick={() => {
              onOpenChange?.(false);
              void onArchive();
            }}
          >
            Archiver à la place
          </Button>
        ) : undefined
      }
    >
      {field.valueCount > 0 ? (
        <p>
          {field.valueCount > 1
            ? `Les ${field.valueCount} valeurs enregistrées seront définitivement perdues.`
            : "La valeur enregistrée sera définitivement perdue."}{" "}
          {onArchive ? "Pour les conserver, archivez plutôt le champ." : null}
        </p>
      ) : (
        <p>Le champ disparaîtra des formulaires. Aucune valeur n'est enregistrée.</p>
      )}
    </ConfirmDialog>
  );
}
