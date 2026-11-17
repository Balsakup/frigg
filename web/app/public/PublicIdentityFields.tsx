import type { FieldErrors, FieldValues, Path, UseFormRegister } from "react-hook-form";
import { Field } from "~/ui/Field";

// Valeurs d'identité d'un formulaire public : noms identiques à ceux des
// fiches (members/form.ts), pour placer les erreurs du serveur sur leur champ.
export interface PublicIdentityValues {
  firstName: string;
  lastName: string;
  birthDate: string;
  email: string;
  phone: string;
  addressLine1: string;
  addressLine2: string;
  postalCode: string;
  city: string;
}

export const emptyPublicIdentity: PublicIdentityValues = {
  firstName: "",
  lastName: "",
  birthDate: "",
  email: "",
  phone: "",
  addressLine1: "",
  addressLine2: "",
  postalCode: "",
  city: "",
};

const addressFields = ["addressLine1", "addressLine2", "postalCode", "city"] as const;

interface PublicIdentityFieldsProps<T extends PublicIdentityValues & FieldValues> {
  register: UseFormRegister<T>;
  errors: FieldErrors<T>;
}

// Identité et coordonnées d'un formulaire public (FR-006) : règles de 005,
// avec l'adresse électronique obligatoire et l'adresse postale facultative,
// repliée tant qu'elle n'est pas en erreur. Seuls les attributs HTML natifs
// sont posés ; le serveur valide (principe VI). Autocomplétion du navigateur
// active : la personne saisit ses propres données.
export function PublicIdentityFields<T extends PublicIdentityValues & FieldValues>({
  register,
  errors,
}: PublicIdentityFieldsProps<T>) {
  const reg = (name: keyof PublicIdentityValues) => register(name as Path<T>);
  const err = (name: keyof PublicIdentityValues) => errors[name]?.message as string | undefined;
  const addressError = addressFields.some((f) => err(f));

  return (
    <>
      <fieldset className="flex flex-col gap-4">
        <legend className="mb-2 font-display text-lg font-bold">Vous</legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Prénom"
            required
            maxLength={80}
            autoComplete="given-name"
            error={err("firstName")}
            {...reg("firstName")}
          />
          <Field
            label="Nom"
            required
            maxLength={80}
            autoComplete="family-name"
            error={err("lastName")}
            {...reg("lastName")}
          />
        </div>
        <Field
          label="Date de naissance"
          type="date"
          required
          className="sm:max-w-60"
          autoComplete="bday"
          error={err("birthDate")}
          {...reg("birthDate")}
        />
      </fieldset>

      <fieldset className="flex flex-col gap-4">
        <legend className="mb-2 font-display text-lg font-bold">Vous joindre</legend>
        <Field
          label="Adresse électronique"
          type="email"
          required
          maxLength={254}
          autoComplete="email"
          hint="Un lien de confirmation vous y sera envoyé."
          error={err("email")}
          {...reg("email")}
        />
        <Field
          label="Téléphone"
          type="tel"
          inputMode="tel"
          maxLength={30}
          hint="10 chiffres, ou format international (+32 …)"
          autoComplete="tel"
          className="sm:max-w-72"
          error={err("phone")}
          {...reg("phone")}
        />
        <details className="group" open={addressError || undefined}>
          <summary className="cursor-pointer font-bold text-brand-ink underline underline-offset-2">
            Adresse postale (facultative)
          </summary>
          <div className="mt-3 flex flex-col gap-4">
            <p className="text-sm text-muted">
              Laissez vide ou complétez les trois champs obligatoires.
            </p>
            <Field
              label="Adresse"
              maxLength={100}
              autoComplete="address-line1"
              error={err("addressLine1")}
              {...reg("addressLine1")}
            />
            <Field
              label="Complément d'adresse"
              maxLength={100}
              autoComplete="address-line2"
              error={err("addressLine2")}
              {...reg("addressLine2")}
            />
            <div className="grid gap-4 sm:grid-cols-[10rem_minmax(0,1fr)]">
              <Field
                label="Code postal"
                inputMode="numeric"
                maxLength={5}
                autoComplete="postal-code"
                error={err("postalCode")}
                {...reg("postalCode")}
              />
              <Field
                label="Commune"
                maxLength={100}
                autoComplete="address-level2"
                error={err("city")}
                {...reg("city")}
              />
            </div>
          </div>
        </details>
      </fieldset>
    </>
  );
}
