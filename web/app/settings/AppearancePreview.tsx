import type { CSSProperties } from "react";
import type { Palette } from "~/api/identity";
import { AssociationMark } from "~/shell/AssociationMark";
import { triplet } from "~/shell/BrandStyle";

// Variables de couleur limitées au conteneur de l'aperçu : les jetons Tailwind
// (--color-*) sont redéfinis ici, sans toucher au reste de l'interface (FR-019).
function scopedStyle(p: Palette): CSSProperties {
  const vars: Record<string, string> = {};
  for (const [name, hex] of [
    ["brand", p.brand],
    ["brand-ink", p.brandInk],
    ["on-brand", p.onBrand],
    ["accent", p.accent],
    ["accent-ink", p.accentInk],
    ["on-accent", p.onAccent],
  ] as const) {
    vars[`--${name}`] = triplet(hex);
    vars[`--color-${name}`] = `rgb(${triplet(hex)})`;
  }
  return vars as CSSProperties;
}

// Aperçu non interactif de l'identité visuelle (contracts/ui-routes.md 010).
export function AppearancePreview({
  name,
  tagline,
  logoUrl,
  palette,
}: {
  name: string;
  tagline?: string | null;
  logoUrl?: string | null;
  palette: Palette;
}) {
  const adjusted = palette.brandInk !== palette.brand || palette.accentInk !== palette.accent;
  return (
    <section
      aria-label="Aperçu"
      style={scopedStyle(palette)}
      className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5"
    >
      <h2 className="font-display text-lg font-bold">Aperçu</h2>
      <div aria-hidden="true" className="flex flex-col gap-4">
        <div className="rounded-md border border-line bg-surface">
          <div className="flex items-center gap-3 border-b border-line p-3">
            <AssociationMark name={name} logoUrl={logoUrl} />
            <span className="min-w-0 leading-tight">
              <span className="block truncate font-display text-[15px] font-bold">{name}</span>
              {tagline ? <span className="block truncate text-xs">{tagline}</span> : null}
            </span>
          </div>
          <div className="p-2">
            <span className="flex min-h-10 items-center rounded-md bg-brand/10 px-3 font-bold text-brand-ink">
              Accueil
            </span>
            <span className="flex min-h-10 items-center px-3">Adhésions</span>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <span className="inline-flex min-h-10 items-center rounded-md bg-brand px-4 font-bold text-on-brand">
            Bouton principal
          </span>
          <span className="inline-flex min-h-10 items-center rounded-md bg-accent px-4 font-bold text-on-accent">
            Mise en avant
          </span>
          <span className="font-bold text-brand-ink underline underline-offset-2">Un lien</span>
        </div>
        <div className="overflow-hidden rounded-md border border-line">
          <div className="flex items-center gap-2 bg-brand px-3 py-2 font-bold text-on-brand">
            {logoUrl ? (
              <img src={logoUrl} alt="" className="h-6 w-auto rounded-sm bg-surface" />
            ) : null}
            <span className="truncate">{name}</span>
          </div>
          <p className="p-3 text-sm text-muted">En-tête des e-mails envoyés aux membres.</p>
        </div>
      </div>
      <p role="status" className="text-sm text-muted">
        {adjusted ? "Certains textes ont été assombris pour rester lisibles." : ""}
      </p>
    </section>
  );
}
