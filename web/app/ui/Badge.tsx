import type { ReactNode } from "react";
import { cn } from "~/lib/cn";

export type BadgeTone = "neutral" | "brand" | "success" | "info" | "warning" | "danger" | "muted";

const tones: Record<BadgeTone, string> = {
  neutral: "bg-ink/5 text-ink",
  brand: "bg-brand/10 text-brand-ink",
  success: "bg-ok/10 text-ok",
  info: "bg-info/10 text-info",
  warning: "bg-warn/10 text-warn",
  danger: "bg-danger/10 text-danger",
  muted: "bg-ink/5 text-muted",
};

// Pastille de statut de l'artefact.
export function Badge({ tone = "neutral", children }: { tone?: BadgeTone; children: ReactNode }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 whitespace-nowrap rounded-full px-2 py-0.5 text-sm font-bold",
        tones[tone],
      )}
    >
      {children}
    </span>
  );
}
