import { Dialog } from "@base-ui/react/dialog";
import { Paperclip, X } from "lucide-react";
import type { Attachment } from "~/api/communications";
import { formatBytes } from "~/lib/format";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

export interface PreviewContent {
  subject: string;
  html: string;
}

// Aperçu du message tel qu'il sera reçu : le HTML rendu par le serveur est
// isolé dans une iframe sans aucun droit (research.md 009 › R9).
export function MessagePreview({
  open,
  onOpenChange,
  preview,
  attachments,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  preview: PreviewContent | null;
  attachments: Attachment[];
}) {
  return (
    <Dialog.Root open={open} onOpenChange={(o) => onOpenChange(o)}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex max-h-[calc(100vh-2rem)] w-[min(44rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col gap-3 overflow-y-auto rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <div className="flex items-start justify-between gap-4">
            <Dialog.Title className="font-display text-xl font-bold">
              Aperçu du message
            </Dialog.Title>
            <Dialog.Close render={<Button variant="tertiary" size="sm" aria-label="Fermer" />}>
              <Icon icon={X} className="h-4 w-4" />
            </Dialog.Close>
          </div>
          {preview ? (
            <>
              <p className="[overflow-wrap:anywhere]">
                <span className="font-bold">Objet : </span>
                {preview.subject}
              </p>
              {attachments.length > 0 ? (
                <ul className="flex flex-col gap-1 text-sm" aria-label="Pièces jointes">
                  {attachments.map((a) => (
                    <li key={a.id} className="flex items-center gap-2">
                      <Icon icon={Paperclip} className="h-4 w-4 text-muted" />
                      <span className="[overflow-wrap:anywhere]">{a.name}</span>
                      <span className="text-muted">({formatBytes(a.sizeBytes)})</span>
                    </li>
                  ))}
                </ul>
              ) : null}
              <iframe
                title="Aperçu du message"
                sandbox=""
                srcDoc={preview.html}
                className="h-96 w-full rounded-md border border-line bg-white"
              />
            </>
          ) : (
            <p className="text-muted">Préparation de l'aperçu…</p>
          )}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
