import { Page } from "~/shell/Page";
import { useSession } from "~/shell/useSession";

export const handle = { crumb: "Accueil" };

// Accueil de l'espace membre ; les activités et actualités viendront avec les
// fonctionnalités correspondantes.
export default function MemberHome() {
  const { association } = useSession();
  return (
    <Page title="Accueil" intro="Vos prochaines activités et les actualités de l'association.">
      <section className="max-w-prose rounded-lg border border-line bg-surface p-5">
        <h2 className="font-display text-lg font-bold">Bienvenue</h2>
        <p className="mt-1 text-muted">
          Vous êtes dans l'espace membre de {association.name}. Les rubriques s'ouvriront au fil des
          prochaines versions de Frigg.
        </p>
      </section>
    </Page>
  );
}
