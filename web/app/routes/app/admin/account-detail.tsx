import { Radio } from "@base-ui/react/radio";
import { RadioGroup } from "@base-ui/react/radio-group";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Ban, KeyRound, RotateCcw, Trash2, UserCog } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { accountListKey, accountQuery, invitationListKey } from "~/api/accounts";
import { accountMembersQuery, invalidateMembers } from "~/api/members";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import type { components } from "~/api/schema";
import { formatDate, roleLabel } from "~/lib/format";
import { fullName } from "~/members/form";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Icon } from "~/ui/Icon";
import { Select } from "~/ui/Select";
import NotFound from "../not-found";
import type { Route } from "./+types/account-detail";

type AdminAccount = components["schemas"]["AdminAccount"];
type Role = AdminAccount["role"];

// Fil d'Ariane : Tableau de bord › Comptes › adresse du compte.
export const handle = {
  crumb: (data: unknown) => (data as AdminAccount | null)?.email ?? "Compte introuvable",
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

// Préchargement ; les erreurs (compte inexistant, 403) sont affichées par la
// page ou le layout admin plutôt que par l'erreur globale de l'application.
export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(accountQuery(id)).catch(() => null);
}

const roleItems: { value: Role; label: string }[] = [
  { value: "member", label: "Membre" },
  { value: "admin", label: "Administrateur" },
];

// Fiche d'un compte et actions d'administration (US1 à US5).
export default function AccountDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...accountQuery(id ?? 0), enabled: id !== null });
  const account = query.data;
  if (account) return <AccountView key={account.id} account={account} />;
  // Compte inexistant (supprimé, adresse saisie à la main) : « Page introuvable ».
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger ce compte. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function AccountView({ account }: { account: AdminAccount }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [disableOpen, setDisableOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [roleDraft, setRoleDraft] = useState<Role>(account.role);

  const path = { params: { path: { id: account.id } } };
  const disable = $api.useMutation("post", "/admin/accounts/{id}/disable");
  const enable = $api.useMutation("post", "/admin/accounts/{id}/enable");
  const setRole = $api.useMutation("put", "/admin/accounts/{id}/role");
  const remove = $api.useMutation("delete", "/admin/accounts/{id}");
  const sendReset = $api.useMutation("post", "/admin/accounts/{id}/password-reset");

  // Exécute l'action, annonce le résultat et rafraîchit fiche et liste (FR-017).
  const run = async (action: () => Promise<unknown>, success: string) => {
    setStatus("");
    setError("");
    try {
      await action();
      setStatus(success);
    } catch (e) {
      setError(isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.");
    } finally {
      await Promise.all([
        qc.invalidateQueries({ queryKey: accountQuery(account.id).queryKey }),
        qc.invalidateQueries({ queryKey: accountListKey }),
      ]);
    }
  };

  // Fiches gérées par le compte, chargées à l'ouverture du dialogue de
  // suppression (005, US5).
  const members = useQuery({ ...accountMembersQuery(account.id), enabled: deleteOpen });
  const managed = members.data?.items ?? [];
  const [membersChoice, setMembersChoice] = useState<"" | "delete" | "keep">("");
  const [membersError, setMembersError] = useState("");
  const [deleteAlert, setDeleteAlert] = useState("");

  const onDelete = async () => {
    setStatus("");
    setError("");
    setMembersError("");
    setDeleteAlert("");
    // Le choix est validé par le serveur (422 query.members), jamais ici
    // (constitution, principe VI).
    const query =
      managed.length > 0
        ? {
            memberIds: managed.map((m) => m.id),
            ...(membersChoice ? { members: membersChoice } : {}),
          }
        : {};
    try {
      await remove.mutateAsync({
        params: { path: { id: account.id }, query },
        querySerializer: { array: { style: "form", explode: false } },
      });
    } catch (e) {
      const problem = isProblem(e) ? e : null;
      const choice = problem?.errors?.find((x) => x.location === "query.members");
      if (choice) {
        setMembersError(choice.message ?? "");
        setDeleteOpen(true);
        return;
      }
      if (problem?.status === 409 && managed.length > 0) {
        // Liste des fiches changée entre-temps : dialogue rouvert et rechargé.
        setDeleteAlert(problem.detail ?? "");
        await members.refetch();
        setDeleteOpen(true);
        return;
      }
      setError(problem?.detail ?? "Une erreur inattendue s'est produite.");
      await qc.invalidateQueries({ queryKey: accountQuery(account.id).queryKey });
      return;
    }
    qc.removeQueries({ queryKey: accountQuery(account.id).queryKey });
    await Promise.all([
      qc.invalidateQueries({ queryKey: accountListKey }),
      qc.invalidateQueries({ queryKey: invitationListKey }),
      invalidateMembers(qc),
    ]);
    navigate("/admin/comptes", { state: { flash: `Le compte ${account.email} a été supprimé.` } });
  };

  const active = account.status === "active";
  const actions = account.isSelf ? null : (
    <>
      <ConfirmDialog
        tone="primary"
        title={`Changer le rôle de ${account.email}`}
        confirmLabel="Enregistrer"
        confirmDisabled={roleDraft === account.role}
        onOpenChange={(open) => {
          if (open) setRoleDraft(account.role);
        }}
        onConfirm={() =>
          run(
            () => setRole.mutateAsync({ ...path, body: { role: roleDraft } }),
            `${account.email} est maintenant ${roleLabel(roleDraft).toLowerCase()}.`,
          )
        }
        content={
          <Select
            label="Nouveau rôle"
            value={roleDraft}
            items={roleItems}
            onValueChange={setRoleDraft}
          />
        }
        trigger={
          <Button variant="secondary">
            <Icon icon={UserCog} className="h-4 w-4" />
            Changer le rôle
          </Button>
        }
      >
        <p>Un administrateur peut gérer les comptes et toutes les rubriques d'administration.</p>
      </ConfirmDialog>

      {active ? (
        <ConfirmDialog
          tone="primary"
          title="Envoyer un lien de réinitialisation ?"
          confirmLabel="Envoyer"
          onConfirm={() =>
            run(
              () => sendReset.mutateAsync(path),
              `Lien de réinitialisation envoyé à ${account.email}.`,
            )
          }
          trigger={
            <Button variant="secondary">
              <Icon icon={KeyRound} className="h-4 w-4" />
              Envoyer un lien de réinitialisation
            </Button>
          }
        >
          <p>Un e-mail de réinitialisation sera envoyé à {account.email}.</p>
        </ConfirmDialog>
      ) : null}

      {active ? (
        <ConfirmDialog
          title={`Désactiver ${account.email} ?`}
          confirmLabel="Désactiver"
          open={disableOpen}
          onOpenChange={setDisableOpen}
          onConfirm={() =>
            run(() => disable.mutateAsync(path), `Le compte ${account.email} a été désactivé.`)
          }
          trigger={
            <Button variant="danger-secondary">
              <Icon icon={Ban} className="h-4 w-4" />
              Désactiver
            </Button>
          }
        >
          <p>
            {account.email} sera déconnecté de tous ses appareils et ne pourra plus se connecter.
            Vous pourrez réactiver le compte.
          </p>
        </ConfirmDialog>
      ) : (
        <Button
          variant="secondary"
          onClick={() =>
            run(() => enable.mutateAsync(path), `Le compte ${account.email} a été réactivé.`)
          }
        >
          <Icon icon={RotateCcw} className="h-4 w-4" />
          Réactiver
        </Button>
      )}

      <ConfirmDialog
        title={`Supprimer définitivement ${account.email} ?`}
        confirmLabel="Supprimer"
        open={deleteOpen}
        onOpenChange={(open) => {
          setDeleteOpen(open);
          if (open) {
            setMembersChoice("");
            setMembersError("");
            setDeleteAlert("");
          }
        }}
        onConfirm={onDelete}
        content={
          managed.length > 0 ? (
            <ManagedMembers
              members={managed}
              choice={membersChoice}
              onChoice={setMembersChoice}
              error={membersError}
              alert={deleteAlert}
            />
          ) : null
        }
        extraActions={
          active ? (
            <Button
              variant="tertiary"
              onClick={() => {
                setDeleteOpen(false);
                setDisableOpen(true);
              }}
            >
              Désactiver plutôt
            </Button>
          ) : null
        }
        trigger={
          <Button variant="danger">
            <Icon icon={Trash2} className="h-4 w-4" />
            Supprimer
          </Button>
        }
      >
        <p>
          Cette action est irréversible : le compte et ses données de connexion seront effacés.
          {active ? " Pour seulement bloquer l'accès, désactivez plutôt le compte." : null}
        </p>
      </ConfirmDialog>
    </>
  );

  return (
    <Page
      title={account.email}
      intro={`${roleLabel(account.role)} · ${active ? "Compte actif" : "Compte désactivé"}`}
      actions={actions}
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
      {account.isSelf ? (
        <Alert kind="info" className="mb-6 max-w-3xl">
          Vous ne pouvez pas modifier votre propre compte ici. Utilisez{" "}
          <Link to="/mon-compte" className="font-bold underline underline-offset-2">
            « Mon compte »
          </Link>
          .
        </Alert>
      ) : null}

      <section
        aria-labelledby="compte-infos-titre"
        className="max-w-3xl rounded-lg border border-line bg-surface p-5"
      >
        <h2 id="compte-infos-titre" className="font-display text-lg font-bold">
          Informations
        </h2>
        <dl className="mt-3 grid gap-x-6 gap-y-3 sm:grid-cols-[12rem_minmax(0,1fr)]">
          <dt className="font-bold">Rôle</dt>
          <dd>
            <Badge tone={account.role === "admin" ? "brand" : "neutral"}>
              {roleLabel(account.role)}
            </Badge>
          </dd>
          <dt className="font-bold">État</dt>
          <dd className="flex flex-wrap items-center gap-2">
            {active ? (
              <Badge tone="success">Actif</Badge>
            ) : (
              <>
                <Badge tone="muted">Désactivé</Badge>
                {account.disabledAt ? (
                  <span>depuis le {formatDate(account.disabledAt)}</span>
                ) : null}
              </>
            )}
          </dd>
          <dt className="font-bold">Adresse confirmée</dt>
          <dd>{account.emailVerified ? "Oui" : "Non"}</dd>
          <dt className="font-bold">Créé le</dt>
          <dd>{formatDate(account.createdAt)}</dd>
          <dt className="font-bold">Dernière connexion</dt>
          <dd>{account.lastLoginAt ? formatDate(account.lastLoginAt) : "Jamais"}</dd>
        </dl>
      </section>
    </Page>
  );
}

interface ManagedMembersProps {
  members: { id: number; firstName: string; lastName: string }[];
  choice: "" | "delete" | "keep";
  onChoice: (choice: "delete" | "keep") => void;
  error: string;
  alert: string;
}

// Fiches gérées par le compte à supprimer et choix de leur sort (005, US5,
// FR-012b) : aucun choix par défaut, l'erreur vient du serveur.
function ManagedMembers({ members, choice, onChoice, error, alert }: ManagedMembersProps) {
  const n = members.length;
  return (
    <div className="flex flex-col gap-3">
      {alert ? (
        <Alert kind="danger" role="alert">
          {alert}
        </Alert>
      ) : null}
      <p>
        Ce compte gère {n} fiche{n > 1 ? "s" : ""} de membre :
      </p>
      <ul className="list-disc pl-6">
        {members.map((m) => (
          <li key={m.id} className="[overflow-wrap:anywhere]">
            {fullName(m)}
          </li>
        ))}
      </ul>
      <div>
        <p id="sort-fiches" className="mb-2 font-bold">
          Que faire de {n > 1 ? "ces fiches" : "cette fiche"} ?
        </p>
        <RadioGroup
          value={choice}
          onValueChange={(v) => onChoice(v as "delete" | "keep")}
          aria-labelledby="sort-fiches"
          aria-describedby={error ? "sort-fiches-erreur" : undefined}
          aria-invalid={error ? true : undefined}
          className="flex flex-col gap-2"
        >
          {(
            [
              ["delete", "Supprimer aussi ces fiches"],
              ["keep", "Conserver ces fiches sans compte"],
            ] as const
          ).map(([value, label]) => (
            // biome-ignore lint/a11y/noLabelWithoutControl: le contrôle est le bouton radio base-ui imbriqué.
            <label key={value} className="flex cursor-pointer items-center gap-2">
              <Radio.Root
                value={value}
                className="grid h-5 w-5 shrink-0 place-items-center rounded-full border border-line-strong bg-surface data-[checked]:border-brand"
              >
                <Radio.Indicator className="h-2.5 w-2.5 rounded-full bg-brand" />
              </Radio.Root>
              {label}
            </label>
          ))}
        </RadioGroup>
        {error ? (
          <p id="sort-fiches-erreur" className="mt-1.5 text-sm font-bold text-danger">
            {error}
          </p>
        ) : null}
      </div>
    </div>
  );
}
