import { Outlet } from "react-router";

// Rubrique « Saisons » : liste, création et fiches partagent ce niveau du fil
// d'Ariane.
export const handle = { crumb: "Saisons" };

export default function SeasonsLayout() {
  return <Outlet />;
}
