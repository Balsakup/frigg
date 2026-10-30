import type { FieldValues, Path, UseFormSetError } from "react-hook-form";

// Format d'erreur de l'API (RFC 9457, huma).
export interface Problem {
  status?: number;
  title?: string;
  detail?: string;
  errors?: { location?: string; message?: string }[];
}

export function isProblem(value: unknown): value is Problem {
  return typeof value === "object" && value !== null && ("detail" in value || "title" in value);
}

// Place chaque erreur `body.<champ>` sous le champ correspondant ; le reste
// devient une erreur globale du formulaire (constitution, principe VI).
export function applyProblemToForm<T extends FieldValues>(
  problem: Problem,
  setError: UseFormSetError<T>,
  fields: readonly Path<T>[],
) {
  const byField = new Map<Path<T>, string[]>();
  const general: string[] = [];
  for (const e of problem.errors ?? []) {
    const field = e.location?.startsWith("body.") ? (e.location.slice(5) as Path<T>) : undefined;
    if (field && fields.includes(field)) {
      byField.set(field, [...(byField.get(field) ?? []), e.message ?? ""]);
    } else if (e.message) {
      general.push(e.message);
    }
  }
  for (const [field, messages] of byField) {
    setError(field, { type: "server", message: messages.join(" ") });
  }
  if (byField.size === 0) {
    setError("root.server" as Path<T>, {
      type: "server",
      message: general.join(" ") || problem.detail || "Une erreur inattendue s'est produite.",
    });
  }
}
