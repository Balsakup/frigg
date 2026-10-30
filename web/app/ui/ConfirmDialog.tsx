import { AlertDialog } from "@base-ui/react/alert-dialog";
import { TriangleAlert } from "lucide-react";
import type { ReactElement, ReactNode } from "react";
import { Button, buttonClasses } from "./Button";
import { Icon } from "./Icon";

interface ConfirmDialogProps {
  /** Bouton qui ouvre la confirmation. */
  trigger: ReactElement;
  title: string;
  children: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
}

// Confirmation des actions destructives (constitution, principe IV) : focus
// piégé, Échap ferme, « Annuler » reçoit le focus.
export function ConfirmDialog({
  trigger,
  title,
  children,
  confirmLabel,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <AlertDialog.Root>
      <AlertDialog.Trigger render={trigger} />
      <AlertDialog.Portal>
        <AlertDialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <AlertDialog.Popup className="fixed left-1/2 top-1/2 z-50 w-[min(30rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <div className="flex gap-4">
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-danger/10 text-danger">
              <Icon icon={TriangleAlert} />
            </span>
            <div className="min-w-0">
              <AlertDialog.Title className="font-display text-xl font-bold">
                {title}
              </AlertDialog.Title>
              <AlertDialog.Description render={<div />} className="mt-2 text-muted">
                {children}
              </AlertDialog.Description>
            </div>
          </div>
          <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <AlertDialog.Close className={buttonClasses({ variant: "secondary" })}>
              Annuler
            </AlertDialog.Close>
            <AlertDialog.Close render={<Button variant="danger" onClick={onConfirm} />}>
              {confirmLabel}
            </AlertDialog.Close>
          </div>
        </AlertDialog.Popup>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
