import { cn } from "~/lib/cn";

// Initiales d'un nom : « Les Sentiers du Val » → « SV ».
export function initials(name: string) {
  const words = name.split(/\s+/).filter((w) => w.length > 2 || /^[A-ZÀ-Ý]/.test(w));
  return (
    words.length >= 2 ? words[0][0] + words[words.length - 1][0] : name.slice(0, 2)
  ).toUpperCase();
}

// Pastille de l'association : son logo, à défaut ses initiales sur la couleur
// principale. Décorative : le nom est toujours écrit à côté (010, FR-023).
export function AssociationMark({
  name,
  logoUrl,
  className,
}: {
  name: string;
  logoUrl?: string | null;
  className?: string;
}) {
  if (logoUrl) {
    return (
      <span
        aria-hidden="true"
        className={cn(
          "grid h-9 w-9 shrink-0 place-items-center overflow-hidden rounded-md border border-line bg-surface",
          className,
        )}
      >
        <img src={logoUrl} alt="" className="h-full w-full object-contain" />
      </span>
    );
  }
  return (
    <span
      aria-hidden="true"
      className={cn(
        "grid h-9 w-9 shrink-0 place-items-center overflow-hidden rounded-md bg-brand font-display text-sm font-bold text-on-brand",
        className,
      )}
    >
      {initials(name)}
    </span>
  );
}
