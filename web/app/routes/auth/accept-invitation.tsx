import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api, sessionQuery } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { roleLabel } from "~/lib/format";
import { useFragmentToken } from "~/lib/fragment";
import { PASSWORD_HINT } from "~/lib/password";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field } from "~/ui/Field";
import type { Route } from "./+types/accept-invitation";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Invitation · Frigg" }];
}

interface AcceptForm {
  password: string;
  confirmation: string;
}

// Acceptation d'une invitation (US4) : choix du mot de passe, puis connexion.
export default function AcceptInvitation() {
  const token = useFragmentToken();
  const navigate = useNavigate();
  const preview = $api.useQuery(
    "post",
    "/auth/invitation/lookup",
    { body: { token } },
    { enabled: token !== "" },
  );
  const accept = $api.useMutation("post", "/auth/invitation/accept");
  const { register, handleSubmit, setError, getValues, formState } = useForm<AcceptForm>();

  const onSubmit = handleSubmit(async ({ password }) => {
    try {
      const session = await accept.mutateAsync({ body: { token, password } });
      queryClient.setQueryData(sessionQuery.queryKey, session);
      navigate(session.account.role === "admin" ? "/admin" : "/", { replace: true });
    } catch (error) {
      if (isProblem(error)) applyProblemToForm(error, setError, ["password"]);
    }
  });

  if (!token || preview.isError) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Invitation</h1>
        <Alert kind="warning" className="mt-4">
          {isProblem(preview.error) && preview.error.detail
            ? preview.error.detail
            : "Cette invitation n'est plus valable."}{" "}
          Contactez l'association pour en recevoir une nouvelle.
        </Alert>
      </>
    );
  }
  if (!preview.data) {
    return <p role="status">Chargement de l'invitation…</p>;
  }

  return (
    <>
      <h1 className="font-display text-2xl font-bold">Créer votre compte</h1>
      <p className="mt-2 text-muted">
        Choisissez votre mot de passe pour rejoindre l'espace en ligne de l'association en tant que{" "}
        {roleLabel(preview.data.role).toLowerCase()}.
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
          value={preview.data.email}
          readOnly
          autoComplete="username"
        />
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
          Créer mon compte
        </Button>
      </form>
    </>
  );
}
