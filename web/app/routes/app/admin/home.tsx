import { Page } from "~/shell/Page";

// Tableau de bord de l'espace administration ; les indicateurs viendront avec
// les fonctionnalités métier.
export default function AdminHome() {
  return (
    <Page title="Tableau de bord" intro="Vue d'ensemble de l'activité de l'association.">
      <section className="max-w-prose rounded-lg border border-line bg-surface p-5">
        <h2 className="font-display text-lg font-bold">Bienvenue dans l'administration</h2>
        <p className="mt-1 text-muted">
          Invitez les personnes de l'association depuis la rubrique « Comptes ». Les autres
          rubriques s'ouvriront au fil des prochaines versions.
        </p>
      </section>
    </Page>
  );
}
