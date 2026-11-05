import type { ReactNode } from "react";
import { NavLink } from "react-router";
import { cn } from "~/lib/cn";
import { Page } from "~/shell/Page";

// Sections des paramètres, dans l'ordre d'affichage. Ajouter une section :
// une entrée ici et une route sous « parametres » dans routes.ts.
export const settingsSections = [
  { path: "/admin/parametres/general", label: "Général" },
  { path: "/admin/parametres/apparence", label: "Apparence" },
  { path: "/admin/parametres/champs", label: "Champs personnalisés" },
  { path: "/admin/parametres/paiements", label: "Paiements" },
];

// Gabarit commun des sous-pages des paramètres : en-tête « Paramètres », liste
// de liens vers les sections (aria-current sur la section affichée), puis le
// contenu. Des liens et non des onglets ARIA : chaque section est une page.
export function SettingsPage({ section, children }: { section: string; children: ReactNode }) {
  return (
    <Page
      title="Paramètres"
      documentTitle={`${section} · Paramètres`}
      intro="Réglages de l'association."
    >
      <div className="flex flex-col gap-6 lg:flex-row lg:items-start">
        <nav aria-label="Sections des paramètres" className="min-w-0 lg:w-56 lg:shrink-0">
          <ul className="-mx-1 flex gap-1 overflow-x-auto px-1 pb-1 lg:mx-0 lg:flex-col lg:overflow-visible lg:px-0 lg:pb-0">
            {settingsSections.map((s) => (
              <li key={s.path} className="shrink-0">
                <NavLink
                  to={s.path}
                  className={({ isActive }) =>
                    cn(
                      "flex min-h-10 items-center whitespace-nowrap rounded-md px-3 py-2",
                      isActive
                        ? "bg-brand/10 font-bold text-brand-ink"
                        : "text-ink hover:bg-canvas",
                    )
                  }
                >
                  {s.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
        <div className="min-w-0 flex-1">{children}</div>
      </div>
    </Page>
  );
}
