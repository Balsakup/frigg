import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { type Path, useForm } from "react-hook-form";
import { useBlocker } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import {
  type GeneralSettings,
  generalSettingsQuery,
  invalidateGeneralSettings,
} from "~/api/settings";
import { formatPhone } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Field, TextAreaField } from "~/ui/Field";
import { SettingsPage } from "./SettingsPage";

export const handle = { crumb: "Général" };

export async function clientLoader() {
  await queryClient.ensureQueryData(generalSettingsQuery()).catch(() => undefined);
  return null;
}

interface GeneralForm {
  name: string;
  tagline: string;
  shortName: string;
  purpose: string;
  rna: string;
  siret: string;
  foundedOn: string;
  address: { line1: string; line2: string; postalCode: string; city: string };
  contactEmail: string;
  phone: string;
  website: string;
  social: {
    facebook: string;
    instagram: string;
    linkedin: string;
    youtube: string;
    mastodon: string;
    bluesky: string;
  };
}

// Ordre d'affichage : le focus va au premier champ en erreur.
const fields = [
  "name",
  "tagline",
  "shortName",
  "purpose",
  "rna",
  "siret",
  "foundedOn",
  "address.line1",
  "address.line2",
  "address.postalCode",
  "address.city",
  "contactEmail",
  "phone",
  "website",
  "social.facebook",
  "social.instagram",
  "social.linkedin",
  "social.youtube",
  "social.mastodon",
  "social.bluesky",
] as const satisfies readonly Path<GeneralForm>[];

const networks = [
  { key: "facebook", label: "Facebook" },
  { key: "instagram", label: "Instagram" },
  { key: "linkedin", label: "LinkedIn" },
  { key: "youtube", label: "YouTube" },
  { key: "mastodon", label: "Mastodon" },
  { key: "bluesky", label: "Bluesky" },
] as const;

function toForm(s: GeneralSettings): GeneralForm {
  const v = (x?: string | null) => x ?? "";
  return {
    name: s.name,
    tagline: v(s.tagline),
    shortName: v(s.shortName),
    purpose: v(s.purpose),
    rna: v(s.rna),
    siret: v(s.siret),
    foundedOn: v(s.foundedOn),
    address: {
      line1: v(s.address.line1),
      line2: v(s.address.line2),
      postalCode: v(s.address.postalCode),
      city: v(s.address.city),
    },
    contactEmail: v(s.contactEmail),
    phone: s.phone ? formatPhone(s.phone) : "",
    website: v(s.website),
    social: {
      facebook: v(s.social.facebook),
      instagram: v(s.social.instagram),
      linkedin: v(s.social.linkedin),
      youtube: v(s.social.youtube),
      mastodon: v(s.social.mastodon),
      bluesky: v(s.social.bluesky),
    },
  };
}

// Date du jour au format AAAA-MM-JJ, en heure locale (borne du champ date).
function todayISO() {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// Paramètres › Général (010, US1, US2, US4). Le serveur normalise et vérifie
// chaque champ (RNA, SIRET, téléphone, adresses web) ; l'écran affiche ses 422.
export default function SettingsGeneral() {
  const settings = useQuery(generalSettingsQuery());
  if (settings.data) return <GeneralView settings={settings.data} />;
  if (settings.isError) {
    return (
      <SettingsPage section="Général">
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger les paramètres. Rechargez la page.
        </Alert>
      </SettingsPage>
    );
  }
  return null;
}

function GeneralView({ settings }: { settings: GeneralSettings }) {
  const qc = useQueryClient();
  const update = $api.useMutation("put", "/admin/settings/general");
  const [status, setStatus] = useState("");
  const [invalid, setInvalid] = useState(false);
  const { register, handleSubmit, setError, setFocus, reset, watch, formState } =
    useForm<GeneralForm>({ defaultValues: toForm(settings) });
  const { errors, isDirty } = formState;
  const tagline = watch("tagline");

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      isDirty && currentLocation.pathname !== nextLocation.pathname,
  );
  useEffect(() => {
    if (!isDirty) return;
    const warn = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [isDirty]);

  const onSubmit = handleSubmit(async (form) => {
    setStatus("");
    setInvalid(false);
    let saved: GeneralSettings;
    try {
      saved = await update.mutateAsync({ body: form });
    } catch (e) {
      if (!isProblem(e)) throw e;
      applyProblemToForm(e, setError, fields);
      const locations = new Set((e.errors ?? []).map((x) => x.location?.replace(/^body\./, "")));
      const first = fields.find((f) => locations.has(f));
      if (first) {
        setInvalid(true);
        setFocus(first);
      }
      return;
    }
    reset(toForm(saved));
    await invalidateGeneralSettings(qc);
    setStatus("Paramètres enregistrés.");
  });

  return (
    <SettingsPage section="Général">
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-3xl">
          {status}
        </Alert>
      ) : null}
      {invalid ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          Certains champs sont à corriger : ils sont signalés ci-dessous.
        </Alert>
      ) : null}
      {errors.root?.server ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {errors.root.server.message}
        </Alert>
      ) : null}

      <form onSubmit={onSubmit} noValidate className="flex max-w-3xl flex-col gap-6">
        <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
          <legend className="px-1 font-display text-lg font-bold">Identité</legend>
          <Field
            label="Nom de l'association"
            required
            maxLength={120}
            autoComplete="organization"
            error={errors.name?.message}
            {...register("name")}
          />
          <Field
            label="Tagline (facultatif)"
            maxLength={150}
            hint={`Affichée sous le nom dans le menu et sur l'accueil des membres. ${tagline.length}/150 caractères.`}
            error={errors.tagline?.message}
            {...register("tagline")}
          />
          <Field
            label="Nom court (facultatif)"
            maxLength={20}
            hint="Sigle ou abréviation, par exemple « SVR »."
            className="max-w-xs"
            error={errors.shortName?.message}
            {...register("shortName")}
          />
          <TextAreaField
            label="Objet (facultatif)"
            maxLength={2000}
            rows={4}
            hint="But de l'association, tel qu'énoncé dans ses statuts."
            error={errors.purpose?.message}
            {...register("purpose")}
          />
        </fieldset>

        <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
          <legend className="px-1 font-display text-lg font-bold">Identité légale</legend>
          <Field
            label="Numéro RNA (facultatif)"
            maxLength={20}
            autoComplete="off"
            spellCheck={false}
            hint="W suivi de 9 chiffres, par exemple W751234567."
            className="max-w-xs"
            error={errors.rna?.message}
            {...register("rna")}
          />
          <Field
            label="Numéro SIRET (facultatif)"
            maxLength={30}
            inputMode="numeric"
            autoComplete="off"
            hint="14 chiffres."
            className="max-w-xs"
            error={errors.siret?.message}
            {...register("siret")}
          />
          <Field
            label="Date de création (facultatif)"
            type="date"
            max={todayISO()}
            className="max-w-xs"
            error={errors.foundedOn?.message}
            {...register("foundedOn")}
          />
        </fieldset>

        <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
          <legend className="px-1 font-display text-lg font-bold">Coordonnées</legend>
          <Field
            label="Adresse du siège (facultatif)"
            maxLength={100}
            autoComplete="address-line1"
            error={errors.address?.line1?.message}
            {...register("address.line1")}
          />
          <Field
            label="Complément d'adresse (facultatif)"
            maxLength={100}
            autoComplete="address-line2"
            error={errors.address?.line2?.message}
            {...register("address.line2")}
          />
          <div className="grid gap-4 sm:grid-cols-[10rem_1fr]">
            <Field
              label="Code postal"
              maxLength={10}
              inputMode="numeric"
              autoComplete="postal-code"
              error={errors.address?.postalCode?.message}
              {...register("address.postalCode")}
            />
            <Field
              label="Ville"
              maxLength={100}
              autoComplete="address-level2"
              error={errors.address?.city?.message}
              {...register("address.city")}
            />
          </div>
          <Field
            label="Adresse électronique de contact (facultatif)"
            type="email"
            maxLength={254}
            autoComplete="email"
            error={errors.contactEmail?.message}
            {...register("contactEmail")}
          />
          <Field
            label="Téléphone (facultatif)"
            type="tel"
            maxLength={30}
            autoComplete="tel"
            className="max-w-xs"
            error={errors.phone?.message}
            {...register("phone")}
          />
          <Field
            label="Site web (facultatif)"
            type="url"
            maxLength={300}
            autoComplete="url"
            hint="Adresse complète, commençant par https://."
            error={errors.website?.message}
            {...register("website")}
          />
        </fieldset>

        <fieldset className="flex flex-col gap-4 rounded-lg border border-line bg-surface p-5">
          <legend className="px-1 font-display text-lg font-bold">Réseaux sociaux</legend>
          <p className="text-sm text-muted">
            Les liens renseignés sont présentés aux membres sur l'accueil de leur espace.
          </p>
          {networks.map((n) => (
            <Field
              key={n.key}
              label={`Lien ${n.label} (facultatif)`}
              type="url"
              maxLength={300}
              autoComplete="off"
              error={errors.social?.[n.key]?.message}
              {...register(`social.${n.key}`)}
            />
          ))}
        </fieldset>

        <div>
          <Button type="submit" aria-disabled={formState.isSubmitting}>
            Enregistrer
          </Button>
        </div>
      </form>

      <ConfirmDialog
        open={blocker.state === "blocked"}
        onOpenChange={(o) => {
          if (!o && blocker.state === "blocked") blocker.reset();
        }}
        title="Quitter sans enregistrer ?"
        confirmLabel="Quitter"
        onConfirm={() => blocker.proceed?.()}
      >
        Les modifications des paramètres seront perdues.
      </ConfirmDialog>
    </SettingsPage>
  );
}
