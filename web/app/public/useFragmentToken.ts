import { useEffect, useState } from "react";

// Lit le jeton d'un lien d'e-mail public (`#jeton=…`) au premier rendu, puis
// l'efface aussitôt de la barre d'adresse et de l'historique : le jeton n'est
// jamais envoyé au serveur dans une URL (research.md 016 › R1). Chemin et
// requête sont conservés (retour de HelloAsso, par exemple).
export function readFragmentToken(hash: string): string | null {
  const token = new URLSearchParams(hash.replace(/^#/, "")).get("jeton");
  return token ? token : null;
}

export function useFragmentToken(): string | null {
  const [token] = useState(() =>
    typeof window === "undefined" ? null : readFragmentToken(window.location.hash),
  );
  useEffect(() => {
    if (window.location.hash) {
      window.history.replaceState(
        window.history.state,
        "",
        window.location.pathname + window.location.search,
      );
    }
  }, []);
  return token;
}
