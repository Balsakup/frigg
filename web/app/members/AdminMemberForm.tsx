import type { ReactNode } from "react";
import { type Control, Controller, type FieldErrors, useForm } from "react-hook-form";
import { Link } from "react-router";
import type { FieldInput, FieldValue, FormField } from "~/api/custom-fields";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { CustomFieldInputs } from "~/customfields/CustomFieldInputs";
import {
  customFieldPaths,
  toFormValues,
  toInputs,
  type WithCustomFields,
} from "~/customfields/form";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { type AccountOption, AccountPicker } from "./AccountPicker";
import { DuplicateConfirm } from "./DuplicateConfirm";
import { formToBody, type MemberFormValues, memberFieldNames } from "./form";
import { MemberFields } from "./MemberFields";

export interface AdminMemberFormValues extends MemberFormValues {
  accountId: AccountOption;
}

type FormValues = AdminMemberFormValues & WithCustomFields;

interface AdminMemberFormProps {
  defaultValues: AdminMemberFormValues;
  /** Fiche signalée « Vous » : changer de compte retire ce signalement. */
  isSelf?: boolean;
  /** Champs personnalisés de membre (définitions à la création, valeurs ensuite). */
  customFields: (FormField | FieldValue)[];
  submitLabel: string;
  onSubmit: (
    body: ReturnType<typeof formToBody> & { accountId: number | null; customFields: FieldInput[] },
  ) => Promise<void>;
  extra?: ReactNode;
}

// Formulaire d'une fiche dans l'administration (ajout et modification).
export function AdminMemberForm({
  defaultValues,
  isSelf = false,
  customFields,
  submitLabel,
  onSubmit,
  extra,
}: AdminMemberFormProps) {
  const { register, handleSubmit, setError, control, watch, formState } = useForm<FormValues>({
    defaultValues: { ...defaultValues, customFields: toFormValues(customFields) },
  });
  const account = watch("accountId");

  const submit = handleSubmit(async (values) => {
    try {
      await onSubmit({
        ...formToBody(values),
        accountId: values.accountId.id,
        customFields: toInputs(customFields, values.customFields),
      });
    } catch (error) {
      if (!isProblem(error)) throw error;
      applyProblemToForm(error, setError, [
        ...memberFieldNames,
        "accountId",
        ...customFieldPaths(customFields),
      ]);
    }
  });

  return (
    <section className="max-w-3xl rounded-lg border border-line bg-surface p-5">
      {formState.errors.root?.server ? (
        <Alert kind="danger" role="alert" className="mb-4">
          {formState.errors.root.server.message}
        </Alert>
      ) : null}
      <form onSubmit={submit} noValidate className="flex flex-col gap-6">
        <Controller
          control={control}
          name="accountId"
          render={({ field, fieldState }) => (
            <AccountPicker
              value={field.value}
              onChange={field.onChange}
              error={fieldState.error?.message}
              hint={
                isSelf && account.id !== defaultValues.accountId.id
                  ? "Le signalement « Vous » sera retiré."
                  : undefined
              }
            />
          )}
        />
        <MemberFields register={register} errors={formState.errors} />
        <CustomFieldInputs
          fields={customFields}
          control={control as unknown as Control<WithCustomFields>}
          errors={formState.errors as FieldErrors<WithCustomFields>}
        />
        <DuplicateConfirm register={register} errors={formState.errors} />
        <div className="flex flex-wrap gap-2">
          <Button type="submit" aria-disabled={formState.isSubmitting}>
            {submitLabel}
          </Button>
          <Link to="/admin/membres" className={buttonClasses({ variant: "secondary" })}>
            Annuler
          </Link>
          {extra ? <div className="sm:ml-auto">{extra}</div> : null}
        </div>
      </form>
    </section>
  );
}
