import { Shield, User } from "lucide-react";
import { Link } from "react-router";
import { cn } from "~/lib/cn";
import { writePreference } from "~/lib/storage";
import { Icon } from "~/ui/Icon";
import type { Space } from "./navigation";

interface SpaceSwitcherProps {
  space: Space;
  collapsed: boolean;
  onNavigate?: () => void;
}

const spaces: { id: Space; label: string; to: string; icon: typeof Shield }[] = [
  { id: "admin", label: "Admin", to: "/admin", icon: Shield },
  { id: "member", label: "Membre", to: "/", icon: User },
];

// Sélecteur d'espace des administrateurs (FR-019a) : l'état actif suit l'URL.
export function SpaceSwitcher({ space, collapsed, onNavigate }: SpaceSwitcherProps) {
  return (
    <nav aria-label="Changer d'espace" className="shrink-0 border-b border-line px-3 py-3">
      <ul
        className={cn(
          "grid gap-1 rounded-lg bg-canvas p-1",
          collapsed ? "grid-cols-1" : "grid-cols-2",
        )}
      >
        {spaces.map((s) => {
          const active = s.id === space;
          return (
            <li key={s.id}>
              <Link
                to={s.to}
                aria-current={active ? "true" : undefined}
                onClick={() => {
                  writePreference("last-space", s.id);
                  onNavigate?.();
                }}
                className={cn(
                  "flex h-9 items-center justify-center gap-2 rounded-md text-sm font-bold",
                  active
                    ? "bg-surface text-brand-ink shadow-sm ring-1 ring-line"
                    : "text-muted hover:text-ink",
                )}
              >
                <Icon icon={s.icon} className="h-4 w-4" />
                <span className={collapsed ? "sr-only" : undefined}>
                  <span className="sr-only">Espace </span>
                  {s.label}
                </span>
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
