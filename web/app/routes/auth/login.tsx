import { useForm } from "react-hook-form";
import { Link, redirect, useNavigate, useSearchParams } from "react-router";
import { isProblem } from "~/api/problem";
import { $api, sessionQuery } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { homeFor } from "~/lib/home";
import { safeRedirect } from "~/lib/redirect";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field } from "~/ui/Field";
import type { Route } from "./+types/login";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Connexion · Frigg" }];
}

// Une personne déjà connectée est renvoyée vers son accueil.
export async function clientLoader() {
  try {
    const session = await queryClient.fetchQuery(sessionQuery);
    return redirect(homeFor(session.account.role));
  } catch {
    return null;
  }
}

interface LoginForm {
  email: string;
  password: string;
}

export default function Login() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const { register, handleSubmit, setError, formState } = useForm<LoginForm>();
  const login = $api.useMutation("post", "/auth/login");

  const onSubmit = handleSubmit(async (body) => {
    try {
      const session = await login.mutateAsync({ body });
      queryClient.setQueryData(sessionQuery.queryKey, session);
      navigate(safeRedirect(params.get("suite")) ?? homeFor(session.account.role), {
        replace: true,
      });
    } catch (error) {
      setError("root.server", {
        message:
          isProblem(error) && error.detail ? error.detail : "Connexion impossible pour le moment.",
      });
    }
  });

  return (
    <>
      <h1 tabIndex={-1} className="font-display text-2xl font-bold">
        Connexion
      </h1>
      {params.get("expiree") === "1" && !formState.errors.root ? (
        <Alert kind="info" className="mt-4" role="status">
          Votre session a expiré. Reconnectez-vous.
        </Alert>
      ) : null}
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
        <Field
          label="Mot de passe"
          type="password"
          autoComplete="current-password"
          required
          error={formState.errors.password?.message}
          {...register("password", { required: "Saisissez votre mot de passe." })}
        />
        <Button type="submit" block aria-disabled={formState.isSubmitting}>
          Se connecter
        </Button>
      </form>
      <p className="mt-5">
        <Link
          to="/mot-de-passe-oublie"
          className="font-bold text-brand-ink underline underline-offset-2"
        >
          Mot de passe oublié ?
        </Link>
      </p>
    </>
  );
}
