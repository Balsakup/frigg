import { type ReactNode, useId } from "react";
import { cn } from "~/lib/cn";
import { ScrollRegion } from "./ScrollRegion";

export interface TableColumn {
  key: string;
  label: ReactNode;
  className?: string;
}

interface TableProps {
  /** Titre du tableau, annoncé aux lecteurs d'écran. */
  caption: string;
  captionVisible?: boolean;
  columns: TableColumn[];
  children: ReactNode;
}

// Tableau de l'artefact : en-têtes de colonne, défilement horizontal au clavier
// sous 320 px sans faire défiler la page.
export function Table({ caption, captionVisible = false, columns, children }: TableProps) {
  const captionId = useId();
  return (
    <ScrollRegion labelledBy={captionId}>
      <table className="w-full border-collapse text-left text-[15px]">
        <caption
          id={captionId}
          className={captionVisible ? "px-5 pb-2 text-left font-bold" : "sr-only"}
        >
          {caption}
        </caption>
        <thead className="text-sm">
          <tr className="border-y border-line">
            {columns.map((c) => (
              <th
                key={c.key}
                scope="col"
                className={cn("whitespace-nowrap px-5 py-2.5", c.className)}
              >
                {c.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </ScrollRegion>
  );
}

// Ligne et cellule du tableau, avec les espacements de l'artefact.
export function TableRow({ children }: { children: ReactNode }) {
  return <tr className="border-b border-line last:border-0">{children}</tr>;
}

export function TableCell({ children, className }: { children?: ReactNode; className?: string }) {
  return <td className={cn("px-5 py-3", className)}>{children}</td>;
}
