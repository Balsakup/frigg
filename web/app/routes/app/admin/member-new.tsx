import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { customFieldsQuery } from "~/api/custom-fields";
import { invalidateMembers } from "~/api/members";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { adminFormFields } from "~/customfields/form";
import { NO_ACCOUNT } from "~/members/AccountPicker";
import { AdminMemberForm } from "~/members/AdminMemberForm";
import { emptyMember } from "~/members/form";
import { Page } from "~/shell/Page";

export const handle = { crumb: "Ajouter un membre" };

export async function clientLoader() {
  await queryClient.ensureQueryData(customFieldsQuery()).catch(() => undefined);
  return null;
}

// Création d'une fiche par l'administration, avec ou sans compte (US4).
export default function MemberNew() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const create = $api.useMutation("post", "/admin/members");
  const fields = useQuery(customFieldsQuery());
  if (fields.isPending) return null;

  return (
    <Page
      title="Ajouter un membre"
      intro="Rattachez la fiche au compte de la famille, ou laissez-la sans compte : elle sera gérée par le bureau."
    >
      <AdminMemberForm
        defaultValues={{ ...emptyMember, accountId: NO_ACCOUNT }}
        customFields={adminFormFields(fields.data, "member")}
        submitLabel="Ajouter le membre"
        onSubmit={async (body) => {
          await create.mutateAsync({ body });
          await invalidateMembers(qc);
          navigate("/admin/membres", { state: { flash: "Membre ajouté." } });
        }}
      />
    </Page>
  );
}
