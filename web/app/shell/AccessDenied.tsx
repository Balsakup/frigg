import { ShieldAlert } from "lucide-react";
import { Link } from "react-router";
import { Icon } from "~/ui/Icon";
import { Page } from "./Page";

// Rubrique non autorisée pour le rôle (FR-023) : aucune donnée n'est chargée.
export function AccessDenied() {
  return (
    <Page title="Accès refusé" breadcrumbs={false}>
      <div className="flex max-w-prose items-start gap-3 rounded-lg border border-line bg-surface p-5">
        <span className="text-danger">
          <Icon icon={ShieldAlert} />
        </span>
        <div>
          <p>Cette rubrique est réservée aux administrateurs de l'association.</p>
          <Link
            to="/"
            className="mt-3 inline-block font-bold text-brand-ink underline underline-offset-2"
          >
            Revenir à l'accueil
          </Link>
        </div>
      </div>
    </Page>
  );
}
