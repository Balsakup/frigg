import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { invalidateMembers, memberQuery } from "~/api/members";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { formatAge, formatDate, formatDay, formatPhone, membershipStatusLabel } from "~/lib/format";
import { NO_ACCOUNT } from "~/members/AccountPicker";
import { AdminMemberForm } from "~/members/AdminMemberForm";
import { type AdminMember, fullName, memberToForm } from "~/members/form";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button, buttonClasses } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Icon } from "~/ui/Icon";
import NotFound from "../not-found";
import type { Route } from "./+types/member-detail";

export const handle = {
  crumb: (data: unknown) => {
    const m = data as AdminMember | null;
    return m ? fullName(m) : "Membre introuvable";
  },
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(memberQuery(id)).catch(() => null);
}

// Fiche d'un membre dans l'administration : informations, modification,
// rattachement et suppression (US3, US4).
export default function MemberDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...memberQuery(id ?? 0), enabled: id !== null });
  if (query.data) {
    // Clé stable : après un conflit, la fiche rechargée met à jour les
    // informations sans effacer les saisies en cours.
    return <MemberView key={query.data.id} member={query.data} />;
  }
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger cette fiche. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function MemberView({ member }: { member: AdminMember }) {
  const memberships = member.memberships ?? [];
  const qc = useQueryClient();
  const navigate = useNavigate();
  const update = $api.useMutation("put", "/admin/members/{id}");
  const remove = $api.useMutation("delete", "/admin/members/{id}");
  const [error, setError] = useState("");
  const path = { params: { path: { id: member.id } } };
  const name = fullName(member);
  const account = member.account
    ? { id: member.account.id, label: member.account.email, disabled: member.account.disabled }
    : NO_ACCOUNT;

  const onDelete = async () => {
    setError("");
    try {
      await remove.mutateAsync(path);
    } catch (e) {
      setError(isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.");
      await invalidateMembers(qc);
      return;
    }
    qc.removeQueries({ queryKey: memberQuery(member.id).queryKey });
    await invalidateMembers(qc);
    navigate("/admin/membres", { state: { flash: "Membre supprimé." } });
  };

  return (
    <Page title={name} intro={`${formatAge(member.age)} · né(e) le ${formatDay(member.birthDate)}`}>
      {error ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {error}
        </Alert>
      ) : null}

      <section
        aria-labelledby="membre-infos-titre"
        className="mb-6 max-w-3xl rounded-lg border border-line bg-surface p-5"
      >
        <h2 id="membre-infos-titre" className="font-display text-lg font-bold">
          Informations
        </h2>
        <dl className="mt-3 grid gap-x-6 gap-y-3 sm:grid-cols-[12rem_minmax(0,1fr)]">
          <dt className="font-bold">Compte de rattachement</dt>
          <dd className="flex flex-wrap items-center gap-2 [overflow-wrap:anywhere]">
            {member.account ? (
              <>
                <Link
                  to={`/admin/comptes/${member.account.id}`}
                  className="font-bold text-brand-ink underline underline-offset-2"
                >
                  {member.account.email}
                </Link>
                {member.account.disabled ? <Badge tone="muted">désactivé</Badge> : null}
              </>
            ) : (
              "Sans compte"
            )}
          </dd>
          <dt className="font-bold">Fiche du titulaire</dt>
          <dd>{member.isSelf ? "Oui" : "Non"}</dd>
          <dt className="font-bold">Téléphone</dt>
          <dd>{member.phone ? formatPhone(member.phone) : "Non renseigné"}</dd>
          <dt className="font-bold">Créée le</dt>
          <dd>{formatDate(member.createdAt)}</dd>
          <dt className="font-bold">Modifiée le</dt>
          <dd>{formatDate(member.updatedAt)}</dd>
        </dl>
      </section>

      <section
        aria-labelledby="membre-adhesions-titre"
        className="mb-6 max-w-3xl rounded-lg border border-line bg-surface p-5"
      >
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 id="membre-adhesions-titre" className="font-display text-lg font-bold">
            Adhésions
          </h2>
          <Link
            to={`/admin/adhesions/nouvelle?membre=${member.id}`}
            className={buttonClasses({ variant: "secondary", size: "sm" })}
          >
            Créer une adhésion
          </Link>
        </div>
        {memberships.length === 0 ? (
          <p className="mt-2 text-muted">Aucune adhésion.</p>
        ) : (
          <ul className="mt-3 divide-y divide-line">
            {memberships.map((ms) => (
              <li key={ms.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                <Link
                  to={`/admin/adhesions/${ms.id}`}
                  className="font-bold text-brand-ink underline underline-offset-2"
                >
                  Saison {ms.season.label}
                </Link>
                <MembershipBadges status={ms.status} payment={ms.payment} />
              </li>
            ))}
          </ul>
        )}
      </section>

      <AdminMemberForm
        defaultValues={{ ...memberToForm(member), accountId: account }}
        isSelf={member.isSelf}
        submitLabel="Enregistrer"
        onSubmit={async (body) => {
          try {
            await update.mutateAsync({
              ...path,
              body: { ...body, expectedAccountId: member.account?.id ?? null },
            });
          } catch (e) {
            // Fiche supprimée ou rattachement changé entre-temps : message et
            // fiche rechargée.
            if (isProblem(e) && (e.status === 404 || e.status === 409)) {
              setError(e.detail ?? "Une erreur inattendue s'est produite.");
              await invalidateMembers(qc);
              return;
            }
            throw e;
          }
          await invalidateMembers(qc);
          navigate("/admin/membres", { state: { flash: "Modifications enregistrées." } });
        }}
        extra={
          <ConfirmDialog
            title={`Supprimer ${name} ?`}
            confirmLabel="Supprimer"
            onConfirm={onDelete}
            trigger={
              <Button variant="danger-secondary" type="button">
                <Icon icon={Trash2} className="h-4 w-4" />
                Supprimer
              </Button>
            }
          >
            <p>La fiche et ses informations seront effacées définitivement.</p>
            {memberships.length > 0 ? (
              <>
                <p className="mt-2">
                  {memberships.length === 1
                    ? "Son adhésion sera aussi supprimée :"
                    : `Ses ${memberships.length} adhésions seront aussi supprimées :`}
                </p>
                <ul className="mt-1 list-disc pl-5">
                  {memberships.map((ms) => (
                    <li key={ms.id}>
                      Saison {ms.season.label} — {membershipStatusLabel(ms.status).toLowerCase()}
                    </li>
                  ))}
                </ul>
              </>
            ) : null}
          </ConfirmDialog>
        }
      />
    </Page>
  );
}
