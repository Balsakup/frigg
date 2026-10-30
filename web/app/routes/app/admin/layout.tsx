import { Outlet } from "react-router";
import { AccessDenied } from "~/shell/AccessDenied";
import { useSession } from "~/shell/useSession";

export const handle = { crumb: "Tableau de bord" };

// Espace administration : « Accès refusé » pour un membre (FR-023). Le serveur
// refuse de toute façon les données (principe II).
export default function AdminLayout() {
  const { account } = useSession();
  return account.role === "admin" ? <Outlet /> : <AccessDenied />;
}
