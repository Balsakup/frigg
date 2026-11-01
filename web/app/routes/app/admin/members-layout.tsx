import { Outlet } from "react-router";

// Rubrique « Membres » : liste, ajout et fiches partagent ce niveau du fil
// d'Ariane.
export const handle = { crumb: "Membres" };

export default function MembersLayout() {
  return <Outlet />;
}
