import { type ComponentProps, useId } from "react";

// Champ piège des formulaires publics (research.md 016 › R10) : hors écran,
// retiré de l'arbre d'accessibilité et de l'ordre de tabulation. Une personne
// ne le voit ni ne l'atteint ; un robot qui le remplit voit sa demande ignorée
// sans le savoir.
export function Honeypot(props: Omit<ComponentProps<"input">, "name" | "type" | "id">) {
  const id = useId();
  return (
    <div aria-hidden="true" className="absolute -left-[10000px] top-auto h-px w-px overflow-hidden">
      <label htmlFor={id}>Site web</label>
      <input id={id} type="text" name="website" tabIndex={-1} autoComplete="off" {...props} />
    </div>
  );
}
