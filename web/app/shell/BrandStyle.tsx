import { useQuery } from "@tanstack/react-query";
import { useLayoutEffect } from "react";
import { identityQuery, type Palette } from "~/api/identity";

// Code « #rrggbb » → triplet « r g b » des variables CSS de app.css.
export function triplet(hex: string) {
  const n = Number.parseInt(hex.slice(1), 16);
  return `${(n >> 16) & 255} ${(n >> 8) & 255} ${n & 255}`;
}

// Variables de marque écrites par BrandStyle ; les couleurs sémantiques
// (--ok, --warn, --danger, --info) ne sont jamais touchées (FR-022).
export const brandVariables = {
  "--brand": "brand",
  "--brand-ink": "brandInk",
  "--on-brand": "onBrand",
  "--accent": "accent",
  "--accent-ink": "accentInk",
  "--on-accent": "onAccent",
} as const satisfies Record<string, keyof Palette>;

export function paletteStyle(palette: Palette): Record<string, string> {
  return Object.fromEntries(
    Object.entries(brandVariables).map(([name, key]) => [name, triplet(palette[key])]),
  );
}

// Applique la palette de l'association à toute l'application (010 › R5). La
// palette est calculée par le serveur, qui garantit les contrastes.
export function BrandStyle() {
  const { data } = useQuery(identityQuery);
  useLayoutEffect(() => {
    if (!data) return;
    const root = document.documentElement;
    for (const [name, value] of Object.entries(paletteStyle(data.palette))) {
      root.style.setProperty(name, value);
    }
  }, [data]);
  return null;
}
