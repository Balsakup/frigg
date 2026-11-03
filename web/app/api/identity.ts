import { queryOptions } from "@tanstack/react-query";
import { client } from "./client";
import { $api } from "./queries";
import type { components } from "./schema";

export type Identity = components["schemas"]["Identity"];
export type Palette = components["schemas"]["Palette"];
export type AssociationContact = components["schemas"]["AssociationContact"];

// Identité de Frigg, utilisée si celle de l'association ne peut pas être lue :
// l'écran reste utilisable (FR-028 de 010). Valeurs de brand.Default (Go).
export const fallbackIdentity: Identity = {
  name: "Frigg",
  logoUrl: null,
  palette: {
    brand: "#1f5278",
    brandInk: "#1f5278",
    onBrand: "#ffffff",
    accent: "#c44822",
    accentInk: "#b63b12",
    onAccent: "#ffffff",
  },
};

export function isFallbackIdentity(identity: Identity) {
  return identity === fallbackIdentity;
}

// Identité visuelle publique (nom, logo, couleurs) : préchargée par root avant
// le premier rendu, dans la session comme hors session (010 › R5).
export const identityQuery = queryOptions({
  queryKey: ["get", "/identity"],
  queryFn: async (): Promise<Identity> => {
    try {
      const { data } = await client.GET("/identity");
      return data ?? fallbackIdentity;
    } catch {
      return fallbackIdentity;
    }
  },
  staleTime: 5 * 60_000,
});

// Coordonnées et réseaux sociaux présentés aux membres (010, US4).
export function associationContactQuery() {
  return $api.queryOptions("get", "/association/contact");
}
