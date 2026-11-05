import type { QueryClient } from "@tanstack/react-query";
import { $api } from "./queries";
import type { components } from "./schema";

export type FieldDefinition = components["schemas"]["FieldDefinition"];
export type CustomFieldList = components["schemas"]["CustomFieldList"];
export type FormField = components["schemas"]["FormField"];
export type FieldValue = components["schemas"]["FieldValue"];
export type FieldInput = components["schemas"]["FieldInput"];
export type FieldType = FieldDefinition["type"];
export type FieldVisibility = FieldDefinition["visibility"];
export type FieldScope = FieldDefinition["scope"];

// Définitions des champs personnalisés, avec leurs compteurs (011, US1).
export function customFieldsQuery() {
  return $api.queryOptions("get", "/admin/custom-fields");
}

// Champs des formulaires de l'espace membre (sans les champs réservés).
export function myCustomFieldsQuery() {
  return $api.queryOptions("get", "/me/custom-fields");
}

// Après une écriture de définition : section, formulaires des deux espaces
// et fiches ou adhésions affichées (libellés, choix).
export async function invalidateCustomFields(qc: QueryClient) {
  await Promise.all([
    qc.invalidateQueries({ queryKey: ["get", "/admin/custom-fields"] }),
    qc.invalidateQueries({ queryKey: ["get", "/me/custom-fields"] }),
    qc.invalidateQueries({ queryKey: ["get", "/admin/members/{id}"] }),
    qc.invalidateQueries({ queryKey: ["get", "/me/members/{id}"] }),
    qc.invalidateQueries({ queryKey: ["get", "/admin/memberships/{id}"] }),
    qc.invalidateQueries({ queryKey: ["get", "/me/memberships/{id}"] }),
  ]);
}

// Après une écriture de valeurs : compteurs de la section.
export async function invalidateCustomFieldCounters(qc: QueryClient) {
  await qc.invalidateQueries({ queryKey: ["get", "/admin/custom-fields"] });
}

export const typeLabels: Record<FieldType, string> = {
  short_text: "Texte court",
  long_text: "Texte long",
  number: "Nombre",
  date: "Date",
  bool: "Oui/non",
  choice: "Liste de choix",
};

export const visibilityLabels: Record<FieldVisibility, string> = {
  admin: "Réservé à l'administration",
  visible: "Visible par le membre",
  editable: "Rempli par le membre",
};

export const visibilityDescriptions: Record<FieldVisibility, string> = {
  admin: "Ni affiché ni modifiable depuis l'espace membre.",
  visible: "Affiché en lecture seule au compte qui gère la fiche ; rempli par le bureau.",
  editable: "Affiché et modifiable par le compte qui gère la fiche.",
};
