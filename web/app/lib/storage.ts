// Préférences propres à l'appareil (barre latérale repliée, dernier espace).
// localStorage peut être indisponible (navigation privée, stockage bloqué) :
// chaque accès est protégé et l'interface fonctionne sans.

export function readPreference(key: string): string | null {
  try {
    return window.localStorage.getItem(`frigg:${key}`);
  } catch {
    return null;
  }
}

export function writePreference(key: string, value: string) {
  try {
    window.localStorage.setItem(`frigg:${key}`, value);
  } catch {
    // préférence non mémorisée, sans conséquence
  }
}
