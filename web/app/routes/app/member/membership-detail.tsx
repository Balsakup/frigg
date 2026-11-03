import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { invalidateMemberships, type MyMembership, myMembershipQuery } from "~/api/memberships";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import {
  formatAmount,
  formatDate,
  formatDay,
  membershipKindLabel,
  type PaymentMethod,
  paymentMethodLabel,
} from "~/lib/format";
import { CardPaymentPanel } from "~/memberships/CardPaymentPanel";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import { PaymentMethodInfo } from "~/memberships/PaymentMethodInfo";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { RadioGroup } from "~/ui/RadioGroup";
import NotFound from "../not-found";
import type { Route } from "./+types/membership-detail";

const title = (m: MyMembership) => `${m.member.firstName} ${m.member.lastName} — ${m.season.label}`;

export const handle = {
  crumb: (data: unknown) => {
    const m = data as MyMembership | null;
    return m ? title(m) : "Adhésion introuvable";
  },
};

function parseID(value: string | undefined) {
  const id = Number(value);
  return Number.isInteger(id) && id > 0 ? id : null;
}

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const id = parseID(params.id);
  if (id === null) return null;
  return queryClient.ensureQueryData(myMembershipQuery(id)).catch(() => null);
}

// Détail d'une adhésion d'un de mes membres : annulation de la demande et
// paiement (006, US3, US4). Une adhésion d'une autre fiche donne « Page
// introuvable ».
export default function MembershipDetail({ params }: Route.ComponentProps) {
  const id = parseID(params.id);
  const query = useQuery({ ...myMembershipQuery(id ?? 0), enabled: id !== null });
  if (query.data) return <MembershipView membership={query.data} />;
  if (id === null || (isProblem(query.error) && query.error.status === 404)) return <NotFound />;
  if (query.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger cette adhésion. Rechargez la page.
      </Alert>
    );
  }
  return null;
}

function MembershipView({ membership: m }: { membership: MyMembership }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const cancel = $api.useMutation("post", "/me/memberships/{id}/cancel");
  const [error, setError] = useState("");
  const path = { params: { path: { id: m.id } } };

  const onCancel = async () => {
    setError("");
    try {
      await cancel.mutateAsync(path);
    } catch (e) {
      setError(isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.");
      await invalidateMemberships(qc);
      return;
    }
    await invalidateMemberships(qc);
    navigate("/adhesions", { state: { flash: "Demande annulée." } });
  };

  const kindText = m.declaredKind
    ? m.kind && m.kind !== m.declaredKind
      ? `${membershipKindLabel(m.kind)} (retenu par le bureau ; déclaré : ${membershipKindLabel(m.declaredKind).toLowerCase()})`
      : membershipKindLabel(m.declaredKind)
    : m.kind
      ? membershipKindLabel(m.kind)
      : "—";

  const details = (
    <section
      aria-labelledby="adhesion-infos"
      className="rounded-lg border border-line bg-surface p-5"
    >
      <h2 id="adhesion-infos" className="font-display text-lg font-bold">
        Adhésion
      </h2>
      <dl className="mt-3 grid gap-3 sm:grid-cols-2">
        <Info label="Membre" value={`${m.member.firstName} ${m.member.lastName}`} />
        <Info label="Saison" value={m.season.label} />
        <Info label="Tarif" value={m.fee.label} />
        <Info label="Type" value={kindText} />
        <Info label="Montant" value={formatAmount(m.amountCents)} />
        <Info label="Date de la demande" value={formatDate(m.requestedAt)} />
        <div>
          <dt className="text-sm text-muted">État</dt>
          <dd className="mt-0.5">
            <MembershipBadges status={m.status} amountCents={m.amountCents} payment={m.payment} />
          </dd>
        </div>
        {m.reason ? <Info label="Motif" value={m.reason} /> : null}
      </dl>
      {m.actions.cancel ? (
        <div className="mt-4">
          <ConfirmDialog
            title={`Annuler la demande d'adhésion de ${m.member.firstName} ?`}
            confirmLabel="Annuler la demande"
            onConfirm={onCancel}
            trigger={<Button variant="danger-secondary">Annuler la demande</Button>}
          >
            <p>Vous pourrez déposer une nouvelle demande pour cette saison.</p>
          </ConfirmDialog>
        </div>
      ) : null}
    </section>
  );

  return (
    <Page title={title(m)}>
      {error ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {error}
        </Alert>
      ) : null}
      <div className="flex max-w-3xl flex-col gap-6">
        <CardReturn membership={m} />
        {m.actions.pay ? <PaymentSection membership={m} /> : null}
        {details}
        {!m.actions.pay ? <PaymentStatus membership={m} /> : null}
      </div>
    </Page>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 whitespace-pre-line font-bold [overflow-wrap:anywhere]">{value}</dd>
    </div>
  );
}

// Paiement terminé ou sans objet.
function PaymentStatus({ membership: m }: { membership: MyMembership }) {
  if (m.status !== "validated") return null;
  if (!m.payment) {
    return (
      <Alert kind="success" className="max-w-3xl">
        Adhésion gratuite : aucun paiement n'est demandé.
      </Alert>
    );
  }
  if (m.payment.status === "paid" && m.payment.paidOn && m.payment.paidMethod) {
    return (
      <Alert kind="success" className="max-w-3xl">
        Payée le {formatDay(m.payment.paidOn)} (
        {paymentMethodLabel(m.payment.paidMethod).toLowerCase()}).
        {m.payment.online?.test ? " Mode test : aucun débit réel." : ""}
      </Alert>
    );
  }
  return null;
}

// Choix du moyen de paiement (US4) : informations du moyen sélectionné et
// confirmation du choix.
function PaymentSection({ membership: m }: { membership: MyMembership }) {
  const qc = useQueryClient();
  const announce = $api.useMutation("put", "/me/memberships/{id}/payment");
  const payment = m.payment;
  const methods = m.paymentMethods ?? [];
  const announced = payment?.announcedMethod ?? null;
  const announcedAvailable = announced !== null && methods.some((x) => x.method === announced);
  const [choice, setChoice] = useState<PaymentMethod | "">(announcedAvailable ? announced : "");
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  if (!payment) return null;
  const selected = methods.find((x) => x.method === choice);
  const card = methods.find((x) => x.method === "card");

  const onConfirm = async () => {
    setError("");
    setStatus("");
    if (!choice) {
      setError("Choisissez un moyen de paiement.");
      return;
    }
    try {
      await announce.mutateAsync({ params: { path: { id: m.id } }, body: { method: choice } });
    } catch (e) {
      setError(
        isProblem(e)
          ? (e.errors?.[0]?.message ?? e.detail ?? "")
          : "Une erreur inattendue s'est produite.",
      );
      await invalidateMemberships(qc);
      return;
    }
    await invalidateMemberships(qc);
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
        Montant : <span className="font-bold">{formatAmount(m.amountCents)}</span>
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
          Contactez le bureau pour régler cette adhésion.
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
                membershipId={m.id}
                amountCents={selected.amountCents ?? m.amountCents}
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
              <Button onClick={onConfirm} aria-disabled={announce.isPending}>
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
type Outcome = (typeof outcomes)[keyof typeof outcomes];
const pollEvery = 3000;
const pollTries = 10;

// Retour de HelloAsso (008, US1) : la page vérifie la tentative auprès du
// serveur, qui seul interroge HelloAsso, puis retire les paramètres de
// l'adresse pour qu'un rechargement ne rejoue rien.
function CardReturn({ membership: m }: { membership: MyMembership }) {
  const qc = useQueryClient();
  const [params, setParams] = useSearchParams();
  const confirm = $api.useMutation("post", "/me/memberships/{id}/card-payment/confirm");
  const [state, setState] = useState<"" | "pending" | "succeeded" | "failed" | "late">("");
  const started = useRef(false);

  useEffect(() => {
    const outcome = outcomes[params.get("carte") as keyof typeof outcomes] as Outcome | undefined;
    const attempt = Number(params.get("tentative"));
    if (started.current || !outcome || !Number.isInteger(attempt) || attempt <= 0) return;
    started.current = true;
    setParams({}, { replace: true });
    if (outcome === "back") {
      // Retour sans payer : la tentative est close, la page de paiement reste.
      confirm.mutate({
        params: { path: { id: m.id } },
        body: { onlinePaymentId: attempt, outcome },
      });
      return;
    }
    let tries = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const check = async () => {
      tries++;
      try {
        const out = await confirm.mutateAsync({
          params: { path: { id: m.id } },
          body: { onlinePaymentId: attempt, outcome },
        });
        await invalidateMemberships(qc);
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
  }, [params, setParams, confirm, m.id, qc]);

  const messages = {
    "": null,
    pending: (
      <Alert kind="info" className="max-w-3xl">
        Confirmation du paiement en cours…
      </Alert>
    ),
    succeeded: (
      <Alert kind="success" className="max-w-3xl">
        Paiement reçu, merci. L'adhésion de {m.member.firstName} est réglée.
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
