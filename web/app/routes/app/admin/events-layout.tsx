import { Outlet } from "react-router";

// Rubrique « Événements » : liste, création et fiches partagent ce niveau du
// fil d'Ariane.
export const handle = { crumb: "Événements" };

export default function EventsLayout() {
  return <Outlet />;
}
