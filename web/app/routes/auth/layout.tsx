import { useQuery } from "@tanstack/react-query";
import { Outlet } from "react-router";
import { fallbackIdentity, identityQuery } from "~/api/identity";
import { AssociationMark } from "~/shell/AssociationMark";

// Gabarit des écrans publics : centré, sans barre latérale. Il affiche
// l'identité visuelle de l'association (nom, logo, couleurs), seule donnée de
// l'association lisible sans session (principe II, amendement 1.8.0).
export default function AuthLayout() {
  const identity = useQuery(identityQuery).data ?? fallbackIdentity;
  return (
    <div className="flex min-h-screen flex-col items-center px-4 py-10 sm:py-16">
      <main id="contenu" className="w-full max-w-[30rem]">
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
