import {
  File,
  FileSpreadsheet,
  FileText,
  Image,
  type LucideIcon,
  Paperclip,
  X,
} from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { type Attachment, deleteAttachment, uploadAttachment } from "~/api/communications";
import { isProblem } from "~/api/problem";
import { cn } from "~/lib/cn";
import { formatBytes } from "~/lib/format";
import { Button, buttonClasses } from "~/ui/Button";
import { Icon } from "~/ui/Icon";

const MAX_TOTAL = 10 * 1024 * 1024;

interface Item {
  key: string;
  name: string;
  size: number;
  status: "uploading" | "done" | "error";
  attachment?: Attachment;
  error?: string;
}

function iconFor(type: string): LucideIcon {
  if (type.startsWith("image/")) return Image;
  if (type.includes("spreadsheet")) return FileSpreadsheet;
  if (type === "application/pdf" || type.includes("document") || type.includes("text"))
    return FileText;
  return File;
}

function errorMessage(error: unknown) {
  if (isProblem(error)) {
    if (error.status === 413) return "Le fichier dépasse 5 Mo.";
    const detail = error.errors?.find((e) => e.message)?.message;
    return detail ?? error.detail ?? "Le fichier n'a pas pu être ajouté.";
  }
  return "Le fichier n'a pas pu être ajouté. Vérifiez votre connexion.";
}

// Pièces jointes d'une communication : chaque fichier est déposé dès son ajout
// et vérifié par le serveur (research.md 009 › R3, R11).
export function AttachmentsField({
  draftKey,
  onChange,
}: {
  draftKey: string;
  onChange: (attachments: Attachment[], uploading: boolean) => void;
}) {
  const id = useId();
  const input = useRef<HTMLInputElement>(null);
  const [items, setItems] = useState<Item[]>([]);
  const queue = useRef(Promise.resolve());
  const seq = useRef(0);

  useEffect(() => {
    onChange(
      items.flatMap((i) => (i.attachment ? [i.attachment] : [])),
      items.some((i) => i.status === "uploading"),
    );
  }, [items, onChange]);

  const update = (key: string, patch: Partial<Item>) =>
    setItems((list) => list.map((i) => (i.key === key ? { ...i, ...patch } : i)));

  const add = (files: FileList | null) => {
    for (const file of Array.from(files ?? [])) {
      seq.current += 1;
      const key = `f${seq.current}`;
      setItems((list) => [...list, { key, name: file.name, size: file.size, status: "uploading" }]);
      // Un dépôt à la fois : le serveur vérifie nombre et total dans l'ordre.
      queue.current = queue.current.then(async () => {
        try {
          const attachment = await uploadAttachment(draftKey, file);
          update(key, {
            status: "done",
            attachment,
            name: attachment.name,
            size: attachment.sizeBytes,
          });
        } catch (error) {
          update(key, { status: "error", error: errorMessage(error) });
        }
      });
    }
    if (input.current) input.current.value = "";
  };

  const remove = async (item: Item) => {
    if (item.attachment) {
      try {
        await deleteAttachment(item.attachment.id);
      } catch (error) {
        if (!isProblem(error) || error.status !== 404) {
          update(item.key, { error: "La pièce jointe n'a pas pu être retirée. Réessayez." });
          return;
        }
      }
    }
    setItems((list) => list.filter((i) => i.key !== item.key));
  };

  const accepted = items.filter((i) => i.attachment);
  const total = accepted.reduce((n, i) => n + i.size, 0);

  return (
    <fieldset className="min-w-0">
      <legend className="mb-1 font-bold">Pièces jointes (facultatif)</legend>
      <p id={`${id}-aide`} className="mb-2 text-sm text-muted">
        PDF, JPEG, PNG, Word (.docx), Excel (.xlsx) ou OpenDocument (.odt, .ods). 5 fichiers au
        plus, 5 Mo par fichier et 10 Mo au total.
      </p>
      <input
        ref={input}
        id={`${id}-fichiers`}
        type="file"
        multiple
        accept=".pdf,.jpg,.jpeg,.png,.docx,.xlsx,.odt,.ods"
        aria-describedby={`${id}-aide`}
        className="peer sr-only"
        onChange={(e) => add(e.target.files)}
      />
      <label
        htmlFor={`${id}-fichiers`}
        className={cn(
          buttonClasses({ variant: "secondary" }),
          "cursor-pointer peer-focus-visible:outline-3 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-brand-ink",
        )}
      >
        <Icon icon={Paperclip} className="h-4 w-4" />
        Ajouter des fichiers
      </label>
      {items.length > 0 ? (
        <ul className="mt-3 flex flex-col gap-2" aria-label="Fichiers joints">
          {items.map((item) => (
            <li
              key={item.key}
              className="flex flex-col gap-1 rounded-md border border-line bg-surface px-3 py-2"
            >
              <div className="flex items-center gap-2">
                <Icon icon={iconFor(item.attachment?.contentType ?? "")} className="text-muted" />
                <span className="min-w-0 flex-1 [overflow-wrap:anywhere]">
                  {item.name}
                  <span className="ml-2 text-sm text-muted">
                    {item.status === "uploading" ? "Envoi…" : formatBytes(item.size)}
                  </span>
                </span>
                <Button
                  variant="tertiary"
                  size="sm"
                  aria-label={
                    item.status === "error"
                      ? `Fermer le message pour ${item.name}`
                      : `Retirer ${item.name}`
                  }
                  aria-disabled={item.status === "uploading"}
                  onClick={() => {
                    if (item.status !== "uploading") void remove(item);
                  }}
                >
                  <Icon icon={X} className="h-4 w-4" />
                  <span className="max-sm:sr-only">
                    {item.status === "error" ? "Fermer" : "Retirer"}
                  </span>
                </Button>
              </div>
              {item.error ? (
                <p role="alert" className="text-sm font-bold text-danger">
                  {item.error}
                </p>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
      {accepted.length > 0 ? (
        <p className="mt-2 text-sm text-muted">
          {accepted.length} fichier{accepted.length > 1 ? "s" : ""} · {formatBytes(total)} sur{" "}
          {formatBytes(MAX_TOTAL).replace(",0", "")}
        </p>
      ) : null}
    </fieldset>
  );
}
