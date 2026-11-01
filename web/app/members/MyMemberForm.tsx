import type { ReactNode } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { DuplicateConfirm } from "./DuplicateConfirm";
import { formToBody, type MemberFormValues, memberFieldNames } from "./form";
import { MemberFields } from "./MemberFields";

export interface MyMemberFormValues extends MemberFormValues {
  isSelf: boolean;
}

interface MyMemberFormProps {
  defaultValues: MyMemberFormValues;
  /** Case « Il s'agit de moi » proposée. */
  canMarkSelf: boolean;
  /** Adresse du compte, proposée quand on coche « Il s'agit de moi ». */
  accountEmail: string;
  submitLabel: string;
  onSubmit: (body: ReturnType<typeof formToBody> & { isSelf: boolean }) => Promise<void>;
  /** Actions supplémentaires (suppression). */
  extra?: ReactNode;
}

// Formulaire d'une fiche dans l'espace membre (ajout et modification).
export function MyMemberForm({
  defaultValues,
  canMarkSelf,
  accountEmail,
  submitLabel,
  onSubmit,
  extra,
}: MyMemberFormProps) {
  const { register, handleSubmit, setError, setValue, getValues, watch, formState } =
    useForm<MyMemberFormValues>({ defaultValues });
  const isSelf = watch("isSelf");
  const selfField = register("isSelf");

  const submit = handleSubmit(async (values) => {
    try {
      await onSubmit({ ...formToBody(values), isSelf: values.isSelf });
    } catch (error) {
      if (!isProblem(error)) throw error;
      applyProblemToForm(error, setError, [...memberFieldNames, "isSelf"]);
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
        {canMarkSelf ? (
          <div>
            <label className="flex items-center gap-2 font-bold">
              <input
                type="checkbox"
                className="h-5 w-5"
                aria-describedby={formState.errors.isSelf ? "moi-erreur" : undefined}
                {...selfField}
                onChange={(e) => {
                  void selfField.onChange(e);
                  // Proposer l'adresse du compte pour sa propre fiche.
                  if (e.target.checked && getValues("email") === "") {
                    setValue("email", accountEmail);
                  }
                }}
              />
              Il s'agit de moi
            </label>
            {formState.errors.isSelf ? (
              <p id="moi-erreur" className="mt-1.5 text-sm font-bold text-danger">
                {formState.errors.isSelf.message}
              </p>
            ) : null}
          </div>
        ) : null}
        <MemberFields register={register} errors={formState.errors} autocompleteSelf={isSelf} />
        <DuplicateConfirm register={register} errors={formState.errors} />
        <div className="flex flex-wrap gap-2">
          <Button type="submit" aria-disabled={formState.isSubmitting}>
            {submitLabel}
          </Button>
          <Link to="/mes-membres" className={buttonClasses({ variant: "secondary" })}>
            Annuler
          </Link>
          {extra ? <div className="sm:ml-auto">{extra}</div> : null}
        </div>
      </form>
    </section>
  );
}
