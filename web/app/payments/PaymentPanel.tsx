import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import type { PaymentMethodInfo as MethodInfo, Payment } from "~/api/memberships";
import { isProblem } from "~/api/problem";
import { formatAmount, type PaymentMethod, paymentMethodLabel } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { RadioGroup } from "~/ui/RadioGroup";
import { CardPaymentPanel } from "./CardPaymentPanel";
import { PaymentMethodInfo } from "./PaymentMethodInfo";

// Page de paiement commune aux adhésions (006, 008) et aux inscriptions (012) :
// choix du moyen, informations du moyen choisi, départ vers HelloAsso et
// retour. La cible fournit ses mutations ; ce composant ne connaît pas l'API.

interface PaymentSectionProps {
  amountCents: number;
  payment: Payment;
  methods: MethodInfo[];
  /** Enregistre le moyen choisi (hors carte). */
  announce: (method: Exclude<PaymentMethod, "card">) => Promise<unknown>;
  /** Lance ou reprend un paiement par carte. */
  startCard: () => Promise<{ redirectUrl: string }>;
  /** Relit les données après une écriture. */
  refresh: () => Promise<unknown>;
  /** « cette adhésion », « cette inscription ». */
  subject: string;
}

export function PaymentSection({
  amountCents,
  payment,
  methods,
  announce,
  startCard,
  refresh,
  subject,
}: PaymentSectionProps) {
  const announced = payment.announcedMethod ?? null;
  const announcedAvailable = announced !== null && methods.some((x) => x.method === announced);
  const [choice, setChoice] = useState<PaymentMethod | "">(announcedAvailable ? announced : "");
  const [pending, setPending] = useState(false);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const selected = methods.find((x) => x.method === choice);
  const card = methods.find((x) => x.method === "card");

  const onConfirm = async () => {
    setError("");
    setStatus("");
    if (!choice || choice === "card") {
      setError("Choisissez un moyen de paiement.");
      return;
    }
    setPending(true);
    try {
      await announce(choice);
    } catch (e) {
      setError(
        isProblem(e)
          ? (e.errors?.[0]?.message ?? e.detail ?? "")
          : "Une erreur inattendue s'est produite.",
      );
      setPending(false);
      await refresh();
      return;
    }
    setPending(false);
    await refresh();
    setStatus("Moyen de paiement enregistré.");
  };

  return (
    <section
      aria-labelledby="paiement"
      className="rounded-lg border border-brand-ink/40 bg-surface p-5"
    >
      <h2 id="paiement" className="font-display text-lg font-bold">
        Paiement
      </h2>
      <p className="mt-2 text-lg">
        Montant : <span className="font-bold">{formatAmount(amountCents)}</span>
        <br />
        Référence : <span className="font-bold">{payment.reference}</span>
      </p>
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mt-4">
          {status}
        </Alert>
      ) : null}
      {announced !== null && !announcedAvailable ? (
        <Alert kind="warning" className="mt-4">
          Ce moyen n'est plus proposé. Choisissez-en un autre.
        </Alert>
      ) : null}
      {card?.test ? (
        <Alert kind="warning" className="mt-4">
          Mode test — aucun débit réel.
        </Alert>
      ) : null}
      {methods.length === 0 ? (
        <Alert kind="info" className="mt-4">
          Contactez le bureau pour régler {subject}.
        </Alert>
      ) : (
        <div className="mt-4 flex flex-col gap-4">
          <RadioGroup
            legend="Comment souhaitez-vous payer ?"
            value={choice}
            onValueChange={setChoice}
            error={error}
            options={methods.map((x) => ({ value: x.method, label: paymentMethodLabel(x.method) }))}
          />
          {selected && selected.method === "card" ? (
            <div className="rounded-md border border-line bg-canvas p-4">
              <CardPaymentPanel
                start={startCard}
                amountCents={selected.amountCents ?? amountCents}
              />
            </div>
          ) : null}
          {selected && selected.method !== "card" ? (
            <div className="rounded-md border border-line bg-canvas p-4">
              <PaymentMethodInfo info={selected} reference={payment.reference} />
            </div>
          ) : null}
          {choice !== "card" ? (
            <div>
              <Button onClick={onConfirm} aria-disabled={pending}>
                {announcedAvailable ? "Changer de moyen" : "Confirmer ce moyen de paiement"}
              </Button>
            </div>
          ) : null}
        </div>
      )}
    </section>
  );
}

const outcomes = { retour: "return", erreur: "error", annule: "back" } as const;
export type CardOutcome = (typeof outcomes)[keyof typeof outcomes];
type CardStatus = "pending" | "succeeded" | "failed" | "abandoned" | "expired" | "refunded";
const pollEvery = 3000;
const pollTries = 10;

interface CardReturnProps {
  /** Vérifie la tentative auprès du serveur. */
  confirm: (attempt: number, outcome: CardOutcome) => Promise<{ status: CardStatus }>;
  refresh: () => Promise<unknown>;
  /** Message affiché quand le paiement est constaté. */
  successMessage: string;
}

// Retour de HelloAsso (008, US1) : la page vérifie la tentative auprès du
// serveur, qui seul interroge HelloAsso, puis retire les paramètres de
// l'adresse pour qu'un rechargement ne rejoue rien.
export function CardReturn({ confirm, refresh, successMessage }: CardReturnProps) {
  const [params, setParams] = useSearchParams();
  const [state, setState] = useState<"" | "pending" | "succeeded" | "failed" | "late">("");
  const started = useRef(false);
  const confirmRef = useRef(confirm);
  const refreshRef = useRef(refresh);
  confirmRef.current = confirm;
  refreshRef.current = refresh;

  useEffect(() => {
    const outcome = outcomes[params.get("carte") as keyof typeof outcomes] as
      | CardOutcome
      | undefined;
    const attempt = Number(params.get("tentative"));
    if (started.current || !outcome || !Number.isInteger(attempt) || attempt <= 0) return;
    started.current = true;
    setParams({}, { replace: true });
    if (outcome === "back") {
      // Retour sans payer : la tentative est close, la page de paiement reste.
      void confirmRef.current(attempt, outcome).catch(() => undefined);
      return;
    }
    let tries = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const check = async () => {
      tries++;
      try {
        const out = await confirmRef.current(attempt, outcome);
        await refreshRef.current();
        if (out.status === "succeeded") return setState("succeeded");
        if (out.status === "failed" || out.status === "abandoned" || out.status === "expired")
          return setState("failed");
      } catch {
        // Erreur passagère : on réessaie comme pour un paiement en cours.
      }
      if (tries >= pollTries) return setState("late");
      setState("pending");
      timer = setTimeout(check, pollEvery);
    };
    setState("pending");
    void check();
    return () => clearTimeout(timer);
  }, [params, setParams]);

  const messages = {
    "": null,
    pending: (
      <Alert kind="info" className="max-w-3xl">
        Confirmation du paiement en cours…
      </Alert>
    ),
    succeeded: (
      <Alert kind="success" className="max-w-3xl">
        {successMessage}
      </Alert>
    ),
    failed: (
      <Alert kind="danger" className="max-w-3xl">
        Le paiement n'a pas abouti. Aucune somme n'a été débitée. Vous pouvez réessayer ou choisir
        un autre moyen de paiement.
      </Alert>
    ),
    late: (
      <Alert kind="info" className="max-w-3xl">
        Votre paiement sera confirmé dans quelques minutes ; vous recevrez un e-mail.
      </Alert>
    ),
  };
  const spoken = {
    "": "",
    pending: "Confirmation du paiement en cours.",
    succeeded: "Paiement reçu, merci.",
    failed: "Le paiement n'a pas abouti. Aucune somme n'a été débitée.",
    late: "Votre paiement sera confirmé dans quelques minutes.",
  };
  return (
    <>
      <p role="status" className="sr-only">
        {spoken[state]}
      </p>
      {messages[state]}
    </>
  );
}
