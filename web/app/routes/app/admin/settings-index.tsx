import { redirect } from "react-router";

// « Paramètres » mène à sa première section. Quand les sections seront
// nombreuses, cette page pourra devenir un sommaire.
export function clientLoader() {
  return redirect("/admin/parametres/paiements");
}

export default function SettingsIndex() {
  return null;
}
