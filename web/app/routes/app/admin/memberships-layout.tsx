import { Outlet } from "react-router";

// Rubrique « Adhésions » (administration) : liste, création et détail partagent ce niveau du fil
// d'Ariane.
export const handle = { crumb: "Adhésions" };

export default function AdminMembershipsLayout() {
  return <Outlet />;
}
