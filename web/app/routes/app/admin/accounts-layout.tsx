import { Outlet } from "react-router";

// Rubrique « Comptes » : liste et fiches partagent ce niveau du fil d'Ariane.
export const handle = { crumb: "Comptes" };

export default function AccountsLayout() {
  return <Outlet />;
}
