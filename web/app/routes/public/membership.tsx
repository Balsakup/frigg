import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { type Control, Controller, type Path, useForm } from "react-hook-form";
import { Link } from "react-router";
import { fallbackIdentity, identityQuery } from "~/api/identity";
import { applyProblemToForm, isProblem } from "~/api/problem";
import {
  type PublicMembershipPage,
  usePublicMembershipPage,
  useSubmitPublicMembership,
} from "~/api/public";
import { sessionQuery } from "~/api/queries";
import { CustomFieldInputs } from "~/customfields/CustomFieldInputs";
import {
  customFieldPaths,
  toFormValues,
  toInputs,
  type WithCustomFields,
} from "~/customfields/form";
import type { MembershipKind } from "~/lib/format";
import { AmountSummary, FeeChoice } from "~/memberships/FeeChoice";
import { ConsentField } from "~/public/ConsentField";
import { Honeypot } from "~/public/Honeypot";
import {
  emptyPublicIdentity,
  PublicIdentityFields,
  type PublicIdentityValues,
} from "~/public/PublicIdentityFields";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Checkbox } from "~/ui/Checkbox";
import { RadioGroup } from "~/ui/RadioGroup";
import { Select } from "~/ui/Select";
import type { Route } from "./+types/membership";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Demande d'adhésion · Frigg" }];
}

interface MembershipFormValues extends PublicIdentityValues, WithCustomFields {
  seasonId: string;
  feeId: number | null;
  declaredKind: MembershipKind | "";
  consent: boolean;
  createAccount: boolean;
  website: string;
}

// Champs du corps dans l'ordre d'affichage : le focus va au premier en erreur.
const baseFields = [
  "firstName",
  "lastName",
  "birthDate",
  "email",
  "phone",
  "addressLine1",
  "addressLine2",
  "postalCode",
  "city",
  "seasonId",
  "feeId",
  "declaredKind",
] as const satisfies readonly Path<MembershipFormValues>[];

const kindOptions: { value: MembershipKind; label: string }[] = [
  { value: "first", label: "Première adhésion" },
  { value: "renewal", label: "Renouvellement" },
];

// Page d'adhésion publique (016, US1 ; contracts/ui-routes.md › /adherer) :
// une personne sans compte dépose sa demande, puis confirme son adresse par
// l'e-mail reçu. Les montants viennent de l'API (principe VI).
export default function PublicMembership() {
  const page = usePublicMembershipPage();
  // Une session de compte ouverte : la demande se fait depuis l'espace membre.
  // Le 401 attendu hors session est silencieux (client.ts).
  const session = useQuery({ ...sessionQuery, retry: false });
  const [sentTo, setSentTo] = useState<string | null>(null);

  return (
    <>
      <h1 className="font-display text-2xl font-bold">
        {sentTo ? "Consultez votre messagerie" : "Demande d'adhésion"}
      </h1>
      {session.data && !sentTo ? (
        <Alert kind="info" className="mt-4">
          Vous êtes connecté : faites votre demande depuis votre espace.{" "}
          <Link
            to="/adhesions/demande"
            className="font-bold text-brand-ink underline underline-offset-2"
          >
            Demander une adhésion
          </Link>
        </Alert>
      ) : null}
      {sentTo ? (
        <Sent email={sentTo} />
      ) : page.isPending ? null : page.isError ? (
        <Alert kind="danger" role="alert" className="mt-4">
          La page n'a pas pu être chargée. Rechargez-la dans quelques instants.
        </Alert>
      ) : !page.data.open ? (
        <Alert kind="info" className="mt-4">
          Les demandes d'adhésion ne sont pas ouvertes pour le moment.
        </Alert>
      ) : (
        <MembershipForm page={page.data} onSent={setSentTo} />
      )}
    </>
  );
}

// Écran neutre après l'envoi : identique quelle que soit l'adresse (FR-009).
function Sent({ email }: { email: string }) {
  return (
    <div className="mt-4 flex flex-col gap-4">
      <p role="status">
        Si l'adresse <strong className="[overflow-wrap:anywhere]">{email}</strong> est valide, un
        e-mail vient de vous y être envoyé. Ouvrez le lien qu'il contient dans les 48 heures pour
        confirmer votre demande : sans confirmation, elle n'est pas enregistrée.
      </p>
      <p>
        <Link to="/suivi/lien" className="font-bold text-brand-ink underline underline-offset-2">
          Je n'ai rien reçu
        </Link>
      </p>
    </div>
  );
}

function MembershipForm({
  page,
  onSent,
}: {
  page: PublicMembershipPage;
  onSent: (email: string) => void;
}) {
  const identity = useQuery(identityQuery).data ?? fallbackIdentity;
  const submit = useSubmitPublicMembership();
  const seasons = page.seasons ?? [];
  const memberFields = page.memberFields ?? [];
  const membershipFields = page.membershipFields ?? [];
  const { register, control, handleSubmit, setError, setFocus, watch, getValues, formState } =
    useForm<MembershipFormValues>({
      defaultValues: {
        ...emptyPublicIdentity,
        seasonId: String(seasons[0]?.id ?? ""),
        feeId: null,
        declaredKind: "",
        consent: false,
        createAccount: false,
        website: "",
        customFields: toFormValues([...memberFields, ...membershipFields]),
      },
    });
  const { errors } = formState;
  const season = seasons.find((s) => String(s.id) === watch("seasonId")) ?? seasons[0];
  const fees = season?.fees ?? [];
  const fee = fees.find((f) => f.id === watch("feeId"));
  const kind = watch("declaredKind");
  const fields = [
    ...baseFields,
    "consent",
    ...customFieldPaths([...memberFields, ...membershipFields]),
  ] as Path<MembershipFormValues>[];
  const customControl = control as unknown as Control<WithCustomFields>;
  const customErrors = errors as Parameters<typeof CustomFieldInputs>[0]["errors"];

  const onSubmit = handleSubmit(async (v) => {
    const custom = getValues("customFields");
    try {
      await submit.mutateAsync({
        body: {
          firstName: v.firstName,
          lastName: v.lastName,
          birthDate: v.birthDate,
          email: v.email,
          phone: v.phone,
          addressLine1: v.addressLine1,
          addressLine2: v.addressLine2,
          postalCode: v.postalCode,
          city: v.city,
          seasonId: Number(v.seasonId),
          feeId: v.feeId ?? 0,
          declaredKind: v.declaredKind,
          consent: v.consent,
          createAccount: v.createAccount,
          website: v.website,
          customFields: [...toInputs(memberFields, custom), ...toInputs(membershipFields, custom)],
        },
      });
    } catch (e) {
      if (!isProblem(e)) throw e;
      applyProblemToForm(e, setError, fields);
      const locations = new Set((e.errors ?? []).map((x) => x.location?.replace(/^body\./, "")));
      const first = fields.find((f) => locations.has(f));
      // Les choix (radios, case de consentement) ne prennent pas le focus
      // par react-hook-form : seul un champ de saisie le reçoit.
      if (first && (baseFields as readonly string[]).includes(first)) {
        setFocus(first);
      }
      return;
    }
    onSent(v.email.trim());
  });

  return (
    <>
      {errors.root?.server ? (
        <Alert kind="danger" role="alert" className="mt-4">
          {errors.root.server.message}
        </Alert>
      ) : null}
      <p className="mt-2 text-muted">
        Demandez votre adhésion à {identity.name}. Votre demande sera examinée par le bureau ; vous
        recevrez ensuite les informations de paiement.
      </p>
      <form onSubmit={onSubmit} noValidate className="mt-6 flex flex-col gap-8">
        <PublicIdentityFields register={register} errors={errors} />

        <fieldset className="flex flex-col gap-4">
          <legend className="mb-2 font-display text-lg font-bold">Votre adhésion</legend>
          {seasons.length > 1 ? (
            <Controller
              control={control}
              name="seasonId"
              render={({ field }) => (
                <Select
                  label="Saison"
                  value={field.value}
                  items={seasons.map((s) => ({ value: String(s.id), label: s.label }))}
                  onValueChange={(value) => {
                    field.onChange(value);
                  }}
                  error={errors.seasonId?.message}
                  className="max-w-xs"
                />
              )}
            />
          ) : (
            <p>
              <span className="font-bold">Saison : </span>
              {season?.label}
            </p>
          )}
          <Controller
            control={control}
            name="feeId"
            render={({ field }) => (
              <FeeChoice
                fees={fees}
                value={fees.some((f) => f.id === field.value) ? field.value : null}
                onChange={field.onChange}
                error={errors.feeId?.message}
              />
            )}
          />
          <Controller
            control={control}
            name="declaredKind"
            render={({ field }) => (
              <RadioGroup
                legend="Type d'adhésion"
                required
                value={field.value}
                onValueChange={field.onChange}
                error={errors.declaredKind?.message}
                options={kindOptions}
              />
            )}
          />
          <AmountSummary fee={fee} kind={kind} />
        </fieldset>

        <CustomFieldInputs
          fields={memberFields}
          control={customControl}
          errors={customErrors}
          legend="À propos de vous"
        />
        <CustomFieldInputs
          fields={membershipFields}
          control={customControl}
          errors={customErrors}
          legend="À propos de votre adhésion"
        />

        <div className="flex flex-col gap-4">
          <Controller
            control={control}
            name="consent"
            render={({ field }) => (
              <ConsentField
                association={identity.name}
                purpose="membership"
                checked={field.value}
                onCheckedChange={field.onChange}
                error={errors.consent?.message}
              />
            )}
          />
          <div>
            <Controller
              control={control}
              name="createAccount"
              render={({ field }) => (
                <Checkbox checked={field.value} onCheckedChange={field.onChange}>
                  Créer un compte pour suivre mes demandes
                </Checkbox>
              )}
            />
            <p className="mt-1 pl-7 text-sm text-muted">
              Après la confirmation de votre adresse, vous pourrez choisir un mot de passe.
            </p>
          </div>
          <Honeypot {...register("website")} />
        </div>

        <div>
          <Button type="submit" aria-disabled={submit.isPending}>
            Envoyer ma demande
          </Button>
        </div>
      </form>
    </>
  );
}
