import { Outlet } from "react-router";

// Rubrique « Adhésions » : liste, demande et détail partagent ce niveau du fil
// d'Ariane.
export const handle = { crumb: "Adhésions" };

export default function MembershipsLayout() {
  return <Outlet />;
}
