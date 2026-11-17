import { useState } from "react";
import { useForm } from "react-hook-form";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { useRequestTrackingLink } from "~/api/tracking";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field } from "~/ui/Field";
import type { Route } from "./+types/tracking-request";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Recevoir mon lien de suivi · Frigg" }];
}

// « Recevoir mon lien de suivi » (016, US2 ; contracts/ui-routes.md ›
// /suivi/lien). Le message de réponse est le même pour toute adresse (FR-017).
export default function TrackingRequest() {
  const [message, setMessage] = useState<string | null>(null);
  const request = useRequestTrackingLink();
  const { register, handleSubmit, setError, formState } = useForm<{ email: string }>();

  const onSubmit = handleSubmit(async (body) => {
    try {
      const out = await request.mutateAsync({ body });
      setMessage(out.message);
    } catch (error) {
      if (isProblem(error)) applyProblemToForm(error, setError, ["email"]);
    }
  });

  return (
    <>
      <h1 className="font-display text-2xl font-bold">Recevoir mon lien de suivi</h1>
      {message ? (
        <Alert kind="success" className="mt-4" role="status">
          {message} Le lien est valable 30 jours. Pensez à vérifier vos courriers indésirables.
        </Alert>
      ) : (
        <>
          <p className="mt-2 text-muted">
            Saisissez l'adresse utilisée pour votre demande : vous recevrez un lien personnel pour
            suivre et régler vos demandes, sans compte.
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
              Recevoir mon lien
            </Button>
          </form>
        </>
      )}
    </>
  );
}
