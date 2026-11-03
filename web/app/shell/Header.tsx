import { PanelLeft } from "lucide-react";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

interface HeaderProps {
  sidebarOpen: boolean;
  onToggleSidebar: () => void;
}

// En-tête : bouton de la barre latérale seul (FR-024). La recherche globale et
// les notifications viendront avec les fonctionnalités qui leur donnent du contenu.
export function Header({ sidebarOpen, onToggleSidebar }: HeaderProps) {
  return (
    <header className="sticky top-0 z-20 bg-surface/95 backdrop-blur">
      <div className="flex h-16 items-center gap-2 border-b border-line px-4 lg:px-8">
        <Button
          variant="tertiary"
          iconOnly
          className="-ml-2 text-muted hover:text-ink"
          aria-expanded={sidebarOpen}
          aria-controls="sidebar"
          onClick={onToggleSidebar}
        >
          <Icon icon={PanelLeft} />
          <span className="sr-only">Menu de navigation</span>
        </Button>
      </div>
    </header>
  );
}
