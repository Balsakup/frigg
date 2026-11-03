import type { QueryClient } from "@tanstack/react-query";
import { client } from "./client";
import type { Problem } from "./problem";
import { $api } from "./queries";
import type { components } from "./schema";

export type PaymentSettings = components["schemas"]["PaymentSettings"];

// Moyens de paiement proposés aux adhérents (006, US5).
export function paymentSettingsQuery() {
  return $api.queryOptions("get", "/admin/settings/payment");
}

export async function invalidatePaymentSettings(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: ["get", "/admin/settings/payment"] }),
    qc.invalidateQueries({ queryKey: ["get", "/admin/memberships/dashboard"] }),
    qc.invalidateQueries({ queryKey: ["get", "/me/memberships/{id}"] }),
  ]);
}

export type GeneralSettings = components["schemas"]["GeneralSettings"];

// Paramètres généraux de l'association (010, section « Général »).
export function generalSettingsQuery() {
  return $api.queryOptions("get", "/admin/settings/general");
}

// Après un enregistrement, le nom et la tagline changent partout (FR-027).
export async function invalidateGeneralSettings(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: ["get", "/admin/settings/general"] }),
    qc.invalidateQueries({ queryKey: ["get", "/session"] }),
    qc.invalidateQueries({ queryKey: ["get", "/identity"] }),
    qc.invalidateQueries({ queryKey: ["get", "/association/contact"] }),
  ]);
}

export type Appearance = components["schemas"]["Appearance"];

// Couleurs et logo de l'association (010, section « Apparence »).
export function appearanceQuery() {
  return $api.queryOptions("get", "/admin/settings/appearance");
}

// Palette dérivée de deux couleurs, calculée par le serveur (contrastes).
export function palettePreviewQuery(primary: string, accent: string) {
  return $api.queryOptions("get", "/admin/settings/appearance/palette", {
    params: { query: { primary, accent } },
  });
}

// L'identité visuelle change partout : écrans hors session, barre latérale,
// couleurs (FR-027).
export async function invalidateAppearance(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: ["get", "/admin/settings/appearance"] }),
    qc.invalidateQueries({ queryKey: ["get", "/identity"] }),
    qc.invalidateQueries({ queryKey: ["get", "/session"] }),
  ]);
}

// Message affiché sur le champ du logo pour un fichier trop gros, y compris
// quand le serveur refuse la requête entière (413).
export const LOGO_FORMAT_MESSAGE = "Déposez une image PNG, JPEG ou WebP de 1 Mo au plus.";

// Dépôt du logo : multipart, champ « file ».
export async function uploadLogo(file: File): Promise<Appearance> {
  const { data, error, response } = await client.PUT("/admin/settings/appearance/logo", {
    body: { file: file as unknown as string },
    bodySerializer: () => {
      const form = new FormData();
      form.append("file", file, file.name);
      return form;
    },
  });
  if (data) return data;
  if (response.status === 413) {
    throw {
      status: 413,
      detail: LOGO_FORMAT_MESSAGE,
      errors: [{ location: "form.file", message: LOGO_FORMAT_MESSAGE }],
    } satisfies Problem;
  }
  throw { ...(error as Problem), status: response.status } satisfies Problem;
}
