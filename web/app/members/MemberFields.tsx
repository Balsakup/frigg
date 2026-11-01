import type { FieldErrors, FieldValues, Path, UseFormRegister } from "react-hook-form";
import { Field } from "~/ui/Field";
import type { MemberFormValues } from "./form";

interface MemberFieldsProps<T extends MemberFormValues & FieldValues> {
  register: UseFormRegister<T>;
  errors: FieldErrors<T>;
  /** Fiche du titulaire : active l'autocomplétion du navigateur. */
  autocompleteSelf?: boolean;
}

// Champs communs d'une fiche membre (contracts/ui-routes.md 005 › Formulaire) :
// identité, coordonnées et adresse postale. Les règles sont celles du serveur ;
// seuls les attributs HTML natifs sont posés ici (constitution, principe VI).
export function MemberFields<T extends MemberFormValues & FieldValues>({
  register,
  errors,
  autocompleteSelf = false,
}: MemberFieldsProps<T>) {
  const reg = (name: keyof MemberFormValues) => register(name as Path<T>);
  const err = (name: keyof MemberFormValues) => errors[name]?.message as string | undefined;
  const auto = (token: string) => (autocompleteSelf ? token : "off");

  return (
    <>
      <fieldset className="flex flex-col gap-4">
        <legend className="mb-2 font-display text-lg font-bold">Identité</legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Prénom"
            required
            maxLength={80}
            autoComplete={auto("given-name")}
            error={err("firstName")}
            {...reg("firstName")}
          />
          <Field
            label="Nom"
            required
            maxLength={80}
            autoComplete={auto("family-name")}
            error={err("lastName")}
            {...reg("lastName")}
          />
        </div>
        <Field
          label="Date de naissance"
          type="date"
          required
          className="sm:max-w-60"
          autoComplete={auto("bday")}
          error={err("birthDate")}
          {...reg("birthDate")}
        />
      </fieldset>

      <fieldset className="flex flex-col gap-4">
        <legend className="mb-2 font-display text-lg font-bold">Coordonnées</legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Adresse électronique"
            type="email"
            maxLength={254}
            autoComplete={auto("email")}
            error={err("email")}
            {...reg("email")}
          />
          <Field
            label="Téléphone"
            type="tel"
            inputMode="tel"
            maxLength={30}
            hint="10 chiffres, ou format international (+32 …)"
            autoComplete={auto("tel")}
            error={err("phone")}
            {...reg("phone")}
          />
        </div>
      </fieldset>

      <fieldset className="flex flex-col gap-4">
        <legend className="mb-2 font-display text-lg font-bold">Adresse postale</legend>
        <p className="-mt-2 text-sm text-muted">
          Laissez vide ou complétez les trois champs obligatoires.
        </p>
        <Field
          label="Adresse"
          maxLength={100}
          autoComplete={auto("address-line1")}
          error={err("addressLine1")}
          {...reg("addressLine1")}
        />
        <Field
          label="Complément d'adresse"
          maxLength={100}
          autoComplete={auto("address-line2")}
          error={err("addressLine2")}
          {...reg("addressLine2")}
        />
        <div className="grid gap-4 sm:grid-cols-[10rem_minmax(0,1fr)]">
          <Field
            label="Code postal"
            inputMode="numeric"
            maxLength={5}
            autoComplete={auto("postal-code")}
            error={err("postalCode")}
            {...reg("postalCode")}
          />
          <Field
            label="Commune"
            maxLength={100}
            autoComplete={auto("address-level2")}
            error={err("city")}
            {...reg("city")}
          />
        </div>
      </fieldset>
    </>
  );
}
