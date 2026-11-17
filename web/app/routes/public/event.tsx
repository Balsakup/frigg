import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Controller, type Path, useForm } from "react-hook-form";
import { Link } from "react-router";
import { fallbackIdentity, identityQuery } from "~/api/identity";
import { applyProblemToForm, isProblem } from "~/api/problem";
import {
  type PublicEvent,
  publicEventQuery,
  usePublicEvent,
  useSubmitPublicRegistration,
} from "~/api/public";
import { formatEventMoment, formatEventRange } from "~/events/date";
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
import type { Route } from "./+types/event";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Événement · Frigg" }];
}

interface RegistrationFormValues extends PublicIdentityValues {
  consent: boolean;
  createAccount: boolean;
  website: string;
}

// Champs du corps dans l'ordre d'affichage : le focus va au premier en erreur.
const identityFields = [
  "firstName",
  "lastName",
  "birthDate",
  "email",
  "phone",
  "addressLine1",
  "addressLine2",
  "postalCode",
  "city",
] as const satisfies readonly Path<RegistrationFormValues>[];

const linkClass = "font-bold text-brand-ink underline underline-offset-2";

// Prix sans décimales quand elles sont nulles : « 35 € », « 12,50 € ».
const wholeEuros = new Intl.NumberFormat("fr-FR", {
  style: "currency",
  currency: "EUR",
  maximumFractionDigits: 0,
});
const centsEuros = new Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" });

function formatPrice(cents: number) {
  if (cents === 0) return "Gratuit";
  return (cents % 100 === 0 ? wholeEuros : centsEuros).format(cents / 100);
}

function placesLabel(n: number) {
  return n === 1 ? "1 place restante" : `${n} places restantes`;
}

// Raison de l'absence de formulaire (016 US3 scénario 4).
function closedMessage(ev: PublicEvent) {
  switch (ev.closedReason) {
    case "cancelled":
      return "Cet événement est annulé : les inscriptions sont fermées.";
    case "finished":
      return "Cet événement est terminé.";
    case "full":
      return "Cet événement est complet : il n'y a plus de place.";
    case "not_yet_open":
      return `Les inscriptions ouvriront le ${formatEventMoment(ev.registrationOpensAt)}.`;
    case "closed":
      return "Les inscriptions sont closes.";
    default:
      return null;
  }
}

// Page publique d'un événement « Ouvert au public » (016, US3 ;
// contracts/ui-routes.md › /e/:slug) : informations propres à l'événement
// (principe II) et formulaire d'inscription, sans champ personnalisé (FR-026).
// Tout lien sans page donne le même écran « Événement introuvable » (FR-025).
export default function PublicEventPage({ params }: Route.ComponentProps) {
  const slug = params.slug ?? "";
  const query = usePublicEvent(slug);
  const [sentTo, setSentTo] = useState<string | null>(null);

  if (isProblem(query.error) && query.error.status === 404) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Événement introuvable</h1>
        <p className="mt-4">
          Ce lien ne correspond à aucun événement ouvert aux inscriptions. Vérifiez l'adresse reçue
          ou contactez l'association.
        </p>
      </>
    );
  }
  if (query.isError) {
    return (
      <>
        <h1 className="font-display text-2xl font-bold">Événement</h1>
        <Alert kind="danger" role="alert" className="mt-4">
          La page n'a pas pu être chargée. Rechargez-la dans quelques instants.
        </Alert>
      </>
    );
  }
  if (!query.data) {
    return (
      <p role="status" className="text-muted">
        Chargement…
      </p>
    );
  }
  const ev = query.data;
  if (sentTo) return <Sent email={sentTo} title={ev.title} />;
  const closed = closedMessage(ev);
  return (
    <>
      <EventDetails event={ev} />
      {closed ? (
        <Alert
          kind={ev.closedReason === "cancelled" ? "danger" : "info"}
          title="Inscription impossible"
          className="mt-6"
        >
          {closed}
        </Alert>
      ) : (
        <RegistrationForm event={ev} slug={slug} onSent={setSentTo} />
      )}
    </>
  );
}

function EventDetails({ event: ev }: { event: PublicEvent }) {
  return (
    <>
      <h1 className="font-display text-2xl font-bold [overflow-wrap:anywhere]">{ev.title}</h1>
      <p className="mt-2 text-muted">
        {formatEventRange(ev.startsAt, ev.endsAt)} · {ev.location}
      </p>
      {ev.description ? (
        <p className="mt-4 whitespace-pre-line [overflow-wrap:anywhere]">{ev.description}</p>
      ) : null}
      <dl className="mt-5 grid gap-3 sm:grid-cols-2">
        <Info label="Date">{formatEventMoment(ev.startsAt)}</Info>
        <Info label="Lieu">{ev.location}</Info>
        <Info label="Prix">{formatPrice(ev.priceCents)}</Info>
        <Info label="Inscriptions">
          du {formatEventMoment(ev.registrationOpensAt)} au{" "}
          {formatEventMoment(ev.registrationClosesAt)}
        </Info>
        {ev.placesLeft !== null ? <Info label="Places">{placesLabel(ev.placesLeft)}</Info> : null}
      </dl>
    </>
  );
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 font-bold [overflow-wrap:anywhere]">{children}</dd>
    </div>
  );
}

// Écran neutre après l'envoi : identique quelle que soit l'adresse (FR-009).
function Sent({ email, title }: { email: string; title: string }) {
  return (
    <>
      <h1 className="font-display text-2xl font-bold">Consultez votre messagerie</h1>
      <div className="mt-4 flex flex-col gap-4">
        <p role="status">
          Si l'adresse <strong className="[overflow-wrap:anywhere]">{email}</strong> est valide, un
          e-mail vient de vous y être envoyé. Ouvrez le lien qu'il contient dans les 48 heures pour
          confirmer votre inscription à « {title} » : aucune place n'est réservée avant la
          confirmation.
        </p>
        <p>
          <Link to="/suivi/lien" className={linkClass}>
            Je n'ai rien reçu
          </Link>
        </p>
      </div>
    </>
  );
}

function RegistrationForm({
  event: ev,
  slug,
  onSent,
}: {
  event: PublicEvent;
  slug: string;
  onSent: (email: string) => void;
}) {
  const identity = useQuery(identityQuery).data ?? fallbackIdentity;
  const qc = useQueryClient();
  const submit = useSubmitPublicRegistration();
  const { register, control, handleSubmit, setError, setFocus, formState } =
    useForm<RegistrationFormValues>({
      defaultValues: { ...emptyPublicIdentity, consent: false, createAccount: false, website: "" },
    });
  const { errors } = formState;
  const fields = [...identityFields, "consent"] as Path<RegistrationFormValues>[];

  const onSubmit = handleSubmit(async (v) => {
    try {
      await submit.mutateAsync({
        params: { path: { slug } },
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
          consent: v.consent,
          createAccount: v.createAccount,
          website: v.website,
        },
      });
    } catch (e) {
      if (!isProblem(e)) throw e;
      if (e.status === 409 || e.status === 404) {
        // L'inscription est devenue impossible : la page relue affiche la
        // raison à la place du formulaire.
        setError("root.server", { message: e.detail ?? "L'inscription n'est plus possible." });
        await qc.invalidateQueries({ queryKey: publicEventQuery(slug).queryKey });
        return;
      }
      applyProblemToForm(e, setError, fields);
      const locations = new Set((e.errors ?? []).map((x) => x.location?.replace(/^body\./, "")));
      const first = identityFields.find((f) => locations.has(f));
      if (first) setFocus(first);
      return;
    }
    onSent(v.email.trim());
  });

  const label = ev.priceCents > 0 ? `M'inscrire — ${formatPrice(ev.priceCents)}` : "M'inscrire";
  return (
    <section aria-labelledby="inscription" className="mt-8">
      <h2 id="inscription" className="font-display text-xl font-bold">
        S'inscrire
      </h2>
      {errors.root?.server ? (
        <Alert kind="danger" role="alert" className="mt-4">
          {errors.root.server.message}
        </Alert>
      ) : null}
      <p className="mt-2 text-muted">
        Après l'envoi, confirmez votre adresse par l'e-mail reçu : votre place est réservée à la
        confirmation.
        {ev.priceCents > 0 ? " Vous pourrez ensuite payer depuis votre lien de suivi." : ""}
      </p>
      <form onSubmit={onSubmit} noValidate className="mt-6 flex flex-col gap-8">
        <PublicIdentityFields register={register} errors={errors} />
        <div className="flex flex-col gap-4">
          <Controller
            control={control}
            name="consent"
            render={({ field }) => (
              <ConsentField
                association={identity.name}
                purpose="registration"
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
                  Créer un compte pour suivre mes inscriptions
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
            {label}
          </Button>
        </div>
      </form>
    </section>
  );
}
