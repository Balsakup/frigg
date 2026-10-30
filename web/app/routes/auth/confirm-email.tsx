import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { $api, sessionQuery } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { useFragmentToken } from "~/lib/fragment";
import { Alert } from "~/ui/Alert";
import type { Route } from "./+types/confirm-email";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Confirmation de l'adresse · Frigg" }];
}

// Confirmation de l'adresse depuis le lien reçu par e-mail (US6), connecté ou non.
export default function ConfirmEmail() {
  const token = useFragmentToken();
  const confirm = $api.useMutation("post", "/auth/email-verification/confirm");
  const [state, setState] = useState<"pending" | "done" | "gone">(token ? "pending" : "gone");
  const started = useRef(false);

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true;
    confirm
      .mutateAsync({ body: { token } })
      .then(() => {
        setState("done");
        return queryClient.invalidateQueries({ queryKey: sessionQuery.queryKey });
      })
      .catch(() => setState("gone"));
  }, [token, confirm]);

  return (
    <>
      <h1 className="font-display text-2xl font-bold">Confirmation de l'adresse</h1>
      {state === "pending" ? (
        <p role="status" className="mt-4">
          Confirmation en cours…
        </p>
      ) : state === "done" ? (
        <Alert kind="success" className="mt-4" role="status">
          Votre adresse électronique est confirmée.
        </Alert>
      ) : (
        <Alert kind="warning" className="mt-4" role="alert">
          Ce lien n'est plus valable. Connectez-vous pour en demander un nouveau.
        </Alert>
      )}
      <p className="mt-5">
        <Link to="/" className="font-bold text-brand-ink underline underline-offset-2">
          Aller à l'accueil
        </Link>
      </p>
    </>
  );
}
