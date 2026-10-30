import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field } from "~/ui/Field";
import type { Route } from "./+types/forgot-password";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Mot de passe oublié · Frigg" }];
}

// Demande d'un lien de réinitialisation (US5) ; message neutre, que l'adresse
// corresponde à un compte ou non.
export default function ForgotPassword() {
  const [sent, setSent] = useState(false);
  const request = $api.useMutation("post", "/auth/password-reset");
  const { register, handleSubmit, setError, formState } = useForm<{ email: string }>();

  const onSubmit = handleSubmit(async (body) => {
    try {
      await request.mutateAsync({ body });
      setSent(true);
    } catch (error) {
      if (isProblem(error)) applyProblemToForm(error, setError, ["email"]);
    }
  });

  return (
    <>
      <h1 className="font-display text-2xl font-bold">Mot de passe oublié</h1>
      {sent ? (
        <Alert kind="success" className="mt-4" role="status">
          Si un compte existe pour cette adresse, un lien de réinitialisation vient d'être envoyé.
          Pensez à vérifier vos courriers indésirables.
        </Alert>
      ) : (
        <>
          <p className="mt-2 text-muted">
            Saisissez votre adresse : vous recevrez un lien pour choisir un nouveau mot de passe.
          </p>
          {formState.errors.root?.server ? (
            <Alert kind="danger" className="mt-4" role="alert">
              {formState.errors.root.server.message}
            </Alert>
          ) : null}
          <form onSubmit={onSubmit} noValidate className="mt-6 flex flex-col gap-5">
            <Field
              label="Adresse électronique"
              type="email"
              autoComplete="email"
              required
              error={formState.errors.email?.message}
              {...register("email", { required: "Saisissez votre adresse électronique." })}
            />
            <Button type="submit" block aria-disabled={formState.isSubmitting}>
              Recevoir le lien
            </Button>
          </form>
        </>
      )}
      <p className="mt-5">
        <Link to="/connexion" className="font-bold text-brand-ink underline underline-offset-2">
          Revenir à la connexion
        </Link>
      </p>
    </>
  );
}
