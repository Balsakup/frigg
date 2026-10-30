import { CircleAlert, CircleCheck, CircleX, Info, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "~/lib/cn";
import { Icon } from "./Icon";

export type AlertKind = "success" | "info" | "warning" | "danger";

const kinds: Record<AlertKind, { box: string; icon: string; glyph: LucideIcon; label: string }> = {
  success: { box: "border-ok/40 bg-ok/5", icon: "text-ok", glyph: CircleCheck, label: "Succès" },
  info: { box: "border-info/40 bg-info/5", icon: "text-info", glyph: Info, label: "Information" },
  warning: {
    box: "border-warn/40 bg-warn/5",
    icon: "text-warn",
    glyph: CircleAlert,
    label: "Avertissement",
  },
  danger: {
    box: "border-danger/40 bg-danger/5",
    icon: "text-danger",
    glyph: CircleX,
    label: "Erreur",
  },
};

interface AlertProps {
  kind: AlertKind;
  title?: ReactNode;
  children?: ReactNode;
  actions?: ReactNode;
  className?: string;
  role?: "alert" | "status";
}

// Alerte de l'artefact ; le type est annoncé aux lecteurs d'écran.
export function Alert({ kind, title, children, actions, className, role }: AlertProps) {
  const k = kinds[kind];
  return (
    <div
      role={role}
      className={cn("flex items-start gap-3 rounded-lg border p-4", k.box, className)}
    >
      <span className={cn("mt-0.5", k.icon)}>
        <Icon icon={k.glyph} />
      </span>
      <div className="min-w-0 flex-1">
        {title ? (
          <p className="font-bold">
            <span className="sr-only">{k.label} : </span>
            {title}
          </p>
        ) : (
          <span className="sr-only">{k.label} : </span>
        )}
        {children ? <div className={title ? "mt-0.5" : undefined}>{children}</div> : null}
        {actions ? <div className="mt-3 flex flex-wrap gap-2">{actions}</div> : null}
      </div>
    </div>
  );
}
