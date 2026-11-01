import { Outlet } from "react-router";

// Rubrique « Paramètres » : chaque section est une sous-page avec sa propre
// adresse (SettingsPage).
export const handle = { crumb: "Paramètres" };

export default function SettingsLayout() {
  return <Outlet />;
}
