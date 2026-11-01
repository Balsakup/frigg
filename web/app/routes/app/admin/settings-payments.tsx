import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import {
  invalidatePaymentSettings,
  type PaymentSettings,
  paymentSettingsQuery,
} from "~/api/settings";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field, TextAreaField } from "~/ui/Field";
import { Switch } from "~/ui/Switch";
import { SettingsPage } from "./SettingsPage";

export const handle = { crumb: "Paiements" };

export async function clientLoader() {
  await queryClient.ensureQueryData(paymentSettingsQuery()).catch(() => undefined);
  return null;
}

interface SettingsForm {
  cash: { enabled: boolean; instructions: string };
  cheque: { enabled: boolean; payee: string; address: string; instructions: string };
  transfer: { enabled: boolean; holder: string; iban: string; bic: string; instructions: string };
}

const fields = [
  "cash.instructions",
  "cheque.payee",
  "cheque.address",
  "cheque.instructions",
  "transfer.holder",
  "transfer.iban",
  "transfer.bic",
  "transfer.instructions",
] as const;

function toForm(s: PaymentSettings): SettingsForm {
  return {
    cash: { enabled: s.cash.enabled, instructions: s.cash.instructions ?? "" },
    cheque: {
      enabled: s.cheque.enabled,
      payee: s.cheque.payee ?? "",
      address: s.cheque.address ?? "",
      instructions: s.cheque.instructions ?? "",
    },
    transfer: {
      enabled: s.transfer.enabled,
      holder: s.transfer.holder ?? "",
      iban: s.transfer.iban ?? "",
      bic: s.transfer.bic ?? "",
      instructions: s.transfer.instructions ?? "",
    },
  };
}

// Paramètres › Paiements : moyens de paiement proposés aux adhérents
// (006, US5). Le serveur valide (IBAN, champs obligatoires des moyens activés).
export default function SettingsPayments() {
  const settings = useQuery(paymentSettingsQuery());
  if (settings.data) return <SettingsView settings={settings.data} />;
  if (settings.isError) {
    return (
      <SettingsPage section="Paiements">
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger les paramètres. Rechargez la page.
        </Alert>
      </SettingsPage>
    );
  }
  return null;
}

function SettingsView({ settings }: { settings: PaymentSettings }) {
  const qc = useQueryClient();
  const update = $api.useMutation("put", "/admin/settings/payment");
  const [status, setStatus] = useState("");
  const { register, handleSubmit, setError, watch, setValue, reset, formState } =
    useForm<SettingsForm>({ defaultValues: toForm(settings) });
  const { errors } = formState;
  const cash = watch("cash.enabled");
  const cheque = watch("cheque.enabled");
  const transfer = watch("transfer.enabled");

  const onSubmit = handleSubmit(async (form) => {
    setStatus("");
    let saved: PaymentSettings;
    try {
      saved = await update.mutateAsync({ body: form });
    } catch (e) {
      if (!isProblem(e)) throw e;
      applyProblemToForm(e, setError, fields);
      return;
    }
    reset(toForm(saved));
    await invalidatePaymentSettings(qc);
    setStatus("Paramètres enregistrés.");
  });

  return (
    <SettingsPage section="Paiements">
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}
      {errors.root?.server ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {errors.root.server.message}
        </Alert>
      ) : null}
      <section aria-labelledby="moyens" className="max-w-3xl">
        <h2 id="moyens" className="font-display text-xl font-bold">
          Moyens de paiement
        </h2>
        <p className="mt-1 text-muted">
          Les informations des moyens activés sont présentées aux adhérents au moment de payer.
        </p>
        <form onSubmit={onSubmit} noValidate className="mt-4 flex flex-col gap-6">
          <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
            <legend className="px-1 font-display text-lg font-bold">Espèces</legend>
            <Switch
              label="Proposer ce moyen"
              checked={cash}
              onCheckedChange={(c) => setValue("cash.enabled", c, { shouldDirty: true })}
            />
            <TextAreaField
              label="Instructions"
              required={cash}
              maxLength={1000}
              hint="Où et quand régler : lieu, permanences…"
              error={errors.cash?.instructions?.message}
              {...register("cash.instructions")}
            />
          </fieldset>

          <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
            <legend className="px-1 font-display text-lg font-bold">Chèque</legend>
            <Switch
              label="Proposer ce moyen"
              checked={cheque}
              onCheckedChange={(c) => setValue("cheque.enabled", c, { shouldDirty: true })}
            />
            <Field
              label="Ordre"
              required={cheque}
              maxLength={120}
              autoComplete="off"
              error={errors.cheque?.payee?.message}
              {...register("cheque.payee")}
            />
            <TextAreaField
              label="Adresse d'envoi (facultatif)"
              maxLength={300}
              error={errors.cheque?.address?.message}
              {...register("cheque.address")}
            />
            <TextAreaField
              label="Instructions (facultatif)"
              maxLength={1000}
              error={errors.cheque?.instructions?.message}
              {...register("cheque.instructions")}
            />
          </fieldset>

          <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
            <legend className="px-1 font-display text-lg font-bold">Virement</legend>
            <Switch
              label="Proposer ce moyen"
              checked={transfer}
              onCheckedChange={(c) => setValue("transfer.enabled", c, { shouldDirty: true })}
            />
            <Field
              label="Titulaire du compte"
              required={transfer}
              maxLength={120}
              autoComplete="off"
              error={errors.transfer?.holder?.message}
              {...register("transfer.holder")}
            />
            <Field
              label="IBAN"
              required={transfer}
              maxLength={50}
              autoComplete="off"
              spellCheck={false}
              error={errors.transfer?.iban?.message}
              {...register("transfer.iban")}
            />
            <Field
              label="BIC (facultatif)"
              maxLength={20}
              autoComplete="off"
              spellCheck={false}
              className="max-w-56"
              error={errors.transfer?.bic?.message}
              {...register("transfer.bic")}
            />
            <TextAreaField
              label="Instructions (facultatif)"
              maxLength={1000}
              error={errors.transfer?.instructions?.message}
              {...register("transfer.instructions")}
            />
          </fieldset>

          <div>
            <Button type="submit" aria-disabled={formState.isSubmitting}>
              Enregistrer
            </Button>
          </div>
        </form>
      </section>
    </SettingsPage>
  );
}
