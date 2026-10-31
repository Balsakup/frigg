import { AlertDialog } from "@base-ui/react/alert-dialog";
import { CircleHelp, TriangleAlert } from "lucide-react";
import type { ReactElement, ReactNode } from "react";
import { cn } from "~/lib/cn";
import { Button, buttonClasses } from "./Button";
import { Icon } from "./Icon";

interface ConfirmDialogProps {
  /** Bouton qui ouvre la confirmation ; facultatif si le dialogue est contrôlé. */
  trigger?: ReactElement;
  title: string;
  /** Description du dialogue, annoncée à l'ouverture. */
  children: ReactNode;
  /** Contenu interactif placé après la description (champ, liste…). */
  content?: ReactNode;
  /** Actions supplémentaires à gauche de « Annuler ». */
  extraActions?: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  /** `danger` pour les actions destructives, `primary` sinon. */
  tone?: "danger" | "primary";
  confirmDisabled?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

// Confirmation des actions (constitution, principe IV : obligatoire pour les
// actions destructives) : focus piégé, Échap ferme, « Annuler » reçoit le focus.
export function ConfirmDialog({
  trigger,
  title,
  children,
  content,
  extraActions,
  confirmLabel,
  onConfirm,
  tone = "danger",
  confirmDisabled = false,
  open,
  onOpenChange,
}: ConfirmDialogProps) {
  const danger = tone === "danger";
  return (
    <AlertDialog.Root open={open} onOpenChange={onOpenChange ? (o) => onOpenChange(o) : undefined}>
      {trigger ? <AlertDialog.Trigger render={trigger} /> : null}
      <AlertDialog.Portal>
        <AlertDialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <AlertDialog.Popup className="fixed left-1/2 top-1/2 z-50 w-[min(30rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <div className="flex gap-4">
            <span
              className={cn(
                "grid h-10 w-10 shrink-0 place-items-center rounded-full",
                danger ? "bg-danger/10 text-danger" : "bg-brand/10 text-brand-ink",
              )}
            >
              <Icon icon={danger ? TriangleAlert : CircleHelp} />
            </span>
            <div className="min-w-0 flex-1">
              <AlertDialog.Title className="font-display text-xl font-bold [overflow-wrap:anywhere]">
                {title}
              </AlertDialog.Title>
              <AlertDialog.Description render={<div />} className="mt-2 text-muted">
                {children}
              </AlertDialog.Description>
              {content ? <div className="mt-4">{content}</div> : null}
            </div>
          </div>
          <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            {extraActions}
            <AlertDialog.Close className={buttonClasses({ variant: "secondary" })}>
              Annuler
            </AlertDialog.Close>
            <AlertDialog.Close
              render={
                <Button
                  variant={danger ? "danger" : "primary"}
                  onClick={onConfirm}
                  disabled={confirmDisabled}
                />
              }
            >
              {confirmLabel}
            </AlertDialog.Close>
          </div>
        </AlertDialog.Popup>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
