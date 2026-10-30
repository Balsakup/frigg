import { readPreference } from "./storage";

// Accueil par défaut après connexion : l'espace administration pour un
// administrateur, sauf s'il a choisi l'espace membre en dernier sur cet appareil.
export function homeFor(role: "member" | "admin"): string {
  if (role === "admin" && readPreference("last-space") !== "member") {
    return "/admin";
  }
  return "/";
}
