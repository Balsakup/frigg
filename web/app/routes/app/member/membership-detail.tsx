import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router";
import { invalidateMemberships, type MyMembership, myMembershipQuery } from "~/api/memberships";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { CustomFieldInputs } from "~/customfields/CustomFieldInputs";
import { CustomFieldValues } from "~/customfields/CustomFieldValues";
import { useCustomFieldsForm } from "~/customfields/useCustomFieldsForm";
import {
  formatAmount,
  formatDate,
  formatDay,
  membershipKindLabel,
  paymentMethodLabel,
} from "~/lib/format";
import { MembershipBadges } from "~/memberships/MembershipBadges";
import {
  CardReturn as SharedCardReturn,
  PaymentSection as SharedPaymentSection,
} from "~/payments/PaymentPanel";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
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
        <MyMembershipCustomFields membership={m} />
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

// Paiement de l'adhésion (US4) : page de paiement partagée (payments/).
function PaymentSection({ membership: m }: { membership: MyMembership }) {
  const qc = useQueryClient();
  const announce = $api.useMutation("put", "/me/memberships/{id}/payment");
  const start = $api.useMutation("post", "/me/memberships/{id}/card-payment");
  if (!m.payment) return null;
  const path = { params: { path: { id: m.id } } };
  return (
    <SharedPaymentSection
      amountCents={m.amountCents}
      payment={m.payment}
      methods={m.paymentMethods ?? []}
      announce={(method) => announce.mutateAsync({ ...path, body: { method } })}
      startCard={() => start.mutateAsync(path)}
      refresh={() => invalidateMemberships(qc)}
      subject="cette adhésion"
    />
  );
}

// Retour de HelloAsso (008, US1).
function CardReturn({ membership: m }: { membership: MyMembership }) {
  const qc = useQueryClient();
  const confirm = $api.useMutation("post", "/me/memberships/{id}/card-payment/confirm");
  return (
    <SharedCardReturn
      confirm={(attempt, outcome) =>
        confirm.mutateAsync({
          params: { path: { id: m.id } },
          body: { onlinePaymentId: attempt, outcome },
        })
      }
      refresh={() => invalidateMemberships(qc)}
      successMessage={`Paiement reçu, merci. L'adhésion de ${m.member.firstName} est réglée.`}
    />
  );
}

// Champs personnalisés d'adhésion visibles par le membre (011, US3) : les
// champs remplis par le membre restent modifiables tant que l'adhésion est
// en attente ou validée.
function MyMembershipCustomFields({ membership: m }: { membership: MyMembership }) {
  const qc = useQueryClient();
  const fields = m.customFields ?? [];
  const editable = fields.filter((f) => f.editable);
  const readOnly = fields.filter((f) => !f.editable);
  const custom = useCustomFieldsForm(editable);
  const save = $api.useMutation("put", "/me/memberships/{id}/fields");
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  if (fields.length === 0) return null;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatus("");
    setError("");
    custom.clearErrors();
    try {
      await save.mutateAsync({
        params: { path: { id: m.id } },
        body: { customFields: custom.inputs() },
      });
    } catch (err) {
      if (isProblem(err) && err.status === 422 && custom.applyProblem(err)) return;
      setError(isProblem(err) && err.detail ? err.detail : "Une erreur inattendue s'est produite.");
      await invalidateMemberships(qc);
      return;
    }
    await invalidateMemberships(qc);
    setStatus("Informations enregistrées.");
  };

  return (
    <section
      aria-labelledby="informations-complementaires"
      className="rounded-lg border border-line bg-surface p-5"
    >
      <h2 id="informations-complementaires" className="font-display text-lg font-bold">
        Informations complémentaires
      </h2>
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mt-3">
          {status}
        </Alert>
      ) : null}
      {error ? (
        <Alert kind="danger" role="alert" className="mt-3">
          {error}
        </Alert>
      ) : null}
      {readOnly.length ? (
        <div className="mt-3">
          <CustomFieldValues fields={readOnly} />
        </div>
      ) : null}
      {editable.length ? (
        <form onSubmit={submit} noValidate className="mt-4 flex flex-col gap-5">
          <CustomFieldInputs
            fields={editable}
            control={custom.control}
            errors={custom.errors}
            legend="À compléter"
          />
          <div>
            <Button type="submit" variant="secondary" aria-disabled={save.isPending}>
              Enregistrer les informations
            </Button>
          </div>
        </form>
      ) : null}
    </section>
  );
}
