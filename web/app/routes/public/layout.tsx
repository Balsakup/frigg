import { useQuery } from "@tanstack/react-query";
import { Outlet } from "react-router";
import { fallbackIdentity, identityQuery } from "~/api/identity";
import { AssociationMark } from "~/shell/AssociationMark";

// Gabarit des parcours publics (research.md 016 › R15) : l'habillage des
// écrans hors session (identité visuelle par get-identity, carte centrée,
// sans barre latérale), un peu plus large pour les formulaires de demande.
// Aucune session n'est exigée : les pages lisent seulement l'identité, puis
// les opérations publiques ou par lien personnel.
export default function PublicLayout() {
  const identity = useQuery(identityQuery).data ?? fallbackIdentity;
  return (
    <div className="flex min-h-screen flex-col items-center px-4 py-10 sm:py-16">
      <main id="contenu" className="w-full max-w-[40rem]">
        <p className="mb-6 flex items-center gap-3">
          <AssociationMark
            name={identity.name}
            logoUrl={identity.logoUrl}
            className="h-10 w-10 text-base"
          />
          <span className="min-w-0 font-display text-xl font-bold [overflow-wrap:anywhere]">
            {identity.name}
          </span>
        </p>
        <div className="rounded-lg border border-line bg-surface p-6 sm:p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
