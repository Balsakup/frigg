import { useEffect } from "react";
import { useForm } from "react-hook-form";
import type { FieldInput, FieldValue, FormField } from "~/api/custom-fields";
import type { Problem } from "~/api/problem";
import { customFieldPath, toFormValues, toInputs, type WithCustomFields } from "./form";

// Formulaire des seuls champs personnalisés, pour les écrans qui ne gèrent
// pas leurs autres champs avec react-hook-form (adhésions, 011 US3).
export function useCustomFieldsForm(fields: (FormField | FieldValue)[]) {
  const form = useForm<WithCustomFields>({
    defaultValues: { customFields: toFormValues(fields) },
  });

  // Valeurs relues du serveur (après une écriture ailleurs sur la page) : le
  // formulaire reprend les valeurs enregistrées. La signature ne change pas
  // lors d'un simple rechargement identique, la saisie en cours est conservée.
  const signature = JSON.stringify(toFormValues(fields));
  const { reset } = form;
  useEffect(() => {
    reset({ customFields: JSON.parse(signature) });
  }, [signature, reset]);

  return {
    control: form.control,
    errors: form.formState.errors,
    reset: (next: (FormField | FieldValue)[]) => reset({ customFields: toFormValues(next) }),
    inputs: (): FieldInput[] => toInputs(fields, form.getValues("customFields")),
    clearErrors: () => form.clearErrors(),
    // Place les erreurs `body.customFields.f<id>` sur leur champ ; renvoie
    // vrai si au moins une a été placée.
    applyProblem: (problem: Problem): boolean => {
      let placed = false;
      for (const e of problem.errors ?? []) {
        const match = e.location?.match(/^body\.customFields\.f(\d+)$/);
        if (!match) continue;
        const id = Number(match[1]);
        if (!fields.some((f) => f.id === id)) continue;
        form.setError(customFieldPath(id), { type: "server", message: e.message ?? "" });
        placed = true;
      }
      return placed;
    },
  };
}
