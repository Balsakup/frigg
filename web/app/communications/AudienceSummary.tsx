import { Dialog } from "@base-ui/react/dialog";
import { Users, X } from "lucide-react";
import { type Audience, type AudienceMember, situationLabels } from "~/api/communications";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Icon } from "~/ui/Icon";
import { Table, TableCell, TableRow } from "~/ui/Table";

function plural(n: number, one: string, many: string) {
  return `${n} ${n > 1 ? many : one}`;
}

// Décompte d'une cible, annoncé à chaque changement (contracts/ui-routes.md 009).
export function AudienceSummary({
  audience,
  loading,
}: {
  audience: Audience | undefined;
  loading: boolean;
}) {
  const members = audience?.membersCount ?? 0;
  const recipients = audience?.recipientsCount ?? 0;
  const unreachable = audience?.unreachableCount ?? 0;
  return (
    <div className="rounded-lg border border-line bg-canvas p-4">
      <p role="status" aria-live="polite" className="flex items-center gap-2 font-bold">
        <Icon icon={Users} className="text-muted" />
        {audience === undefined
          ? "Calcul du nombre de destinataires…"
          : `${plural(members, "adhérent ciblé", "adhérents ciblés")} · ${plural(recipients, "destinataire", "destinataires")}`}
        {loading && audience !== undefined ? (
          <span className="text-sm font-normal text-muted">(mise à jour…)</span>
        ) : null}
      </p>
      {unreachable > 0 ? (
        <Alert kind="warning" className="mt-3">
          {unreachable > 1
            ? `${unreachable} adhérents n'ont aucune adresse et ne recevront pas ce message.`
            : "1 adhérent n'a aucune adresse et ne recevra pas ce message."}
        </Alert>
      ) : null}
      {audience && members > 0 ? (
        <div className="mt-3 flex flex-wrap gap-2">
          {(audience.members ?? []).length > 0 ? (
            <MembersDialog
              title="Adhérents ciblés"
              label="Voir les adhérents ciblés"
              members={audience.members ?? []}
            />
          ) : null}
          {(audience.unreachable ?? []).length > 0 ? (
            <MembersDialog
              title="Adhérents non joignables"
              label="Voir les non joignables"
              members={audience.unreachable ?? []}
            />
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function MembersDialog({
  title,
  label,
  members,
}: {
  title: string;
  label: string;
  members: AudienceMember[];
}) {
  return (
    <Dialog.Root>
      <Dialog.Trigger render={<Button variant="secondary" size="sm" />}>{label}</Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 flex max-h-[calc(100vh-2rem)] w-[min(48rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col rounded-lg border border-line bg-surface text-ink shadow-xl outline-none">
          <div className="flex items-start justify-between gap-4 p-6 pb-2">
            <Dialog.Title className="font-display text-xl font-bold">
              {title} ({members.length})
            </Dialog.Title>
            <Dialog.Close render={<Button variant="tertiary" size="sm" aria-label="Fermer" />}>
              <Icon icon={X} className="h-4 w-4" />
            </Dialog.Close>
          </div>
          <div className="min-h-0 overflow-y-auto pb-4">
            <Table
              caption={title}
              columns={[
                { key: "nom", label: "Nom" },
                { key: "prenom", label: "Prénom" },
                { key: "situation", label: "Situation" },
                { key: "tarif", label: "Tarif" },
                { key: "adresse", label: "Adresse de contact" },
              ]}
            >
              {members.map((m) => (
                <TableRow key={m.memberId}>
                  <TableCell>{m.lastName}</TableCell>
                  <TableCell>{m.firstName}</TableCell>
                  <TableCell className="whitespace-nowrap">
                    {situationLabels[m.situation]}
                  </TableCell>
                  <TableCell>{m.feeLabel}</TableCell>
                  <TableCell className="[overflow-wrap:anywhere]">
                    {m.contactEmail ?? <span className="text-muted">Aucune adresse</span>}
                  </TableCell>
                </TableRow>
              ))}
            </Table>
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
