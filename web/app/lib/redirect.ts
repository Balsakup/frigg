// N'accepte qu'un chemin interne à l'application comme destination après
// connexion (FR-004) : pas d'URL absolue ni de « //domaine ».
export function safeRedirect(value: string | null | undefined): string | null {
  if (!value?.startsWith("/") || value.startsWith("//") || value.startsWith("/\\")) {
    return null;
  }
  return value;
}
