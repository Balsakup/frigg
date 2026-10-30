import { useState } from "react";
import { useForm } from "react-hook-form";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { roleLabel } from "~/lib/format";
import { PASSWORD_HINT } from "~/lib/password";
import { Page } from "~/shell/Page";
import { useSession } from "~/shell/useSession";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { Field } from "~/ui/Field";

export const handle = { crumb: "Mon compte" };

interface PasswordForm {
  currentPassword: string;
  newPassword: string;
  confirmation: string;
}

// « Mon compte » (US7) : identité, rôle et changement de mot de passe.
export default function Account() {
  const { account } = useSession();
  const [status, setStatus] = useState("");
  const change = $api.useMutation("put", "/me/password");
  const { register, handleSubmit, setError, getValues, reset, formState } = useForm<PasswordForm>();

  const onSubmit = handleSubmit(async ({ currentPassword, newPassword }) => {
    setStatus("");
    try {
      await change.mutateAsync({ body: { currentPassword, newPassword } });
      reset();
      setStatus("Votre mot de passe a été modifié. Vos autres sessions ont été fermées.");
    } catch (error) {
      if (isProblem(error)) applyProblemToForm(error, setError, ["currentPassword", "newPassword"]);
    }
  });

  return (
    <Page title="Mon compte" intro="Vos informations de connexion.">
      <section
        aria-labelledby="identite-titre"
        className="max-w-2xl rounded-lg border border-line bg-surface p-5"
      >
        <h2 id="identite-titre" className="font-display text-lg font-bold">
          Identité
        </h2>
        <dl className="mt-3 grid gap-x-6 gap-y-3 sm:grid-cols-[12rem_minmax(0,1fr)]">
          <dt className="font-bold">Adresse électronique</dt>
          <dd className="flex flex-wrap items-center gap-2 [overflow-wrap:anywhere]">
            {account.email}
            {account.emailVerified ? (
              <Badge tone="success">Confirmée</Badge>
            ) : (
              <Badge tone="warning">Non confirmée</Badge>
            )}
          </dd>
          <dt className="font-bold">Rôle</dt>
          <dd>{roleLabel(account.role)}</dd>
        </dl>
      </section>

      <section
        aria-labelledby="mdp-titre"
        className="mt-6 max-w-2xl rounded-lg border border-line bg-surface p-5"
      >
        <h2 id="mdp-titre" className="font-display text-lg font-bold">
          Changer de mot de passe
        </h2>
        <p role="status" className="sr-only">
          {status}
        </p>
        {status ? (
          <Alert kind="success" className="mt-4">
            {status}
          </Alert>
        ) : null}
        {formState.errors.root?.server ? (
          <Alert kind="danger" className="mt-4" role="alert">
            {formState.errors.root.server.message}
          </Alert>
        ) : null}
        <form onSubmit={onSubmit} noValidate className="mt-4 flex flex-col gap-5">
          <Field
            label="Mot de passe actuel"
            type="password"
            autoComplete="current-password"
            required
            error={formState.errors.currentPassword?.message}
            {...register("currentPassword", { required: "Saisissez votre mot de passe actuel." })}
          />
          <Field
            label="Nouveau mot de passe"
            type="password"
            autoComplete="new-password"
            minLength={8}
            required
            hint={PASSWORD_HINT}
            error={formState.errors.newPassword?.message}
            {...register("newPassword", { required: "Choisissez un nouveau mot de passe." })}
          />
          <Field
            label="Confirmez le nouveau mot de passe"
            type="password"
            autoComplete="new-password"
            required
            error={formState.errors.confirmation?.message}
            {...register("confirmation", {
              validate: (v) =>
                v === getValues("newPassword") || "Les deux mots de passe ne correspondent pas.",
            })}
          />
          <div>
            <Button type="submit" aria-disabled={formState.isSubmitting}>
              Enregistrer le mot de passe
            </Button>
          </div>
        </form>
      </section>
    </Page>
  );
}
