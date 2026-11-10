import { Dialog } from "@base-ui/react/dialog";
import { UserPlus } from "lucide-react";
import { useState } from "react";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import type { ManualPaymentMethod } from "~/lib/format";
import { type MemberOption, MemberPicker } from "~/memberships/MemberPicker";
import { DialogShell, methodOptions, todayISO } from "~/payments/AdminPayments";
import { AmountToPayField } from "~/payments/AmountToPayField";
import { useTrackedAmount } from "~/payments/useTrackedAmount";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Checkbox } from "~/ui/Checkbox";
import { Field } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { RadioGroup } from "~/ui/RadioGroup";

interface Props {
  eventId: number;
  priceCents: number;
  onDone: (message: string) => Promise<void>;
  onError: (e: unknown) => Promise<void>;
}

// Inscription d'un membre par l'administration (US5 §2), hors période,
// éventuellement déjà payée.
export function RegisterMemberDialog({ eventId, priceCents, onDone, onError }: Props) {
  const [open, setOpen] = useState(false);
  const [member, setMember] = useState<MemberOption | null>(null);
  const [paid, setPaid] = useState(false);
  const [method, setMethod] = useState<ManualPaymentMethod | "">("");
  const [receivedOn, setReceivedOn] = useState(todayISO());
  // Montant à payer et montant reçu (014) ; le montant reçu recopie le
  // montant à payer tant qu'il n'est pas modifié.
  const amount = useTrackedAmount(priceCents);
  const [received, setReceived] = useState<string | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const create = $api.useMutation("post", "/admin/events/{id}/registrations");

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!member) {
      setErrors({ memberId: "Choisissez un membre." });
      return;
    }
    if (paid && !method) {
      setErrors({ "payment.method": "Choisissez le moyen de paiement." });
      return;
    }
    try {
      await create.mutateAsync({
        params: { path: { id: eventId } },
        body: {
          memberId: member.id,
          ...amount.payload(),
          ...(paid && method
            ? {
                payment: {
                  method,
                  receivedOn,
                  ...(received !== null ? { receivedAmount: received } : {}),
                },
              }
            : {}),
        },
      });
    } catch (err) {
      if (isProblem(err) && (err.status === 422 || err.status === 409)) {
        const next: Record<string, string> = {};
        for (const x of err.errors ?? [])
          next[x.location?.replace("body.", "") ?? "root"] = x.message ?? "";
        if (err.status === 409) next.root = err.detail ?? "";
        setErrors(next);
        return;
      }
      setOpen(false);
      await onError(err);
      return;
    }
    setOpen(false);
    await onDone(`${member.label} est inscrit(e).`);
  };

  return (
    <DialogShell
      title="Inscrire un membre"
      trigger={
        <Button variant="secondary" size="sm">
          <Icon icon={UserPlus} className="h-4 w-4" />
          Inscrire un membre
        </Button>
      }
      open={open}
      onOpenChange={(o) => {
        if (o) {
          setMember(null);
          setPaid(false);
          setMethod("");
          setReceivedOn(todayISO());
          amount.restart(null);
          setReceived(null);
          setErrors({});
        }
        setOpen(o);
      }}
    >
      <form onSubmit={submit} noValidate className="mt-4 flex flex-col gap-5">
        {errors.root ? (
          <Alert kind="danger" role="alert">
            {errors.root}
          </Alert>
        ) : null}
        <MemberPicker value={member} onChange={setMember} error={errors.memberId} />
        <AmountToPayField
          amount={amount}
          theoreticalCents={priceCents}
          errors={{ amount: errors.amount, adjustmentReason: errors.adjustmentReason }}
        />

        <Checkbox checked={paid} onCheckedChange={setPaid}>
          Règlement déjà reçu
        </Checkbox>
        {paid ? (
          <>
            <p className="text-sm text-muted">Ignoré si l'inscription est gratuite.</p>
            <Field
              label="Montant reçu (€)"
              inputMode="decimal"
              autoComplete="off"
              value={received ?? amount.text}
              onChange={(ev) => setReceived(ev.target.value)}
              hint="Un montant supérieur au montant à payer est enregistré comme un don."
              error={errors["payment.receivedAmount"]}
              className="max-w-56"
            />
            <RadioGroup
              legend="Moyen de paiement"
              required
              value={method}
              onValueChange={setMethod}
              error={errors["payment.method"]}
              options={methodOptions}
            />
            <Field
              label="Date de réception"
              type="date"
              required
              value={receivedOn}
              onChange={(ev) => setReceivedOn(ev.target.value)}
              error={errors["payment.receivedOn"]}
              className="max-w-56"
            />
          </>
        ) : null}

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
          <Button type="submit" aria-disabled={create.isPending}>
            Inscrire
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}
