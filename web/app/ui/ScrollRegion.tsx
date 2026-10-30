import type { ReactNode } from "react";

// Zone à défilement horizontal atteignable au clavier (tableaux sur petit écran).
export function ScrollRegion({
  labelledBy,
  children,
}: {
  labelledBy: string;
  children: ReactNode;
}) {
  return (
    // biome-ignore lint/a11y/noNoninteractiveTabindex: la zone doit pouvoir défiler au clavier
    <section aria-labelledby={labelledBy} tabIndex={0} className="overflow-x-auto">
      {children}
    </section>
  );
}
