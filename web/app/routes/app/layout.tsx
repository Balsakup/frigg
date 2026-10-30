import { Outlet, redirect } from "react-router";
import { sessionQuery } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { AppShell } from "~/shell/AppShell";
import type { Route } from "./+types/layout";

// Toute page du shell exige une session (principe II) ; sinon, connexion puis
// retour sur la page demandée (FR-004).
export async function clientLoader({ request }: Route.ClientLoaderArgs) {
  try {
    return await queryClient.ensureQueryData(sessionQuery);
  } catch {
    const url = new URL(request.url);
    throw redirect(`/connexion?suite=${encodeURIComponent(url.pathname + url.search)}`);
  }
}

export default function AppLayout() {
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  );
}
