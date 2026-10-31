import type { FieldErrors, UseFormRegister } from "react-hook-form";
import { Field } from "~/ui/Field";

export interface SeasonDatesForm {
  startsOn: string;
  endsOn: string;
}

interface SeasonDatesFieldsProps {
  register: UseFormRegister<SeasonDatesForm>;
  errors: FieldErrors<SeasonDatesForm>;
  /** `editable` : champ saisi ; objet : début imposé affiché en lecture seule ; `hidden` : absent. */
  startsOn: "editable" | "hidden" | { value: string; hint: string };
  startsOnHint?: string;
  endsOnMin?: string;
  endsOnMax?: string;
  endsOnHint?: string;
}

// Champs de dates partagés par la création et la modification d'une saison.
// Les bornes viennent de l'API (attributs natifs min et max) ; aucune règle
// n'est dupliquée ici, les erreurs 422 font foi (principe VI).
export function SeasonDatesFields({
  register,
  errors,
  startsOn,
  startsOnHint,
  endsOnMin,
  endsOnMax,
  endsOnHint,
}: SeasonDatesFieldsProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {startsOn === "editable" ? (
        <Field
          label="Date de début"
          type="date"
          required
          hint={startsOnHint}
          error={errors.startsOn?.message}
          {...register("startsOn", { required: "Saisissez la date de début." })}
        />
      ) : startsOn === "hidden" ? null : (
        <Field
          label="Date de début"
          type="date"
          readOnly
          value={startsOn.value}
          hint={startsOn.hint}
          className="[&_input]:bg-canvas"
        />
      )}
      <Field
        label="Date de fin"
        type="date"
        required
        min={endsOnMin}
        max={endsOnMax}
        hint={endsOnHint ?? "Dernier jour inclus."}
        error={errors.endsOn?.message}
        {...register("endsOn", { required: "Saisissez la date de fin." })}
      />
    </div>
  );
}
