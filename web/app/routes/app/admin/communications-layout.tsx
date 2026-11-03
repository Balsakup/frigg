import { Outlet } from "react-router";

// Rubrique « Communication » (administration) : historique, préparation et
// détail partagent ce niveau du fil d'Ariane.
export const handle = { crumb: "Communication" };

export default function CommunicationsLayout() {
  return <Outlet />;
}
