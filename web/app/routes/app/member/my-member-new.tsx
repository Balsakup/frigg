import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { myCustomFieldsQuery } from "~/api/custom-fields";
import { invalidateMembers, myMemberListQuery } from "~/api/members";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { emptyMember } from "~/members/form";
import { MyMemberForm } from "~/members/MyMemberForm";
import { Page } from "~/shell/Page";
import { useSession } from "~/shell/useSession";
import { Alert } from "~/ui/Alert";

export const handle = { crumb: "Ajouter un membre" };

export async function clientLoader() {
  await Promise.all([
    queryClient.ensureQueryData(myMemberListQuery()).catch(() => undefined),
    queryClient.ensureQueryData(myCustomFieldsQuery()).catch(() => undefined),
  ]);
  return null;
}

// Ajout d'une fiche dans « Mes membres » (US1).
export default function MyMemberNew() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { account } = useSession();
  const list = useQuery(myMemberListQuery());
  const create = $api.useMutation("post", "/me/members");
  const fields = useQuery(myCustomFieldsQuery());

  if (list.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger vos membres. Rechargez la page.
      </Alert>
    );
  }
  if (!list.data || fields.isPending) return null;

  return (
    <Page title="Ajouter un membre" intro="Une personne que vous inscrivez à l'association.">
      {!list.data.actions.add ? (
        <Alert kind="info" className="mb-6 max-w-3xl">
          20 membres au plus par compte. Contactez le bureau pour en ajouter.
        </Alert>
      ) : null}
      <MyMemberForm
        defaultValues={{ ...emptyMember, isSelf: false }}
        canMarkSelf={list.data.actions.markSelf}
        accountEmail={account.email}
        customFields={fields.data?.member ?? []}
        submitLabel="Ajouter le membre"
        onSubmit={async (body) => {
          await create.mutateAsync({ body });
          await invalidateMembers(qc);
          navigate("/mes-membres", { state: { flash: "Membre ajouté." } });
        }}
      />
    </Page>
  );
}
