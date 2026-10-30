// Liens d'évitement (RGAA 12.7), visibles au focus.
export function SkipLinks() {
  const cls =
    "sr-only focus:not-sr-only focus:inline-block rounded bg-ink px-4 py-2 font-bold text-white";
  return (
    <nav aria-label="Accès rapide" className="fixed left-2 top-2 z-[70]">
      <ul className="flex flex-col gap-1">
        <li>
          <a href="#contenu" className={cls}>
            Aller au contenu
          </a>
        </li>
        <li>
          <a href="#nav-principale" className={cls}>
            Aller à la navigation
          </a>
        </li>
      </ul>
    </nav>
  );
}
