import { Outlet } from "react-router";

// Gabarit des écrans publics : centré, sans barre latérale. Le nom de
// l'association n'est pas affiché hors connexion (principe II).
export default function AuthLayout() {
  return (
    <div className="flex min-h-screen flex-col items-center px-4 py-10 sm:py-16">
      <main id="contenu" className="w-full max-w-[30rem]">
        <p className="mb-6 flex items-center gap-3">
          <span
            aria-hidden="true"
            className="grid h-10 w-10 place-items-center rounded-md bg-brand font-display font-bold text-on-brand"
          >
            F
          </span>
          <span className="font-display text-xl font-bold">Frigg</span>
        </p>
        <div className="rounded-lg border border-line bg-surface p-6 sm:p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
