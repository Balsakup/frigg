import { Check, Copy } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "./Button";
import { Icon } from "./Icon";

// Copie une valeur dans le presse-papiers et l'annonce aux lecteurs d'écran
// (contracts/ui-routes.md 006 › Détail et paiement).
export function CopyButton({ value, what }: { value: string; what: string }) {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [copied]);

  return (
    <>
      <Button
        variant="secondary"
        size="sm"
        onClick={async () => {
          try {
            await navigator.clipboard.writeText(value);
            setCopied(true);
          } catch {
            setCopied(false);
          }
        }}
      >
        <Icon icon={copied ? Check : Copy} className="h-4 w-4" />
        {copied ? "Copié" : "Copier"}
        {/* Le nom accessible reprend le texte visible (RGAA, critère 11.2). */}
        <span className="sr-only"> {what}</span>
      </Button>
      <span role="status" className="sr-only">
        {copied ? "Copié." : ""}
      </span>
    </>
  );
}
