import { Hourglass } from "lucide-react";
import { useLocation } from "react-router";
import { entryForPath } from "~/shell/navigation";
import { Page } from "~/shell/Page";
import { Icon } from "~/ui/Icon";

// Rubrique livrée par une fonctionnalité ultérieure (FR-025).
export default function ComingSoon() {
  const { pathname } = useLocation();
  const title =
    entryForPath(pathname)?.label ??
    (pathname === "/mon-compte" ? "Mon compte" : "Bientôt disponible");
  return (
    <Page title={title}>
      <div className="flex max-w-prose items-start gap-3 rounded-lg border-2 border-dashed border-line-strong/60 bg-surface p-6">
        <span className="text-muted">
          <Icon icon={Hourglass} />
        </span>
        <div>
          <h2 className="font-display text-lg font-bold">Bientôt disponible</h2>
          <p className="mt-1 text-muted">
            Cette rubrique arrivera dans une prochaine version de Frigg.
          </p>
        </div>
      </div>
    </Page>
  );
}
