import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";
import { invalidateMembers, myMemberListQuery, myMemberQuery } from "~/api/members";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { fullName, type Member, memberToForm } from "~/members/form";
import { MyMemberForm } from "~/members/MyMemberForm";
import { Page } from "~/shell/Page";
import { useSession } from "~/shell/useSession";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Icon } from "~/ui/Icon";
import NotFound from "../not-found";
import type { Route } from "./+types/my-member-detail";

export const handle = {
  crumb: (data: unknown) => {
    const m = data as Member | null;
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
  await queryClient.ensureQueryData(myMemberListQuery()).catch(() => undefined);
  return queryClient.ensureQueryData(myMemberQuery(id)).catch(() => null);
}

// Fiche d'un de mes membres : modification et suppression (US1). Une fiche
// d'un autre compte donne « Page introuvable » (US2).
export default function MyMemberDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...myMemberQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <MyMemberView key={query.data.id} member={query.data} />;
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

function MyMemberView({ member }: { member: Member }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { account } = useSession();
  const list = useQuery(myMemberListQuery());
  const update = $api.useMutation("put", "/me/members/{id}");
  const remove = $api.useMutation("delete", "/me/members/{id}");
  const [error, setError] = useState("");
  const path = { params: { path: { id: member.id } } };
  const name = fullName(member);

  const onDelete = async () => {
    setError("");
    try {
      await remove.mutateAsync(path);
    } catch (e) {
      setError(isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.");
      await invalidateMembers(qc);
      return;
    }
    qc.removeQueries({ queryKey: myMemberQuery(member.id).queryKey });
    await invalidateMembers(qc);
    navigate("/mes-membres", { state: { flash: "Membre supprimé." } });
  };

  return (
    <Page title={name} intro={member.isSelf ? "Votre fiche" : undefined}>
      {error ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {error}
        </Alert>
      ) : null}
      <MyMemberForm
        defaultValues={{ ...memberToForm(member), isSelf: member.isSelf }}
        canMarkSelf={member.isSelf || (list.data?.actions.markSelf ?? false)}
        accountEmail={account.email}
        submitLabel="Enregistrer"
        onSubmit={async (body) => {
          try {
            await update.mutateAsync({ ...path, body });
          } catch (e) {
            if (isProblem(e) && e.status === 404) await invalidateMembers(qc);
            throw e;
          }
          await invalidateMembers(qc);
          navigate("/mes-membres", { state: { flash: "Modifications enregistrées." } });
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
          </ConfirmDialog>
        }
      />
    </Page>
  );
}
