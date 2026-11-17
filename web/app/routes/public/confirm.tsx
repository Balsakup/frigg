import { useEffect, useRef, useState } from "react";
import { Link, Navigate } from "react-router";
import { isProblem } from "~/api/problem";
import { type ConfirmOutcome, useConfirmPublicSubmission } from "~/api/public";
import { useFragmentToken } from "~/public/useFragmentToken";
import { Alert } from "~/ui/Alert";
import { buttonClasses } from "~/ui/Button";
import type { Route } from "./+types/confirm";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Confirmation de la demande · Frigg" }];
}

type State =
  | { kind: "pending" }
  | { kind: "done"; outcome: ConfirmOutcome }
  | { kind: "gone" }
  | { kind: "error"; message: string };

const linkClass = "font-bold text-brand-ink underline underline-offset-2";

// Confirmation d'un dépôt public depuis le lien de l'e-mail (016, US1 ;
// contracts/ui-routes.md › /confirmer-demande). Le jeton est lu dans le
// fragment, effacé de la barre d'adresse, puis envoyé une seule fois dans le
// corps de la requête.
export default function ConfirmPublicSubmission() {
  const token = useFragmentToken();
  const confirm = useConfirmPublicSubmission();
  const [state, setState] = useState<State>(token ? { kind: "pending" } : { kind: "gone" });
  const started = useRef(false);

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true;
    confirm
      .mutateAsync({ body: { token } })
      .then((outcome) => setState({ kind: "done", outcome }))
      .catch((e: unknown) => {
        if (isProblem(e) && e.status === 410) {
          setState({ kind: "gone" });
          return;
        }
        setState({
          kind: "error",
          message:
            (isProblem(e) && e.detail) ||
            "Une erreur inattendue s'est produite. Réessayez dans quelques minutes.",
        });
      });
  }, [token, confirm]);

  switch (state.kind) {
    case "pending":
      return (
        <>
          <h1 className="font-display text-2xl font-bold">Confirmation de la demande</h1>
          <p role="status" className="mt-4">
            Confirmation en cours…
          </p>
        </>
      );
    case "gone":
      return (
        <>
          <h1 className="font-display text-2xl font-bold">Ce lien n'est plus valable</h1>
          <p className="mt-4">
            Le lien de confirmation a déjà servi ou a expiré : il n'est valable que 48 heures.
          </p>
          <ul className="mt-5 flex flex-col gap-2">
            <li>
              <Link to="/adherer" className={linkClass}>
                Recommencer ma demande
              </Link>
            </li>
            <li>
              <Link to="/suivi/lien" className={linkClass}>
                Recevoir mon lien de suivi
              </Link>
            </li>
          </ul>
        </>
      );
    case "error":
      return (
        <>
          <h1 className="font-display text-2xl font-bold">Confirmation de la demande</h1>
          <Alert kind="danger" role="alert" className="mt-4">
            {state.message}
          </Alert>
        </>
      );
    case "done":
      return <Outcome outcome={state.outcome} />;
  }
}

function Outcome({ outcome }: { outcome: ConfirmOutcome }) {
  const registration = outcome.kind === "registration";
  switch (outcome.outcome) {
    case "tracking":
      // « Créer un compte » coché au dépôt : l'accès par lien est ouvert, la
      // personne choisit son mot de passe (US4).
      if (outcome.createAccount) return <Navigate to="/suivi/compte" replace />;
      return (
        <>
          <h1 className="font-display text-2xl font-bold">
            {registration ? "Inscription confirmée" : "Demande enregistrée"}
          </h1>
          <Alert kind="success" role="status" className="mt-4">
            Votre adresse est confirmée.{" "}
            {registration
              ? "Votre inscription est enregistrée."
              : "Votre demande d'adhésion sera examinée par le bureau."}
          </Alert>
          <p className="mt-4">
            Un e-mail vous a été envoyé avec votre lien de suivi. Gardez-le : il vous permet de
            suivre votre demande et de la régler.
          </p>
          <p className="mt-5">
            <Link to="/suivi" className={buttonClasses()}>
              Voir mon suivi
            </Link>
          </p>
        </>
      );
    case "account":
      return (
        <>
          <h1 className="font-display text-2xl font-bold">
            Votre demande est dans votre espace membre
          </h1>
          <p className="mt-4">
            Un compte existe pour cette adresse : connectez-vous pour suivre votre{" "}
            {registration ? "inscription" : "demande"}.
          </p>
          <p className="mt-5">
            <Link
              to={`/connexion?suite=${encodeURIComponent(registration ? "/mes-inscriptions" : "/adhesions")}`}
              className={buttonClasses()}
            >
              Se connecter
            </Link>
          </p>
        </>
      );
    case "duplicate":
      return (
        <>
          <h1 className="font-display text-2xl font-bold">Cette demande existe déjà</h1>
          <p className="mt-4">
            {registration
              ? "Cette personne est déjà inscrite à cet événement."
              : "Une demande d'adhésion pour cette saison est déjà enregistrée pour cette personne."}
          </p>
          <p className="mt-5">
            <Link to="/suivi" className={linkClass}>
              Voir mon suivi
            </Link>
          </p>
        </>
      );
    case "closed":
      return (
        <>
          <h1 className="font-display text-2xl font-bold">Demande impossible</h1>
          <Alert kind="warning" role="alert" className="mt-4">
            {outcome.reason ?? "Cette demande n'est plus possible."}
          </Alert>
          {registration ? null : (
            <p className="mt-5">
              <Link to="/adherer" className={linkClass}>
                Revenir à la page d'adhésion
              </Link>
            </p>
          )}
        </>
      );
  }
}
