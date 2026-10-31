import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  invalidateMembershipFees,
  type MembershipFee,
  membershipFeesQuery,
} from "~/api/membership-fees";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { invalidateSeasons, type Season } from "~/api/seasons";
import { formatAmount } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Icon } from "~/ui/Icon";
import { Table, TableCell, TableRow } from "~/ui/Table";
import { FeeDialog } from "./FeeDialog";

const unexpected = "Une erreur inattendue s'est produite.";

// Section « Tarifs de l'adhésion » de la fiche d'une saison
// (contracts/ui-routes.md 004) : seules les actions annoncées par l'API sont
// proposées (research.md 004 › R5).
export function SeasonFees({
  season,
  onStatus,
  onError,
}: {
  season: Season;
  onStatus: (message: string) => void;
  onError: (message: string) => void;
}) {
  const qc = useQueryClient();
  const query = useQuery(membershipFeesQuery(season.id));
  const remove = $api.useMutation("delete", "/admin/seasons/{id}/fees/{feeId}");
  // Après une suppression, la ligne a disparu : le focus revient au bouton
  // « Ajouter un tarif ».
  const addRef = useRef<HTMLButtonElement>(null);
  const [focusAdd, setFocusAdd] = useState(false);
  useEffect(() => {
    if (focusAdd && query.isFetched) {
      addRef.current?.focus();
      setFocusAdd(false);
    }
  }, [focusAdd, query.isFetched]);

  const data = query.data;
  const items = data?.items ?? [];
  const actions = data?.actions;

  const done = (message: string) => {
    onError("");
    onStatus(message);
  };

  const onDelete = async (fee: MembershipFee) => {
    onStatus("");
    onError("");
    try {
      await remove.mutateAsync({ params: { path: { id: season.id, feeId: fee.id } } });
    } catch (e) {
      onError(isProblem(e) && e.detail ? e.detail : unexpected);
      await Promise.all([invalidateMembershipFees(qc), invalidateSeasons(qc)]);
      return;
    }
    await invalidateMembershipFees(qc);
    onStatus(`Tarif ${fee.label} supprimé.`);
    setFocusAdd(true);
  };

  const addDialog = (variant: "primary" | "secondary") => (
    <FeeDialog
      seasonId={season.id}
      onDone={done}
      trigger={
        <Button ref={addRef} variant={variant}>
          <Icon icon={Plus} className="h-4 w-4" />
          Ajouter un tarif
        </Button>
      }
    />
  );

  const withActions = actions?.edit || actions?.delete;

  return (
    <section
      aria-labelledby="saison-tarifs-titre"
      className="mt-6 max-w-3xl rounded-lg border border-line bg-surface p-5"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id="saison-tarifs-titre" className="font-display text-lg font-bold">
          Tarifs de l'adhésion
        </h2>
        {items.length > 0 ? (
          actions?.add ? (
            addDialog("secondary")
          ) : actions?.edit ? (
            <p className="text-sm text-muted">20 tarifs au plus par saison.</p>
          ) : null
        ) : null}
      </div>

      {query.isError ? (
        <Alert kind="danger" role="alert" className="mt-4">
          Impossible de charger les tarifs. Rechargez la page.
        </Alert>
      ) : null}

      {data && items.length === 0 ? (
        <>
          {actions?.edit ? (
            <Alert kind="info" className="mt-4">
              Cette saison ne sera pas proposée à l'adhésion tant qu'elle n'a aucun tarif.
            </Alert>
          ) : null}
          {actions?.add ? (
            <div className="mt-4 flex flex-col items-start gap-3">
              <p>Un tarif est le prix que paie un adhérent pour adhérer pendant la saison.</p>
              {addDialog("primary")}
            </div>
          ) : (
            <p className="mt-4 text-muted">Aucun tarif n'a été défini pour cette saison.</p>
          )}
        </>
      ) : null}

      {items.length > 0 ? (
        <div className="-mx-5 mt-4">
          <Table
            caption={`Tarifs de la saison ${season.label}`}
            columns={[
              { key: "tarif", label: "Tarif" },
              { key: "montant", label: "Montant", className: "text-right" },
              ...(withActions
                ? [{ key: "actions", label: <span className="sr-only">Actions</span> }]
                : []),
            ]}
          >
            {items.map((fee) => (
              <TableRow key={fee.id}>
                <th scope="row" className="px-5 py-3 text-left font-normal">
                  <span className="font-bold">{fee.label}</span>
                  {fee.description ? (
                    <span className="mt-0.5 block text-sm text-muted">{fee.description}</span>
                  ) : null}
                </th>
                <TableCell className="whitespace-nowrap text-right">
                  {formatAmount(fee.amountCents)}
                  {fee.renewalAmountCents != null ? (
                    <span className="mt-0.5 block text-sm text-muted">
                      Renouvellement {formatAmount(fee.renewalAmountCents)}
                    </span>
                  ) : null}
                </TableCell>
                {withActions ? (
                  <TableCell className="w-0 whitespace-nowrap">
                    <div className="flex justify-end gap-1">
                      {actions?.edit ? (
                        <FeeDialog
                          seasonId={season.id}
                          fee={fee}
                          onDone={done}
                          trigger={
                            <Button variant="tertiary" iconOnly>
                              <Icon icon={Pencil} label={`Modifier le tarif ${fee.label}`} />
                            </Button>
                          }
                        />
                      ) : null}
                      {actions?.delete ? (
                        <ConfirmDialog
                          title={`Supprimer le tarif ${fee.label} ?`}
                          confirmLabel="Supprimer"
                          onConfirm={() => onDelete(fee)}
                          trigger={
                            <Button variant="tertiary" iconOnly>
                              <Icon icon={Trash2} label={`Supprimer le tarif ${fee.label}`} />
                            </Button>
                          }
                        >
                          <p>Cette action est définitive.</p>
                        </ConfirmDialog>
                      ) : null}
                    </div>
                  </TableCell>
                ) : null}
              </TableRow>
            ))}
          </Table>
        </div>
      ) : null}
    </section>
  );
}
