import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink } from "lucide-react";
import { useId, useState } from "react";
import { useForm } from "react-hook-form";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import {
  invalidatePaymentSettings,
  type PaymentSettings,
  paymentSettingsQuery,
} from "~/api/settings";
import { formatDate } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { CopyButton } from "~/ui/CopyButton";
import { Field, TextAreaField } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { Switch } from "~/ui/Switch";
import { SettingsPage } from "./SettingsPage";

export const handle = { crumb: "Paiements" };

export async function clientLoader() {
  await queryClient.ensureQueryData(paymentSettingsQuery()).catch(() => undefined);
  return null;
}

interface SettingsForm {
  card: { enabled: boolean };
  cash: { enabled: boolean; instructions: string };
  cheque: { enabled: boolean; payee: string; address: string; instructions: string };
  transfer: { enabled: boolean; holder: string; iban: string; bic: string; instructions: string };
}

const fields = [
  "card.enabled",
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
    card: { enabled: s.card.enabled },
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
  const card = watch("card.enabled");
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
          <CardFieldset
            settings={settings}
            checked={card}
            error={errors.card?.enabled?.message}
            onCheckedChange={(c) => setValue("card.enabled", c, { shouldDirty: true })}
          />
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

const helloAssoErrors: Record<string, string> = {
  unauthorized: "les identifiants ont été refusés par HelloAsso",
  forbidden: "le client API n'a pas le droit « Checkout »",
  not_found: "l'association est introuvable sur HelloAsso",
  unavailable: "HelloAsso est injoignable",
  unexpected: "HelloAsso a fait une réponse inattendue",
};

function ExternalDocLink({ href, children }: { href: string; children: string }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-1 text-brand-ink underline underline-offset-2"
    >
      {children}
      <Icon icon={ExternalLink} className="h-4 w-4" />
      <span className="sr-only"> (nouvel onglet)</span>
    </a>
  );
}

// Carte bancaire via HelloAsso (008, US2) : les identifiants viennent de la
// configuration de l'instance ; l'administrateur teste la connexion, déclare
// l'adresse de notification dans HelloAsso et active le moyen.
function CardFieldset({
  settings,
  checked,
  error,
  onCheckedChange,
}: {
  settings: PaymentSettings;
  checked: boolean;
  error?: string;
  onCheckedChange: (c: boolean) => void;
}) {
  const qc = useQueryClient();
  const test = $api.useMutation("post", "/admin/settings/payment/helloasso-test");
  const [testError, setTestError] = useState("");
  const explanation = useId();
  const c = settings.card;
  const testedOK = c.lastTest?.ok === true && c.lastTest.environment === c.environment;
  const canEnable = c.configured && (testedOK || c.enabled);

  const onTest = async () => {
    if (test.isPending) return;
    setTestError("");
    try {
      await test.mutateAsync({});
    } catch (e) {
      setTestError(isProblem(e) && e.detail ? e.detail : "Une erreur inattendue s'est produite.");
    }
    await invalidatePaymentSettings(qc);
  };

  return (
    <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
      <legend className="px-1 font-display text-lg font-bold">Carte bancaire (HelloAsso)</legend>
      {!c.configured ? (
        <p id={explanation}>
          Le paiement par carte passe par HelloAsso, la plateforme de paiement des associations. La
          personne qui héberge Frigg doit d'abord renseigner les identifiants HelloAsso de
          l'association dans la configuration de l'instance (variables FRIGG_HELLOASSO_*). Voir la{" "}
          <ExternalDocLink href="https://github.com/Balsakup/frigg#paiement-par-carte-helloasso">
            documentation d'installation
          </ExternalDocLink>{" "}
          et la{" "}
          <ExternalDocLink href="https://dev.helloasso.com">
            documentation de HelloAsso
          </ExternalDocLink>
          .
        </p>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            {c.environment === "sandbox" ? (
              <Badge tone="warning">Mode test</Badge>
            ) : (
              <Badge tone="success">Production</Badge>
            )}
            <span className="text-muted">
              Association sur HelloAsso :{" "}
              <span className="font-bold text-ink">{c.organization}</span>
            </span>
          </div>
          <p id={explanation}>
            Déclarez cette adresse de notification dans l'espace HelloAsso de l'association (Mon
            compte › Intégrations et API), puis testez la connexion. Voir la{" "}
            <ExternalDocLink href="https://dev.helloasso.com">
              documentation de HelloAsso
            </ExternalDocLink>
            .
          </p>
          {c.notificationUrl ? (
            <div className="rounded-md border border-line bg-canvas p-3">
              <p className="text-sm text-muted">Adresse de notification</p>
              <div className="mt-1 flex flex-wrap items-center justify-between gap-2">
                <code className="min-w-0 font-bold [overflow-wrap:anywhere]">
                  {c.notificationUrl}
                </code>
                <CopyButton value={c.notificationUrl} what="l'adresse de notification" />
              </div>
            </div>
          ) : null}
          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              variant="secondary"
              onClick={onTest}
              aria-disabled={test.isPending}
            >
              {test.isPending ? "Test en cours…" : "Tester la connexion"}
            </Button>
          </div>
          <div role="status">
            {c.lastTest ? (
              c.lastTest.ok ? (
                <Alert kind="success">
                  Connexion à HelloAsso réussie
                  {c.lastTest.organizationName ? ` — ${c.lastTest.organizationName}` : ""} (
                  {formatDate(c.lastTest.at)}).
                </Alert>
              ) : (
                <Alert kind="danger">
                  Connexion à HelloAsso impossible :{" "}
                  {helloAssoErrors[c.lastTest.error ?? "unexpected"]} ({formatDate(c.lastTest.at)}).
                </Alert>
              )
            ) : null}
          </div>
          {testError ? (
            <Alert kind="danger" role="alert">
              {testError}
            </Alert>
          ) : null}
        </>
      )}
      <Switch
        label="Proposer ce moyen"
        checked={checked}
        disabled={!canEnable}
        describedBy={explanation}
        error={error}
        onCheckedChange={onCheckedChange}
      />
      {c.configured && !testedOK && !c.enabled ? (
        <p className="text-sm text-muted">Testez d'abord la connexion pour activer ce moyen.</p>
      ) : null}
    </fieldset>
  );
}
