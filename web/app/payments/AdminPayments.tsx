import { Dialog } from "@base-ui/react/dialog";
import { ExternalLink, RefreshCw } from "lucide-react";
import { type ReactElement, useState } from "react";
import { isProblem } from "~/api/problem";
import type { components } from "~/api/schema";
import { formatAmount, formatDate, formatMoney, type ManualPaymentMethod } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Field, TextAreaField } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { RadioGroup } from "~/ui/RadioGroup";
import { Table } from "~/ui/Table";

// Panneaux de paiement de l'administration, communs aux adhésions (006, 008)
// et aux inscriptions (012). Chaque cible fournit ses mutations.

type OnlinePayment = components["schemas"]["OnlinePayment"];
type PaymentAlert = components["schemas"]["PaymentAlert"];

export const methodOptions: { value: ManualPaymentMethod; label: string }[] = [
  { value: "cash", label: "Espèces" },
  { value: "cheque", label: "Chèque" },
  { value: "transfer", label: "Virement" },
];

export function errorMessage(e: unknown) {
  if (!isProblem(e)) return "Une erreur inattendue s'est produite.";
  return (
    e.errors?.map((x) => x.message).join(" ") || e.detail || "Une erreur inattendue s'est produite."
  );
}

export function todayISO() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

interface DialogShellProps {
  title: string;
  trigger: ReactElement;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}

export function DialogShell({ title, trigger, open, onOpenChange, children }: DialogShellProps) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Trigger render={trigger} />
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 max-h-[calc(100vh-2rem)] w-[min(34rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <Dialog.Title className="font-display text-xl font-bold">{title}</Dialog.Title>
          {children}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

export interface RecordedPayment {
  method: ManualPaymentMethod;
  receivedOn: string;
  note?: string;
}

interface RecordPaymentDialogProps {
  amountCents: number;
  reference?: string;
  announcedMethod?: string | null;
  record: (body: RecordedPayment) => Promise<unknown>;
  onDone: (message: string) => Promise<void>;
  onError: (e: unknown) => Promise<void>;
  trigger?: ReactElement;
}

// Constatation d'un paiement reçu (006 US4 §9, 012 US5).
export function RecordPaymentDialog({
  amountCents,
  reference,
  announcedMethod,
  record,
  onDone,
  onError,
  trigger,
}: RecordPaymentDialogProps) {
  const [open, setOpen] = useState(false);
  const initialMethod = (announcedMethod ?? "") as ManualPaymentMethod | "";
  const [method, setMethod] = useState<ManualPaymentMethod | "">(initialMethod);
  const [receivedOn, setReceivedOn] = useState(todayISO());
  const [note, setNote] = useState("");
  const [pending, setPending] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!method) {
      setErrors({ method: "Choisissez le moyen de paiement." });
      return;
    }
    setPending(true);
    try {
      await record({ method, receivedOn, ...(note.trim() ? { note } : {}) });
    } catch (err) {
      setPending(false);
      if (isProblem(err) && err.status === 422) {
        const next: Record<string, string> = {};
        for (const x of err.errors ?? [])
          next[x.location?.replace("body.", "") ?? "root"] = x.message ?? "";
        setErrors(next);
        return;
      }
      setOpen(false);
      await onError(err);
      return;
    }
    setPending(false);
    setOpen(false);
    await onDone("Paiement enregistré.");
  };

  return (
    <DialogShell
      title="Marquer comme payée"
      trigger={trigger ?? <Button>Marquer comme payée</Button>}
      open={open}
      onOpenChange={(o) => {
        if (o) {
          setMethod(initialMethod);
          setReceivedOn(todayISO());
          setNote("");
          setErrors({});
        }
        setOpen(o);
      }}
    >
      <form onSubmit={submit} noValidate className="mt-4 flex flex-col gap-5">
        <p className="text-muted">
          {formatAmount(amountCents)}
          {reference ? ` · référence ${reference}` : ""}
        </p>
        <RadioGroup
          legend="Moyen de paiement"
          required
          value={method}
          onValueChange={setMethod}
          error={errors.method}
          options={methodOptions}
        />
        <Field
          label="Date de réception"
          type="date"
          required
          value={receivedOn}
          onChange={(e) => setReceivedOn(e.target.value)}
          error={errors.receivedOn}
          className="max-w-56"
        />
        <TextAreaField
          label="Commentaire (facultatif)"
          maxLength={200}
          hint="Par exemple le numéro du chèque."
          value={note}
          onChange={(e) => setNote(e.target.value)}
          error={errors.note}
        />
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
          <Button type="submit" aria-disabled={pending}>
            Enregistrer le paiement
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}

const onlineStatus: Record<
  string,
  { label: string; tone: "success" | "info" | "danger" | "muted" }
> = {
  pending: { label: "En cours", tone: "info" },
  succeeded: { label: "Réussi", tone: "success" },
  failed: { label: "Échoué", tone: "danger" },
  abandoned: { label: "Abandonné", tone: "muted" },
  expired: { label: "Expiré", tone: "muted" },
  refunded: { label: "Remboursé", tone: "info" },
};

const onlineErrors: Record<string, string> = {
  unavailable: "HelloAsso injoignable",
  unauthorized: "identifiants HelloAsso refusés",
  forbidden: "droits HelloAsso insuffisants",
  payment_error: "paiement refusé ou erreur chez HelloAsso",
  environment_changed: "changement d'environnement HelloAsso",
  no_checkout: "page de paiement non créée",
  redirect_host: "réponse inattendue de HelloAsso",
};

export const alertLabels: Record<string, string> = {
  duplicate: "Paiement en double — remboursement à effectuer dans HelloAsso.",
  amount_mismatch: "Montant reçu différent du montant attendu.",
  refund_due: "Paiement par carte reçu — remboursement à effectuer dans HelloAsso.",
  unverified: "Paiement par carte à vérifier dans HelloAsso.",
};

export function HelloAssoOrder({ orderId, url }: { orderId: number; url?: string | null }) {
  if (!url) return <>n° {orderId}</>;
  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-1 text-brand-ink underline underline-offset-2"
    >
      n° {orderId}
      <Icon icon={ExternalLink} className="h-4 w-4" />
      <span className="sr-only"> (espace HelloAsso, nouvel onglet)</span>
    </a>
  );
}

interface PaymentAlertsPanelProps {
  alerts: PaymentAlert[];
  resolve: (id: number) => Promise<unknown>;
  onDone: (message: string) => Promise<void>;
  onError: (e: unknown) => Promise<void>;
}

// Signalements de paiement à traiter (008, US3).
export function PaymentAlertsPanel({ alerts, resolve, onDone, onError }: PaymentAlertsPanelProps) {
  const open = alerts.filter((a) => !a.resolvedAt);
  if (open.length === 0) return null;
  return (
    <div className="flex flex-col gap-3">
      {open.map((alert) => (
        <Alert
          key={alert.id}
          kind="warning"
          actions={
            <ConfirmDialog
              title="Marquer ce signalement comme traité ?"
              confirmLabel="Marquer comme traité"
              onConfirm={async () => {
                try {
                  await resolve(alert.id);
                } catch (e) {
                  await onError(e);
                  return;
                }
                await onDone("Signalement traité.");
              }}
              trigger={
                <Button variant="secondary" size="sm">
                  Marquer comme traité
                </Button>
              }
            >
              <p>{alertLabels[alert.kind]}</p>
              <p className="mt-2">Le signalement disparaîtra du tableau de bord.</p>
            </ConfirmDialog>
          }
        >
          {alertLabels[alert.kind]}{" "}
          <span className="text-muted">(signalé le {formatDate(alert.createdAt)})</span>
        </Alert>
      ))}
    </div>
  );
}

interface OnlinePaymentsPanelProps {
  ops: OnlinePayment[];
  pending: boolean;
  canVerify: boolean;
  verify: () => Promise<unknown>;
  onDone: (message: string) => Promise<void>;
  onError: (e: unknown) => Promise<void>;
}

// Tentatives de paiement par carte et vérification auprès de HelloAsso (008, US3).
export function OnlinePaymentsPanel({
  ops,
  pending,
  canVerify,
  verify,
  onDone,
  onError,
}: OnlinePaymentsPanelProps) {
  const [verifying, setVerifying] = useState(false);
  if (ops.length === 0) return null;
  const onVerify = async () => {
    if (verifying) return;
    setVerifying(true);
    try {
      await verify();
    } catch (e) {
      setVerifying(false);
      await onError(e);
      return;
    }
    setVerifying(false);
    await onDone("Vérification auprès de HelloAsso effectuée.");
  };
  return (
    <section
      aria-labelledby="paiements-en-ligne"
      className="rounded-lg border border-line bg-surface"
    >
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 pt-5">
        <h2 id="paiements-en-ligne" className="font-display text-lg font-bold">
          Paiements en ligne
        </h2>
        {canVerify ? (
          <Button variant="secondary" size="sm" onClick={onVerify} aria-disabled={verifying}>
            <Icon icon={RefreshCw} className="h-4 w-4" />
            {verifying ? "Vérification…" : "Vérifier auprès de HelloAsso"}
          </Button>
        ) : null}
      </div>
      {pending ? <p className="px-5 pt-2 text-muted">Un paiement par carte est en cours.</p> : null}
      <div className="mt-3 pb-2">
        <Table
          caption="Tentatives de paiement par carte"
          columns={[
            { key: "date", label: "Date" },
            { key: "status", label: "État" },
            { key: "amount", label: "Montant" },
            { key: "order", label: "Commande HelloAsso" },
          ]}
        >
          {ops.map((o) => (
            <tr key={o.id} className="border-b border-line">
              <td className="whitespace-nowrap px-5 py-2.5">{formatDate(o.createdAt)}</td>
              <td className="px-5 py-2.5">
                <span className="inline-flex flex-wrap items-center gap-2">
                  <Badge tone={onlineStatus[o.status]?.tone ?? "muted"}>
                    {onlineStatus[o.status]?.label ?? o.status}
                  </Badge>
                  {o.environment === "sandbox" ? <Badge tone="warning">Mode test</Badge> : null}
                </span>
                {o.errorCode ? (
                  <span className="block text-sm text-muted">
                    {onlineErrors[o.errorCode] ?? o.errorCode}
                  </span>
                ) : null}
              </td>
              <td className="whitespace-nowrap px-5 py-2.5">
                {formatMoney(o.receivedAmountCents ?? o.amountCents)}
                {o.receivedAmountCents !== null &&
                o.receivedAmountCents !== undefined &&
                o.receivedAmountCents !== o.amountCents ? (
                  <span className="block text-sm text-muted">
                    attendu {formatMoney(o.amountCents)}
                  </span>
                ) : null}
              </td>
              <td className="px-5 py-2.5">
                {o.orderId ? <HelloAssoOrder orderId={o.orderId} url={o.helloassoUrl} /> : "—"}
              </td>
            </tr>
          ))}
        </Table>
      </div>
    </section>
  );
}
