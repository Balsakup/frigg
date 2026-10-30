import { Link } from "react-router";
import { Page } from "~/shell/Page";

// Adresse inexistante pour une personne connectée (FR-023).
export default function NotFound() {
  return (
    <Page title="Page introuvable" breadcrumbs={false}>
      <p className="max-w-prose">La page demandée n'existe pas ou a été déplacée.</p>
      <Link
        to="/"
        className="mt-3 inline-block font-bold text-brand-ink underline underline-offset-2"
      >
        Revenir à l'accueil
      </Link>
    </Page>
  );
}
