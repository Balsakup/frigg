// Téléchargement d'un fichier reçu de l'API (CSV des inscrits, modèle et
// lignes rejetées de l'import).
export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
