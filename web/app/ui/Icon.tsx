import type { LucideIcon } from "lucide-react";
import { cn } from "~/lib/cn";

interface IconProps {
  icon: LucideIcon;
  className?: string;
  /** Nom accessible ; sans lui l'icône est décorative (aria-hidden). */
  label?: string;
}

// Icônes lucide au tracé de l'artefact (1,8), décoratives par défaut.
export function Icon({ icon: Glyph, className, label }: IconProps) {
  return (
    <Glyph
      strokeWidth={1.8}
      className={cn("block h-5 w-5 shrink-0", className)}
      aria-hidden={label ? undefined : true}
      aria-label={label}
      focusable="false"
      role={label ? "img" : undefined}
    />
  );
}
