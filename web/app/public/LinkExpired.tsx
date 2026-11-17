import { Link } from "react-router";

// Lien personnel absent, expiré ou révoqué (016 FR-018) : aucune donnée, une
// invitation à demander un nouveau lien.
export function LinkExpired() {
  return (
    <>
      <h1 className="font-display text-2xl font-bold">Ce lien n'est plus valable</h1>
      <p className="mt-4">
        Votre lien de suivi a expiré, a été remplacé ou l'accès a été fermé sur cet appareil.
        Demandez un nouveau lien : il vous sera envoyé par e-mail.
      </p>
      <p className="mt-5">
        <Link to="/suivi/lien" className="font-bold text-brand-ink underline underline-offset-2">
          Recevoir un nouveau lien
        </Link>
      </p>
    </>
  );
}
