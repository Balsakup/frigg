import { useEffect, useState } from "react";

// Jeton reçu par e-mail, placé dans le fragment de l'URL (jamais envoyé au
// serveur ni dans l'en-tête Referer). Il est retiré de la barre d'adresse dès
// la lecture.
export function useFragmentToken() {
  const [token] = useState(() =>
    typeof window === "undefined" ? "" : window.location.hash.slice(1),
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
