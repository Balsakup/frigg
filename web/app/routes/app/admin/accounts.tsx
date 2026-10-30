import { useQueryClient } from "@tanstack/react-query";
import { RotateCw, Send, Trash2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { formatDate, roleLabel } from "~/lib/format";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Field } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { ScrollRegion } from "~/ui/ScrollRegion";

export const handle = { crumb: "Comptes" };

interface InviteForm {
  email: string;
  role: "member" | "admin";
}

// Rubrique « Comptes » : invitations par e-mail (US4).
export default function Accounts() {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState("");
  const list = $api.useQuery("get", "/admin/invitations");
  const listKey = $api.queryOptions("get", "/admin/invitations").queryKey;
  const create = $api.useMutation("post", "/admin/invitations");
  const resend = $api.useMutation("post", "/admin/invitations/{id}/resend");
  const cancel = $api.useMutation("delete", "/admin/invitations/{id}");
  const { register, handleSubmit, setError, reset, formState } = useForm<InviteForm>({
    defaultValues: { email: "", role: "member" },
  });

  const onSubmit = handleSubmit(async (body) => {
    setStatus("");
    try {
      const inv = await create.mutateAsync({ body });
      reset();
      setStatus(`Invitation envoyée à ${inv.email}.`);
      await queryClient.invalidateQueries({ queryKey: listKey });
    } catch (error) {
      if (isProblem(error)) applyProblemToForm(error, setError, ["email", "role"]);
    }
  });

  const onResend = async (id: number, email: string) => {
    setStatus("");
    await resend.mutateAsync({ params: { path: { id } } });
    setStatus(`Invitation renvoyée à ${email}.`);
    await queryClient.invalidateQueries({ queryKey: listKey });
  };

  const onCancel = async (id: number, email: string) => {
    setStatus("");
    await cancel.mutateAsync({ params: { path: { id } } });
    setStatus(`Invitation de ${email} annulée.`);
    await queryClient.invalidateQueries({ queryKey: listKey });
  };

  const items = list.data?.items ?? [];

  return (
    <Page title="Comptes" intro="Invitez les personnes de l'association à créer leur compte.">
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}

      <section
        aria-labelledby="inviter-titre"
        className="max-w-3xl rounded-lg border border-line bg-surface p-5"
      >
        <h2 id="inviter-titre" className="font-display text-lg font-bold">
          Inviter une personne
        </h2>
        <p className="mt-1 text-muted">
          Elle recevra un e-mail pour choisir son mot de passe. Le lien est valable 7 jours.
        </p>
        {formState.errors.root?.server ? (
          <Alert kind="danger" className="mt-4" role="alert">
            {formState.errors.root.server.message}
          </Alert>
        ) : null}
        <form
          onSubmit={onSubmit}
          noValidate
          className="mt-4 grid gap-4 sm:grid-cols-[minmax(0,1fr)_12rem_auto] sm:items-end"
        >
          <Field
            label="Adresse électronique"
            type="email"
            autoComplete="off"
            required
            error={formState.errors.email?.message}
            {...register("email", {
              required: "Saisissez l'adresse électronique de la personne à inviter.",
            })}
          />
          <div>
            <label htmlFor="invite-role" className="mb-1 block font-bold">
              Rôle
            </label>
            <select
              id="invite-role"
              className="h-11 w-full rounded-md border border-line-strong bg-surface px-3"
              {...register("role")}
            >
              <option value="member">Membre</option>
              <option value="admin">Administrateur</option>
            </select>
          </div>
          <Button type="submit" aria-disabled={formState.isSubmitting} className="h-11">
            <Icon icon={Send} className="h-4 w-4" />
            Envoyer l'invitation
          </Button>
        </form>
      </section>

      <section
        aria-labelledby="invitations-titre"
        className="mt-6 max-w-5xl rounded-lg border border-line bg-surface"
      >
        <div className="flex flex-wrap items-baseline justify-between gap-2 px-5 pb-2 pt-4">
          <h2 id="invitations-titre" className="font-display text-lg font-bold">
            Invitations en attente
          </h2>
          <p className="text-sm text-muted">
            {items.length} invitation{items.length > 1 ? "s" : ""}
          </p>
        </div>
        {list.isError ? (
          <Alert kind="danger" className="m-5">
            Impossible de charger les invitations. Rechargez la page.
          </Alert>
        ) : items.length === 0 ? (
          <p className="px-5 pb-5 text-muted">
            {list.isPending ? "Chargement…" : "Aucune invitation en attente."}
          </p>
        ) : (
          <ScrollRegion labelledBy="invitations-titre">
            <table className="w-full border-collapse text-left text-[15px]">
              <thead className="text-sm">
                <tr className="border-y border-line">
                  <th scope="col" className="whitespace-nowrap px-5 py-2.5">
                    Adresse
                  </th>
                  <th scope="col" className="whitespace-nowrap px-5 py-2.5">
                    Rôle
                  </th>
                  <th scope="col" className="whitespace-nowrap px-5 py-2.5">
                    Statut
                  </th>
                  <th scope="col" className="whitespace-nowrap px-5 py-2.5">
                    Valable jusqu'au
                  </th>
                  <th scope="col" className="whitespace-nowrap px-5 py-2.5">
                    Invitée par
                  </th>
                  <th scope="col" className="px-5 py-2.5 text-right">
                    <span className="sr-only">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((inv) => (
                  <tr key={inv.id} className="border-b border-line last:border-0">
                    <td className="px-5 py-3 [overflow-wrap:anywhere]">{inv.email}</td>
                    <td className="whitespace-nowrap px-5 py-3">{roleLabel(inv.role)}</td>
                    <td className="px-5 py-3">
                      {inv.status === "expired" ? (
                        <Badge tone="warning">Expirée</Badge>
                      ) : (
                        <Badge tone="info">En attente</Badge>
                      )}
                    </td>
                    <td className="whitespace-nowrap px-5 py-3">{formatDate(inv.expiresAt)}</td>
                    <td className="px-5 py-3 text-muted [overflow-wrap:anywhere]">
                      {inv.invitedBy ?? "—"}
                    </td>
                    <td className="px-5 py-3">
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => onResend(inv.id, inv.email)}
                        >
                          <Icon icon={RotateCw} className="h-4 w-4" />
                          Renvoyer<span className="sr-only"> l'invitation de {inv.email}</span>
                        </Button>
                        <ConfirmDialog
                          title="Annuler l'invitation ?"
                          confirmLabel="Annuler l'invitation"
                          onConfirm={() => onCancel(inv.id, inv.email)}
                          trigger={
                            <Button variant="danger-secondary" size="sm">
                              <Icon icon={Trash2} className="h-4 w-4" />
                              Annuler<span className="sr-only"> l'invitation de {inv.email}</span>
                            </Button>
                          }
                        >
                          <p>
                            Le lien envoyé à {inv.email} ne fonctionnera plus. Vous pourrez
                            l'inviter à nouveau.
                          </p>
                        </ConfirmDialog>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </ScrollRegion>
        )}
      </section>
    </Page>
  );
}
