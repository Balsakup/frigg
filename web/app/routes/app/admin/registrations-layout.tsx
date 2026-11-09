import { Outlet } from "react-router";

// Rubrique « Inscriptions » : liste et fiches partagent ce niveau du fil
// d'Ariane.
export const handle = { crumb: "Inscriptions" };

export default function RegistrationsLayout() {
  return <Outlet />;
}
