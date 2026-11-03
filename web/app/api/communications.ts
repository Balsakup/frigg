import type { QueryClient } from "@tanstack/react-query";
import { client } from "./client";
import { isProblem, type Problem } from "./problem";
import { $api } from "./queries";
import type { components } from "./schema";

export type Audience = components["schemas"]["Audience"];
export type AudienceMember = components["schemas"]["AudienceMember"];
export type Attachment = components["schemas"]["Attachment"];
export type Communication = components["schemas"]["Communication"];
export type CommunicationSummary = components["schemas"]["CommunicationSummary"];
export type Delivery = components["schemas"]["Delivery"];
export type Situation = AudienceMember["situation"];
export type Preset = Audience["preset"];

export interface TargetInput {
  seasonId: number;
  situations: Situation[];
  feeIds: number[];
}

// Cible normalisée : la clé de requête ne dépend pas de l'ordre des cases.
export function normalizeTarget(t: TargetInput): TargetInput {
  return {
    seasonId: t.seasonId,
    situations: [...new Set(t.situations)].sort(),
    feeIds: [...new Set(t.feeIds)].sort((a, b) => a - b),
  };
}

export function audienceQuery(target: TargetInput) {
  const t = normalizeTarget(target);
  return $api.queryOptions("get", "/admin/communications/audience", {
    params: { query: { seasonId: t.seasonId, situations: t.situations, feeIds: t.feeIds } },
  });
}

export function communicationQuery(id: number) {
  return $api.queryOptions("get", "/admin/communications/{id}", { params: { path: { id } } });
}

export function communicationsQuery(page: number) {
  return $api.queryOptions("get", "/admin/communications", { params: { query: { page } } });
}

export async function invalidateCommunications(qc: QueryClient) {
  await qc.invalidateQueries({ queryKey: ["get", "/admin/communications"] });
}

// Réponse 409 de l'envoi : la cible recalculée accompagne le problème.
export function changedAudience(error: unknown): Audience | null {
  if (!isProblem(error) || error.status !== 409) return null;
  return (error as Problem & { audience?: Audience }).audience ?? null;
}

// Dépôt d'une pièce jointe : multipart, un fichier par requête.
export async function uploadAttachment(draftKey: string, file: File) {
  const { data, error, response } = await client.POST("/admin/communications/attachments", {
    body: { draftKey, file: file as unknown as string },
    bodySerializer: () => {
      const form = new FormData();
      form.append("draftKey", draftKey);
      form.append("file", file, file.name);
      return form;
    },
  });
  if (data) return data;
  throw { ...(error as Problem), status: response.status } satisfies Problem;
}

export async function deleteAttachment(id: number) {
  const { error, response } = await client.DELETE("/admin/communications/attachments/{id}", {
    params: { path: { id } },
  });
  if (error) throw { ...(error as Problem), status: response.status } satisfies Problem;
}

export function attachmentDownloadHref(communicationId: number, attachmentId: number) {
  return `/api/admin/communications/${communicationId}/attachments/${attachmentId}`;
}

export const situationLabels: Record<Situation, string> = {
  up_to_date: "À jour",
  announced: "Paiement annoncé",
  due: "À payer",
};

export const presetOptions: {
  value: Exclude<Preset, "custom">;
  label: string;
  situations: Situation[];
}[] = [
  { value: "all", label: "Tous les adhérents", situations: ["up_to_date", "announced", "due"] },
  { value: "up_to_date", label: "À jour", situations: ["up_to_date"] },
  { value: "unpaid", label: "En impayé", situations: ["announced", "due"] },
];
