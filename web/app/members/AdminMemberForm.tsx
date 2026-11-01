import type { ReactNode } from "react";
import { Controller, useForm } from "react-hook-form";
import { Link } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { type AccountOption, AccountPicker } from "./AccountPicker";
import { DuplicateConfirm } from "./DuplicateConfirm";
import { formToBody, type MemberFormValues, memberFieldNames } from "./form";
import { MemberFields } from "./MemberFields";

export interface AdminMemberFormValues extends MemberFormValues {
  accountId: AccountOption;
}

interface AdminMemberFormProps {
  defaultValues: AdminMemberFormValues;
  /** Fiche signalée « Vous » : changer de compte retire ce signalement. */
  isSelf?: boolean;
  submitLabel: string;
  onSubmit: (body: ReturnType<typeof formToBody> & { accountId: number | null }) => Promise<void>;
  extra?: ReactNode;
}

// Formulaire d'une fiche dans l'administration (ajout et modification).
export function AdminMemberForm({
  defaultValues,
  isSelf = false,
  submitLabel,
  onSubmit,
  extra,
}: AdminMemberFormProps) {
  const { register, handleSubmit, setError, control, watch, formState } =
    useForm<AdminMemberFormValues>({ defaultValues });
  const account = watch("accountId");

  const submit = handleSubmit(async (values) => {
    try {
      await onSubmit({ ...formToBody(values), accountId: values.accountId.id });
    } catch (error) {
      if (!isProblem(error)) throw error;
      applyProblemToForm(error, setError, [...memberFieldNames, "accountId"]);
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
