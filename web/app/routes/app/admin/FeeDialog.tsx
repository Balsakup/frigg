import { Dialog } from "@base-ui/react/dialog";
import { useQueryClient } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { useForm } from "react-hook-form";
import { invalidateMembershipFees, type MembershipFee } from "~/api/membership-fees";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { invalidateSeasons } from "~/api/seasons";
import { amountInput } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field, TextAreaField } from "~/ui/Field";

interface FeeForm {
  label: string;
  amount: string;
  hasRenewalAmount: boolean;
  renewalAmount: string;
  description: string;
}

function initialValues(fee?: MembershipFee): FeeForm {
  const renewal = fee?.renewalAmountCents;
  return {
    label: fee?.label ?? "",
    amount: fee ? amountInput(fee.amountCents) : "",
    hasRenewalAmount: renewal != null,
    renewalAmount: renewal != null ? amountInput(renewal) : "",
    description: fee?.description ?? "",
  };
}

// Ajout (fee absent) ou modification d'un tarif (contracts/ui-routes.md 004 ›
// Dialogue). Le serveur analyse le montant et valide les champs ; le
// formulaire ne s'appuie que sur les attributs natifs (principe VI).
export function FeeDialog({
  seasonId,
  fee,
  trigger,
  onDone,
}: {
  seasonId: number;
  fee?: MembershipFee;
  trigger: ReactElement;
  onDone: (message: string) => void;
}) {
  const qc = useQueryClient();
  const [isOpen, setOpen] = useState(false);
  const create = $api.useMutation("post", "/admin/seasons/{id}/fees");
  const update = $api.useMutation("put", "/admin/seasons/{id}/fees/{feeId}");
  const { register, handleSubmit, setError, reset, watch, formState } = useForm<FeeForm>({
    defaultValues: initialValues(fee),
  });
  const { errors } = formState;
  const hasRenewalAmount = watch("hasRenewalAmount");

  const onSubmit = handleSubmit(async (form) => {
    const body = {
      label: form.label,
      amount: form.amount,
      hasRenewalAmount: form.hasRenewalAmount,
      renewalAmount: form.hasRenewalAmount ? form.renewalAmount : "",
      description: form.description,
    };
    let saved: MembershipFee;
    try {
      saved = fee
        ? await update.mutateAsync({ params: { path: { id: seasonId, feeId: fee.id } }, body })
        : await create.mutateAsync({ params: { path: { id: seasonId } }, body });
    } catch (error) {
      if (!isProblem(error)) throw error;
      applyProblemToForm(error, setError, ["label", "amount", "renewalAmount", "description"]);
      if (error.status === 409 || error.status === 404) {
        await Promise.all([invalidateMembershipFees(qc), invalidateSeasons(qc)]);
      }
      return;
    }
    setOpen(false);
    await invalidateMembershipFees(qc);
    onDone(fee ? `Tarif ${saved.label} modifié.` : `Tarif ${saved.label} ajouté.`);
  });

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={(o) => {
        if (o) reset(initialValues(fee));
        setOpen(o);
      }}
    >
      <Dialog.Trigger render={trigger} />
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 max-h-[calc(100vh-2rem)] w-[min(34rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <Dialog.Title className="font-display text-xl font-bold">
            {fee ? `Modifier le tarif ${fee.label}` : "Ajouter un tarif"}
          </Dialog.Title>
          {errors.root?.server ? (
            <Alert kind="danger" role="alert" className="mt-4">
              {errors.root.server.message}
            </Alert>
          ) : null}
          <form onSubmit={onSubmit} noValidate className="mt-4 flex flex-col gap-5">
            <Field
              label="Libellé"
              required
              maxLength={80}
              autoComplete="off"
              error={errors.label?.message}
              {...register("label")}
            />
            <Field
              label={hasRenewalAmount ? "Montant de première adhésion" : "Montant"}
              required
              inputMode="decimal"
              autoComplete="off"
              hint="En euros. 0 pour un tarif gratuit."
              error={errors.amount?.message}
              className="max-w-48"
              {...register("amount")}
            />
            <label className="flex items-center gap-2.5 font-bold">
              <input
                type="checkbox"
                className="h-5 w-5 shrink-0 accent-brand"
                {...register("hasRenewalAmount")}
              />
              Tarif de renouvellement différent
            </label>
            {hasRenewalAmount ? (
              <Field
                label="Montant du renouvellement"
                required
                inputMode="decimal"
                autoComplete="off"
                hint="En euros."
                error={errors.renewalAmount?.message}
                className="max-w-48"
                {...register("renewalAmount")}
              />
            ) : null}
            <TextAreaField
              label="Description (facultatif)"
              maxLength={500}
              hint="Affichée sous le libellé, par exemple les conditions du tarif."
              error={errors.description?.message}
              {...register("description")}
            />
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
              <Button type="submit" aria-disabled={formState.isSubmitting}>
                {fee ? "Enregistrer" : "Ajouter"}
              </Button>
            </div>
          </form>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
