import { Collapsible } from "@base-ui/react/collapsible";
import { ChevronDown } from "lucide-react";
import { type ReactNode, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import type { AdminEvent, EventBody } from "~/api/events";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { amountInput } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Field, TextAreaField } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { RadioGroup } from "~/ui/RadioGroup";

export interface EventFormValues {
  title: string;
  location: string;
  description: string;
  startsAt: string;
  endsAt: string;
  openness: "season_members" | "all_members" | "";
  capacity: string;
  price: string;
  registrationOpensAt: string;
  registrationClosesAt: string;
}

const fields = [
  "title",
  "location",
  "description",
  "startsAt",
  "endsAt",
  "openness",
  "capacity",
  "price",
  "registrationOpensAt",
  "registrationClosesAt",
] as const;

function initialValues(e?: AdminEvent): EventFormValues {
  return {
    title: e?.title ?? "",
    location: e?.location ?? "",
    description: e?.description ?? "",
    startsAt: e?.local.startsAt ?? "",
    endsAt: e?.local.endsAt ?? "",
    openness: e?.openness ?? "season_members",
    capacity: e?.capacity ? String(e.capacity) : "",
    price: e && e.priceCents > 0 ? amountInput(e.priceCents) : "",
    registrationOpensAt: e?.local.registrationOpensAt ?? "",
    registrationClosesAt: e?.local.registrationClosesAt ?? "",
  };
}

// Corps envoyé : le serveur valide et analyse tout (principe VI) ; le
// formulaire ne fait que transmettre la saisie.
function toBody(v: EventFormValues): EventBody {
  const capacity = v.capacity.trim();
  return {
    title: v.title,
    location: v.location,
    description: v.description.trim() ? v.description : null,
    startsAt: v.startsAt,
    endsAt: v.endsAt,
    openness: (v.openness || "season_members") as EventBody["openness"],
    capacity: capacity ? Number(capacity) : null,
    price: v.price,
    registrationOpensAt: v.registrationOpensAt || null,
    registrationClosesAt: v.registrationClosesAt || null,
  };
}

interface EventFormProps {
  event?: AdminEvent;
  submitLabel: string;
  onSubmit: (body: EventBody) => Promise<void>;
  cancel: ReactNode;
}

// Formulaire d'un événement (contracts/ui-routes.md 012 › Formulaire).
export function EventForm({ event, submitLabel, onSubmit, cancel }: EventFormProps) {
  const { register, control, handleSubmit, setError, watch, formState } = useForm<EventFormValues>({
    defaultValues: initialValues(event),
  });
  const { errors } = formState;
  const [periodOpen, setPeriodOpen] = useState(
    Boolean(event?.local.registrationOpensAt || event?.local.registrationClosesAt),
  );
  const values = watch();
  const hasRegistrations = (event?.confirmedCount ?? 0) > 0;
  const noticeChange =
    event?.status === "published" &&
    hasRegistrations &&
    (values.startsAt !== event.local.startsAt ||
      values.endsAt !== event.local.endsAt ||
      values.location.trim() !== event.location);
  const priceChanged = hasRegistrations && event && values.price !== initialValues(event).price;

  const submit = handleSubmit(async (v) => {
    try {
      await onSubmit(toBody(v));
    } catch (error) {
      if (!isProblem(error)) throw error;
      applyProblemToForm(error, setError, fields);
      if (errors.registrationOpensAt || errors.registrationClosesAt) setPeriodOpen(true);
    }
  });

  return (
    <form onSubmit={submit} noValidate className="flex flex-col gap-5">
      {errors.root?.server ? (
        <Alert kind="danger" role="alert">
          {errors.root.server.message}
        </Alert>
      ) : null}
      <Field
        label="Titre"
        required
        maxLength={120}
        error={errors.title?.message}
        {...register("title")}
      />
      <Field
        label="Lieu"
        required
        maxLength={200}
        error={errors.location?.message}
        {...register("location")}
      />
      <TextAreaField
        label="Description (facultative)"
        maxLength={5000}
        rows={6}
        hint="Programme, matériel à prévoir… Le texte est affiché tel quel."
        error={errors.description?.message}
        {...register("description")}
      />
      <div className="grid gap-5 sm:grid-cols-2">
        <Field
          label="Début"
          type="datetime-local"
          required
          hint="L'événement est rattaché à la saison qui contient sa date de début."
          error={errors.startsAt?.message}
          {...register("startsAt")}
        />
        <Field
          label="Fin"
          type="datetime-local"
          required
          error={errors.endsAt?.message}
          {...register("endsAt")}
        />
      </div>
      {event ? <p className="text-sm text-muted">Saison actuelle : {event.season.label}</p> : null}
      <Controller
        control={control}
        name="openness"
        render={({ field }) => (
          <RadioGroup
            legend="Qui peut s'inscrire ?"
            value={field.value}
            onValueChange={field.onChange}
            error={errors.openness?.message}
            options={[
              {
                value: "season_members",
                label: "Adhérents de la saison",
                description: "Seuls les membres dont l'adhésion à la saison est validée.",
              },
              {
                value: "all_members",
                label: "Toutes les fiches",
                description: "Tout membre, adhérent ou non (portes ouvertes, découverte…).",
              },
            ]}
          />
        )}
      />
      <div className="grid gap-5 sm:grid-cols-2">
        <Field
          label="Capacité (facultative)"
          type="number"
          inputMode="numeric"
          min={1}
          max={10000}
          hint="Nombre maximal d'inscrits. Laisser vide : illimité. Pas de liste d'attente."
          error={errors.capacity?.message}
          {...register("capacity")}
        />
        <Field
          label="Prix (facultatif)"
          inputMode="decimal"
          hint="En euros, par exemple 15,50. Laisser vide pour un événement gratuit."
          error={errors.price?.message}
          {...register("price")}
        />
      </div>
      {noticeChange ? (
        <Alert kind="info">Les inscrits seront prévenus par e-mail du changement.</Alert>
      ) : null}
      {priceChanged ? (
        <Alert kind="info">
          Le nouveau prix s'appliquera aux prochaines inscriptions ; les montants déjà dus ne
          changent pas.
        </Alert>
      ) : null}
      <Collapsible.Root open={periodOpen} onOpenChange={setPeriodOpen}>
        <Collapsible.Trigger className="group flex items-center gap-2 font-bold">
          <Icon
            icon={ChevronDown}
            className="h-4 w-4 transition-transform group-data-[panel-open]:rotate-180"
          />
          Période d'inscription
        </Collapsible.Trigger>
        <Collapsible.Panel>
          <p className="mt-2 text-sm text-muted">
            Par défaut, les inscriptions sont ouvertes de la publication jusqu'au début de
            l'événement.
          </p>
          <div className="mt-3 grid gap-5 sm:grid-cols-2">
            <Field
              label="Ouverture"
              type="datetime-local"
              error={errors.registrationOpensAt?.message}
              {...register("registrationOpensAt")}
            />
            <Field
              label="Clôture"
              type="datetime-local"
              error={errors.registrationClosesAt?.message}
              {...register("registrationClosesAt")}
            />
          </div>
        </Collapsible.Panel>
      </Collapsible.Root>
      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        {cancel}
        <Button type="submit" aria-disabled={formState.isSubmitting}>
          {submitLabel}
        </Button>
      </div>
    </form>
  );
}
