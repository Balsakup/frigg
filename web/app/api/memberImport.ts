import { client } from "./client";
import type { Problem } from "./problem";
import type { components } from "./schema";

export type ImportAnalysis = components["schemas"]["ImportAnalysis"];
export type ImportPreview = components["schemas"]["ImportPreview"];
export type ImportReport = components["schemas"]["ImportReport"];
export type ImportRow = components["schemas"]["ImportRow"];
export type ImportTargetOption = components["schemas"]["ImportTargetOption"];

// Le fichier reste dans le navigateur et accompagne chaque étape : le serveur
// n'en garde rien (research.md 013 › R1).
function form(file: File, fields: Record<string, string> = {}) {
  const f = new FormData();
  for (const [k, v] of Object.entries(fields)) f.append(k, v);
  f.append("file", file, file.name);
  return f;
}

function fail(error: unknown, response: Response): never {
  throw { ...(error as Problem), status: response.status } satisfies Problem;
}

// Le corps est construit par bodySerializer ; le type de l'objet passé à
// openapi-fetch est sans importance.
type AnyBody = never;

export async function analyzeImport(file: File) {
  const { data, error, response } = await client.POST("/admin/members/import/analyze", {
    body: {} as AnyBody,
    bodySerializer: () => form(file),
  });
  if (data) return data;
  fail(error, response);
}

export async function previewImport(file: File, mapping: string[]) {
  const { data, error, response } = await client.POST("/admin/members/import/preview", {
    body: {} as AnyBody,
    bodySerializer: () => form(file, { mapping: JSON.stringify(mapping) }),
  });
  if (data) return data;
  fail(error, response);
}

export async function runImport(
  file: File,
  mapping: string[],
  fieldsVersion: string,
  expectedReady: number,
) {
  const { data, error, response } = await client.POST("/admin/members/import", {
    body: {} as AnyBody,
    bodySerializer: () =>
      form(file, {
        mapping: JSON.stringify(mapping),
        fieldsVersion,
        expectedReady: String(expectedReady),
      }),
  });
  if (data) return data;
  fail(error, response);
}

export async function rejectedImport(file: File, mapping: string[]): Promise<Blob> {
  const { data, error, response } = await client.POST("/admin/members/import/rejected", {
    body: {} as AnyBody,
    bodySerializer: () => form(file, { mapping: JSON.stringify(mapping) }),
    parseAs: "blob",
  });
  if (data) return data as Blob;
  fail(error, response);
}

export async function downloadTemplate(): Promise<Blob> {
  const { data, error, response } = await client.GET("/admin/members/import/template", {
    parseAs: "blob",
  });
  if (data) return data as Blob;
  fail(error, response);
}

// Erreurs d'une réponse 422 à un emplacement donné (« form.file »).
export function errorsAt(problem: Problem | null, location: string): string[] {
  return (problem?.errors ?? [])
    .filter((e) => e.location === location)
    .map((e) => e.message ?? "")
    .filter(Boolean);
}
