import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { invalidateMembers } from "~/api/members";
import { $api } from "~/api/queries";
import { NO_ACCOUNT } from "~/members/AccountPicker";
import { AdminMemberForm } from "~/members/AdminMemberForm";
import { emptyMember } from "~/members/form";
import { Page } from "~/shell/Page";

export const handle = { crumb: "Ajouter un membre" };

// Création d'une fiche par l'administration, avec ou sans compte (US4).
export default function MemberNew() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const create = $api.useMutation("post", "/admin/members");

  return (
    <Page
      title="Ajouter un membre"
      intro="Rattachez la fiche au compte de la famille, ou laissez-la sans compte : elle sera gérée par le bureau."
    >
      <AdminMemberForm
        defaultValues={{ ...emptyMember, accountId: NO_ACCOUNT }}
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
