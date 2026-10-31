import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
import { queryClient } from "~/api/query-client";
import { type SeasonList, seasonListQuery } from "~/api/seasons";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { buttonClasses } from "~/ui/Button";

// Préchargement ; en cas d'erreur, le tableau de bord s'affiche sans encart.
export async function clientLoader() {
  await queryClient.ensureQueryData(seasonListQuery()).catch(() => undefined);
  return null;
}

// Tableau de bord de l'espace administration ; les indicateurs viendront avec
// les fonctionnalités métier.
export default function AdminHome() {
  const seasons = useQuery(seasonListQuery());
  return (
    <Page title="Tableau de bord" intro="Vue d'ensemble de l'activité de l'association.">
      {seasons.data ? <SeasonSetupAlert list={seasons.data} /> : null}
      <section className="max-w-prose rounded-lg border border-line bg-surface p-5">
        <h2 className="font-display text-lg font-bold">Bienvenue dans l'administration</h2>
        <p className="mt-1 text-muted">
          Créez les saisons de l'association depuis la rubrique « Saisons » et invitez les personnes
          depuis « Comptes ». Les autres rubriques s'ouvriront au fil des prochaines versions.
        </p>
      </section>
    </Page>
  );
}

// Encart de mise en place des saisons (FR-023), piloté par l'API.
function SeasonSetupAlert({ list }: { list: SeasonList }) {
  const link = (to: string, label: string) => (
    <Link to={to} className={buttonClasses({ variant: "secondary", size: "sm" })}>
      {label}
    </Link>
  );
  switch (list.setup) {
    case "missing":
      return (
        <Alert
          kind="info"
          className="mb-6 max-w-prose"
          actions={link("/admin/saisons/nouvelle", "Créer la saison")}
        >
          Créez la première saison pour mettre en place Frigg.
        </Alert>
      );
    case "no-current":
      return (
        <Alert
          kind="info"
          className="mb-6 max-w-prose"
          actions={link("/admin/saisons/nouvelle", "Créer la saison suivante")}
        >
          Aucune saison n'est en cours.
        </Alert>
      );
    case "current-draft": {
      const current = (list.items ?? []).find((s) => s.id === list.currentId);
      if (!current) return null;
      return (
        <Alert
          kind="warning"
          className="mb-6 max-w-prose"
          actions={link(`/admin/saisons/${current.id}`, "Voir la saison")}
        >
          La saison {current.label} a commencé mais n'est pas encore ouverte.
        </Alert>
      );
    }
    default:
      return null;
  }
}
