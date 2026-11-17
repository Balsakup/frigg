import { useQueryClient } from "@tanstack/react-query";
import { Controller, useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { sessionQuery } from "~/api/queries";
import { type Tracking, useCreateTrackingAccount, useTracking } from "~/api/tracking";
import { PASSWORD_HINT } from "~/lib/password";
import { LinkExpired } from "~/public/LinkExpired";
import { useTrackingAccess } from "~/public/useTrackingAccess";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field } from "~/ui/Field";
import { RadioGroup } from "~/ui/RadioGroup";
import type { Route } from "./+types/tracking-account";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Créer mon compte · Frigg" }];
}

const linkClass = "font-bold text-brand-ink underline underline-offset-2";

// Valeur de l'option « Aucune » de la fiche « Vous ».
const NONE = "aucune";

interface AccountForm {
  password: string;
  confirmation: string;
  selfMemberId: string;
}

// Création d'un compte depuis le suivi par lien (016, US4 ;
// contracts/ui-routes.md › /suivi/compte) : mot de passe, fiche « Vous » s'il
// y a plusieurs fiches, puis connexion et redirection vers l'espace membre.
export default function TrackingAccountPage() {
  const access = useTrackingAccess();
  const tracking = useTracking(access.kind === "ready");

  if (access.kind === "gone" || tracking.expired) return <LinkExpired />;
  if (access.kind === "error" || tracking.isError) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Créer mon compte</h1>
        <Alert kind="danger" role="alert" className="mt-4">
          {access.kind === "error"
            ? access.message
            : "Impossible de charger votre suivi. Rechargez la page."}
        </Alert>
      </>
    );
  }
  if (!tracking.data) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Créer mon compte</h1>
        <p role="status" className="mt-4">
          Chargement…
        </p>
      </>
    );
  }
  if (!tracking.data.canCreateAccount) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Créer mon compte</h1>
        <Alert kind="warning" className="mt-4">
          Un compte ne peut pas être créé pour cette adresse : soit un compte existe déjà, et vous
          pouvez vous connecter avec elle, soit elle suit plus de 20 fiches, et l'association peut
          créer votre compte.
        </Alert>
        <ul className="mt-5 flex flex-col gap-2">
          <li>
            <Link to="/connexion" className={linkClass}>
              Se connecter
            </Link>
          </li>
          <li>
            <Link to="/suivi" className={linkClass}>
              Revenir à mon suivi
            </Link>
          </li>
        </ul>
      </>
    );
  }
  return <AccountFormView tracking={tracking.data} />;
}

function AccountFormView({ tracking }: { tracking: Tracking }) {
  const members = tracking.members ?? [];
  const several = members.length > 1;
  const navigate = useNavigate();
  const qc = useQueryClient();
  const create = useCreateTrackingAccount();
  const { register, control, handleSubmit, setError, getValues, formState } = useForm<AccountForm>({
    defaultValues: { password: "", confirmation: "", selfMemberId: "" },
  });

  const onSubmit = handleSubmit(async ({ password, selfMemberId }) => {
    const self = several && selfMemberId !== NONE ? Number(selfMemberId) : null;
    try {
      const session = await create.mutateAsync({ body: { password, selfMemberId: self } });
      // L'accès par lien est retiré : les lectures par lien sont oubliées et la
      // session du nouveau compte prend le relais.
      for (const key of [
        "/tracking",
        "/tracking/memberships/{id}",
        "/tracking/registrations/{id}",
      ]) {
        qc.removeQueries({ queryKey: ["get", key] });
      }
      qc.setQueryData(sessionQuery.queryKey, session);
      navigate("/", { replace: true });
    } catch (error) {
      if (isProblem(error)) applyProblemToForm(error, setError, ["password", "selfMemberId"]);
    }
  });

  return (
    <>
      <h1 className="font-display text-2xl font-bold">Créer mon compte</h1>
      <p className="mt-2 text-muted">
        Choisissez votre mot de passe : vous retrouverez vos demandes et vos inscriptions dans votre
        espace membre, sans lien de suivi.
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
          value={tracking.email}
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
        {several ? (
          <Controller
            name="selfMemberId"
            control={control}
            rules={{ required: "Indiquez votre fiche, ou « Aucune »." }}
            render={({ field }) => (
              <RadioGroup
                legend="Laquelle de ces fiches est la vôtre ?"
                hint="Elle sera marquée « Vous » dans votre espace membre."
                required
                value={field.value}
                onValueChange={field.onChange}
                error={formState.errors.selfMemberId?.message}
                options={[
                  ...members.map((m) => ({
                    value: String(m.id),
                    label: `${m.firstName} ${m.lastName}`,
                  })),
                  { value: NONE, label: "Aucune" },
                ]}
              />
            )}
          />
        ) : null}
        <Button type="submit" block aria-disabled={formState.isSubmitting}>
          Créer mon compte
        </Button>
      </form>
      <p className="mt-5">
        <Link to="/suivi" className={linkClass}>
          Revenir à mon suivi
        </Link>
      </p>
    </>
  );
}
