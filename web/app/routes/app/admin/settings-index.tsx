import { redirect } from "react-router";

// « Paramètres » mène à sa première section, « Général » (FR-001 de 010).
export function clientLoader() {
  return redirect("/admin/parametres/general");
}

export default function SettingsIndex() {
  return null;
}
