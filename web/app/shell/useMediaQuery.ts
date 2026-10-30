import { useSyncExternalStore } from "react";

// État d'une media query, mis à jour au redimensionnement.
export function useMediaQuery(query: string, serverValue = true) {
  return useSyncExternalStore(
    (onChange) => {
      const mql = window.matchMedia(query);
      mql.addEventListener("change", onChange);
      return () => mql.removeEventListener("change", onChange);
    },
    () => window.matchMedia(query).matches,
    () => serverValue,
  );
}
