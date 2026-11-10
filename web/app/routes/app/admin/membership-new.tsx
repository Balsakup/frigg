import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { customFieldsQuery } from "~/api/custom-fields";
import { memberQuery } from "~/api/members";
import { membershipFeesQuery } from "~/api/membership-fees";
import { invalidateMemberships, membershipPageQuery } from "~/api/memberships";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { seasonListQuery } from "~/api/seasons";
import { CustomFieldInputs } from "~/customfields/CustomFieldInputs";
import { adminFormFields } from "~/customfields/form";
import { useCustomFieldsForm } from "~/customfields/useCustomFieldsForm";
import { formatAge, type ManualPaymentMethod, type MembershipKind } from "~/lib/format";
import { FeeChoice } from "~/memberships/FeeChoice";
import { type MemberOption, MemberPicker } from "~/memberships/MemberPicker";
import { AmountToPayField } from "~/payments/AmountToPayField";
import { useTrackedAmount } from "~/payments/useTrackedAmount";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { Field, TextAreaField } from "~/ui/Field";
import { RadioGroup } from "~/ui/RadioGroup";
import { Select } from "~/ui/Select";

export const handle = { crumb: "Créer une adhésion" };

export async function clientLoader() {
  await Promise.all([
    queryClient.ensureQueryData(seasonListQuery()).catch(() => undefined),
    queryClient.ensureQueryData(customFieldsQuery()).catch(() => undefined),
  ]);
  return null;
}

const kindOptions: { value: MembershipKind; label: string }[] = [
  { value: "first", label: "Première adhésion" },
  { value: "renewal", label: "Renouvellement" },
];

const methodOptions: { value: ManualPaymentMethod; label: string }[] = [
  { value: "cash", label: "Espèces" },
  { value: "cheque", label: "Chèque" },
  { value: "transfer", label: "Virement" },
];

function todayISO() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

type Errors = Record<string, string | undefined>;

// Création directe d'une adhésion validée, par exemple à partir d'un bulletin
// papier (006, US6).
export default function MembershipNew() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const preselected = Number(params.get("membre")) || 0;
  const seasons = useQuery(seasonListQuery());
  const open = (seasons.data?.items ?? []).filter((s) => s.status === "open");
  const [seasonId, setSeasonId] = useState<number | null>(null);
  const season = open.find((s) => s.id === seasonId) ?? open[0];
  const fees = useQuery({ ...membershipFeesQuery(season?.id ?? 0), enabled: season !== undefined });
  const initial = useQuery({ ...memberQuery(preselected), enabled: preselected > 0 });
  const [member, setMember] = useState<MemberOption | null>(null);
  const [feeId, setFeeId] = useState<number | null>(null);
  const [kind, setKind] = useState<MembershipKind | "">("");
  const [paid, setPaid] = useState(false);
  const [method, setMethod] = useState<ManualPaymentMethod | "">("");
  const [receivedOn, setReceivedOn] = useState(todayISO());
  const [note, setNote] = useState("");
  // Montant reçu (014) : recopie le montant à payer tant qu'il n'est pas modifié.
  const [received, setReceived] = useState<string | null>(null);
  const [errors, setErrors] = useState<Errors>({});
  const [existing, setExisting] = useState<number | null>(null);
  const create = $api.useMutation("post", "/admin/memberships");
  const definitions = useQuery(customFieldsQuery());
  const fields = adminFormFields(definitions.data, "membership");
  const custom = useCustomFieldsForm(fields);

  const chosen: MemberOption | null =
    member ??
    (initial.data
      ? {
          id: initial.data.id,
          label: `${initial.data.lastName} ${initial.data.firstName}`,
          detail: `${formatAge(initial.data.age)} — ${initial.data.account ? initial.data.account.email : "Sans compte"}`,
        }
      : null);
  const feeList = fees.data?.items ?? [];
  const fee = feeList.find((f) => f.id === feeId);
  const amount = useTrackedAmount(fee && kind ? fee.amounts[kind] : undefined);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setExisting(null);
    const missing: Errors = {
      memberId: chosen ? undefined : "Choisissez le membre.",
      feeId: fee ? undefined : "Choisissez un tarif.",
      kind: kind ? undefined : "Choisissez le type d'adhésion.",
      "paid.method": paid && !method ? "Choisissez le moyen de paiement." : undefined,
    };
    setErrors(missing);
    custom.clearErrors();
    if (Object.values(missing).some(Boolean) || !chosen || !fee || !kind || !season) return;
    let id: number;
    try {
      const created = await create.mutateAsync({
        body: {
          memberId: chosen.id,
          seasonId: season.id,
          feeId: fee.id,
          kind,
          ...amount.payload(),
          customFields: custom.inputs(),
          ...(paid && method
            ? {
                paid: {
                  method,
                  receivedOn,
                  ...(note.trim() ? { note } : {}),
                  ...(received !== null ? { receivedAmount: received } : {}),
                },
              }
            : {}),
        },
      });
      id = created.id;
    } catch (err) {
      if (!isProblem(err)) throw err;
      const next: Errors = {};
      const placed = custom.applyProblem(err);
      for (const x of err.errors ?? []) {
        if (x.location?.startsWith("body.customFields.")) continue;
        next[x.location?.replace("body.", "") ?? "root"] = x.message;
      }
      if (err.status === 409) {
        next.root = err.detail;
        const found = await queryClient
          .fetchQuery(membershipPageQuery({ member: chosen.id, season: String(season.id) }))
          .catch(() => null);
        const active = (found?.items ?? []).find(
          (x) => x.status === "pending" || x.status === "validated",
        );
        setExisting(active?.id ?? null);
      } else if (Object.keys(next).length === 0 && !placed) {
        next.root = err.detail;
      }
      setErrors(next);
      return;
    }
    await invalidateMemberships(qc);
    navigate(`/admin/adhesions/${id}`, { state: { flash: "Adhésion enregistrée." } });
  };

  if (seasons.isPending) return null;
  if (!season) {
    return (
      <Page title="Créer une adhésion">
        <Alert kind="info" className="max-w-3xl">
          Aucune saison n'est ouverte à l'adhésion.
        </Alert>
      </Page>
    );
  }

  return (
    <Page title="Créer une adhésion" intro="L'adhésion est enregistrée directement validée.">
      <section className="max-w-3xl rounded-lg border border-line bg-surface p-5">
        {errors.root ? (
          <Alert
            kind="danger"
            role="alert"
            className="mb-4"
            actions={
              existing ? (
                <Link
                  to={`/admin/adhesions/${existing}`}
                  className={buttonClasses({ variant: "secondary", size: "sm" })}
                >
                  Ouvrir l'adhésion existante
                </Link>
              ) : undefined
            }
          >
            {errors.root}
          </Alert>
        ) : null}
        <form onSubmit={onSubmit} noValidate className="flex flex-col gap-6">
          <MemberPicker value={chosen} onChange={setMember} error={errors.memberId} />
          {open.length > 1 ? (
            <Select
              label="Saison"
              value={String(season.id)}
              items={open.map((s) => ({ value: String(s.id), label: s.label }))}
              onValueChange={(v) => {
                setSeasonId(Number(v));
                setFeeId(null);
              }}
              className="max-w-xs"
            />
          ) : (
            <p>
              <span className="font-bold">Saison : </span>
              {season.label}
            </p>
          )}
          {feeList.length === 0 && !fees.isPending ? (
            <Alert kind="info">
              Cette saison n'a aucun tarif : ajoutez-en depuis la fiche de la saison.
            </Alert>
          ) : (
            <FeeChoice fees={feeList} value={feeId} onChange={setFeeId} error={errors.feeId} />
          )}
          <RadioGroup
            legend="Type d'adhésion"
            required
            value={kind}
            onValueChange={setKind}
            error={errors.kind}
            options={kindOptions}
          />
          <AmountToPayField
            amount={amount}
            theoreticalCents={fee && kind ? fee.amounts[kind] : undefined}
            errors={{ amount: errors.amount, adjustmentReason: errors.adjustmentReason }}
          />

          <label className="flex items-center gap-2.5 font-bold">
            <input
              type="checkbox"
              className="h-5 w-5 shrink-0 accent-brand-ink"
              checked={paid}
              onChange={(e) => setPaid(e.target.checked)}
            />
            Règlement déjà reçu
          </label>
          {paid ? (
            <div className="flex flex-col gap-5 rounded-lg border border-line bg-canvas p-4">
              <RadioGroup
                legend="Moyen de paiement"
                required
                value={method}
                onValueChange={setMethod}
                error={errors["paid.method"]}
                options={methodOptions}
              />
              <Field
                label="Montant reçu (€)"
                inputMode="decimal"
                autoComplete="off"
                value={received ?? amount.text}
                onChange={(e) => setReceived(e.target.value)}
                hint="Un montant supérieur au montant à payer est enregistré comme un don."
                error={errors["paid.receivedAmount"]}
                className="max-w-56"
              />
              <Field
                label="Date de réception"
                type="date"
                required
                value={receivedOn}
                onChange={(e) => setReceivedOn(e.target.value)}
                error={errors["paid.receivedOn"]}
                className="max-w-56"
              />
              <TextAreaField
                label="Commentaire (facultatif)"
                maxLength={200}
                hint="Par exemple le numéro du chèque."
                value={note}
                onChange={(e) => setNote(e.target.value)}
                error={errors["paid.note"]}
              />
            </div>
          ) : null}

          <CustomFieldInputs fields={fields} control={custom.control} errors={custom.errors} />

          <div className="flex flex-col-reverse gap-2 sm:flex-row">
            <Link to="/admin/adhesions" className={buttonClasses({ variant: "secondary" })}>
              Annuler
            </Link>
            <Button type="submit" aria-disabled={create.isPending}>
              Enregistrer l'adhésion
            </Button>
          </div>
        </form>
      </section>
    </Page>
  );
}
