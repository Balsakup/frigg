const dateFormat = new Intl.DateTimeFormat("fr-FR", { dateStyle: "long" });

// Date lisible en français, par exemple « 6 novembre 2026 ».
export function formatDate(value: string | Date) {
  return dateFormat.format(typeof value === "string" ? new Date(value) : value);
}

export function roleLabel(role: "member" | "admin") {
  return role === "admin" ? "Administrateur" : "Membre";
}
