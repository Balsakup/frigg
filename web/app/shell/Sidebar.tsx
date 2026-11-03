import { Dialog } from "@base-ui/react/dialog";
import { Tooltip } from "@base-ui/react/tooltip";
import { X } from "lucide-react";
import { Link, NavLink, useLocation } from "react-router";
import { cn } from "~/lib/cn";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { AssociationMark } from "./AssociationMark";
import { entriesFor, type NavEntry, spaceFromPath } from "./navigation";
import { SpaceSwitcher } from "./SpaceSwitcher";
import { UserMenu } from "./UserMenu";
import { useSession } from "./useSession";

interface SidebarProps {
  desktop: boolean;
  collapsed: boolean;
  mobileOpen: boolean;
  onMobileOpenChange: (open: boolean) => void;
}

// Barre latérale : fixe et repliable en icônes sur grand écran (FR-020), tiroir
// en surimpression sur petit écran (fermé par le bouton, Échap ou clic extérieur).
export function Sidebar({ desktop, collapsed, mobileOpen, onMobileOpenChange }: SidebarProps) {
  if (desktop) {
    return (
      <aside
        id="sidebar"
        aria-label="Barre latérale"
        style={{ width: collapsed ? "var(--sidebar-w-collapsed)" : "var(--sidebar-w)" }}
        className="fixed inset-y-0 left-0 z-40 flex flex-col border-r border-line bg-surface transition-[width] duration-200"
      >
        <SidebarContent collapsed={collapsed} />
      </aside>
    );
  }
  return (
    <Dialog.Root open={mobileOpen} onOpenChange={onMobileOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-30 bg-ink/50" />
        <Dialog.Popup
          id="sidebar"
          className="fixed inset-y-0 left-0 z-40 flex w-[var(--sidebar-w)] max-w-[85vw] flex-col border-r border-line bg-surface outline-none"
        >
          <Dialog.Title className="sr-only">Menu de navigation</Dialog.Title>
          <SidebarContent collapsed={false} onNavigate={() => onMobileOpenChange(false)} mobile />
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

interface ContentProps {
  collapsed: boolean;
  mobile?: boolean;
  onNavigate?: () => void;
}

function SidebarContent({ collapsed, mobile, onNavigate }: ContentProps) {
  const { account, association } = useSession();
  const { pathname } = useLocation();
  const isAdmin = account.role === "admin";
  // Un membre ne voit jamais les rubriques d'administration, même sur /admin (FR-019).
  const space = isAdmin ? spaceFromPath(pathname) : "member";
  const home = space === "admin" ? "/admin" : "/";

  return (
    <Tooltip.Provider delay={200}>
      <div className="flex h-16 shrink-0 items-center gap-3 border-b border-line px-4">
        <Link
          to={home}
          onClick={onNavigate}
          className={cn(
            "flex min-w-0 flex-1 items-center gap-3 rounded",
            collapsed && "justify-center",
          )}
        >
          <AssociationMark name={association.name} logoUrl={association.logoUrl} />
          <span className={cn("min-w-0 leading-tight", collapsed && "sr-only")}>
            <span
              className="block truncate font-display text-[15px] font-bold"
              title={association.name}
            >
              {association.name}
            </span>
            {association.tagline ? (
              <span className="block truncate text-xs text-ink" title={association.tagline}>
                {association.tagline}
              </span>
            ) : null}
            <span className="block text-xs text-muted">
              {space === "admin" ? "Espace administration" : "Espace membre"}
            </span>
          </span>
        </Link>
        {mobile ? (
          <Dialog.Close render={<Button variant="tertiary" iconOnly className="text-muted" />}>
            <Icon icon={X} />
            <span className="sr-only">Fermer le menu</span>
          </Dialog.Close>
        ) : null}
      </div>

      {isAdmin ? (
        <SpaceSwitcher space={space} collapsed={collapsed} onNavigate={onNavigate} />
      ) : null}

      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden px-3 py-4">
        <nav
          id="nav-principale"
          aria-labelledby="nav-principale-titre"
          tabIndex={-1}
          className="rounded"
        >
          <h2
            id="nav-principale-titre"
            className={cn(
              "px-3 pb-2 text-xs font-bold uppercase tracking-[.08em] text-muted",
              collapsed && "sr-only",
            )}
          >
            Navigation principale
          </h2>
          <NavList
            entries={entriesFor(space, "main")}
            collapsed={collapsed}
            onNavigate={onNavigate}
          />
        </nav>
        <nav aria-labelledby="nav-secondaire-titre" className="mt-auto border-t border-line pt-4">
          <h2
            id="nav-secondaire-titre"
            className={cn(
              "px-3 pb-2 text-xs font-bold uppercase tracking-[.08em] text-muted",
              collapsed && "sr-only",
            )}
          >
            {space === "admin" ? "Configuration" : "Ressources"}
          </h2>
          <NavList
            entries={entriesFor(space, "settings")}
            collapsed={collapsed}
            onNavigate={onNavigate}
          />
        </nav>
      </div>

      <UserMenu collapsed={collapsed} />
    </Tooltip.Provider>
  );
}

function NavList({
  entries,
  collapsed,
  onNavigate,
}: {
  entries: NavEntry[];
  collapsed: boolean;
  onNavigate?: () => void;
}) {
  return (
    <ul className="flex flex-col gap-0.5">
      {entries.map((entry) => {
        const link = (
          <NavLink
            to={entry.path}
            // Les accueils ne sont actifs que sur leur adresse exacte ; les
            // autres rubriques le restent sur leurs sous-pages.
            end={entry.path === "/" || entry.path === "/admin"}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                "flex min-h-10 items-center gap-3 rounded-md px-3 py-2",
                collapsed && "justify-center px-0",
                isActive ? "bg-brand/10 font-bold text-brand-ink" : "text-ink hover:bg-canvas",
              )
            }
          >
            <Icon icon={entry.icon} />
            <span className={collapsed ? "sr-only" : "truncate"}>{entry.label}</span>
          </NavLink>
        );
        return (
          <li key={entry.id}>
            {collapsed ? (
              <Tooltip.Root>
                <Tooltip.Trigger render={link} />
                <Tooltip.Portal>
                  <Tooltip.Positioner side="right" sideOffset={8} className="z-[80]">
                    <Tooltip.Popup
                      aria-hidden="true"
                      className="rounded bg-ink px-2.5 py-1 text-sm font-bold text-white shadow-lg"
                    >
                      {entry.label}
                    </Tooltip.Popup>
                  </Tooltip.Positioner>
                </Tooltip.Portal>
              </Tooltip.Root>
            ) : (
              link
            )}
          </li>
        );
      })}
    </ul>
  );
}
