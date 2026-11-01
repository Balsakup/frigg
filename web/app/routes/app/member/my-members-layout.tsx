import { Outlet } from "react-router";

// Rubrique « Mes membres » : liste, ajout et fiches partagent ce niveau du fil
// d'Ariane.
export const handle = { crumb: "Mes membres" };

export default function MyMembersLayout() {
  return <Outlet />;
}
