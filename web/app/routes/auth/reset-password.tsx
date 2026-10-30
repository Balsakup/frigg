import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { useFragmentToken } from "~/lib/fragment";
import { PASSWORD_HINT } from "~/lib/password";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field } from "~/ui/Field";
import type { Route } from "./+types/reset-password";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Nouveau mot de passe · Frigg" }];
}

interface ResetForm {
  password: string;
  confirmation: string;
}

// Choix d'un nouveau mot de passe depuis le lien reçu par e-mail (US5).
export default function ResetPassword() {
  const token = useFragmentToken();
  const [state, setState] = useState<"form" | "done" | "gone">(token ? "form" : "gone");
  const confirm = $api.useMutation("post", "/auth/password-reset/confirm");
  const { register, handleSubmit, setError, getValues, formState } = useForm<ResetForm>();

  const onSubmit = handleSubmit(async ({ password }) => {
    try {
      await confirm.mutateAsync({ body: { token, password } });
      setState("done");
    } catch (error) {
      if (isProblem(error) && error.status === 410) setState("gone");
      else if (isProblem(error)) applyProblemToForm(error, setError, ["password"]);
    }
  });

  return (
    <>
      <h1 className="font-display text-2xl font-bold">Nouveau mot de passe</h1>
      {state === "done" ? (
        <Alert kind="success" className="mt-4" role="status">
          Votre mot de passe a été modifié. Vous pouvez vous connecter.
        </Alert>
      ) : state === "gone" ? (
        <Alert kind="warning" className="mt-4" role="alert">
          Ce lien n'est plus valable.{" "}
          <Link
            to="/mot-de-passe-oublie"
            className="font-bold text-brand-ink underline underline-offset-2"
          >
            Demander un nouveau lien
          </Link>
        </Alert>
      ) : (
        <>
          {formState.errors.root?.server ? (
            <Alert kind="danger" className="mt-4" role="alert">
              {formState.errors.root.server.message}
            </Alert>
          ) : null}
          <form onSubmit={onSubmit} noValidate className="mt-6 flex flex-col gap-5">
            <Field
              label="Mot de passe"
              type="password"
              autoComplete="new-password"
              minLength={8}
              required
              hint={PASSWORD_HINT}
              error={formState.errors.password?.message}
              {...register("password", { required: "Choisissez un mot de passe." })}
            />
            <Field
              label="Confirmez le mot de passe"
              type="password"
              autoComplete="new-password"
              required
              error={formState.errors.confirmation?.message}
              {...register("confirmation", {
                validate: (v) =>
                  v === getValues("password") || "Les deux mots de passe ne correspondent pas.",
              })}
            />
            <Button type="submit" block aria-disabled={formState.isSubmitting}>
              Enregistrer le mot de passe
            </Button>
          </form>
        </>
      )}
      <p className="mt-5">
        <Link to="/connexion" className="font-bold text-brand-ink underline underline-offset-2">
          {state === "done" ? "Se connecter" : "Revenir à la connexion"}
        </Link>
      </p>
    </>
  );
}
