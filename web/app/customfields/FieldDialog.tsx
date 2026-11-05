import { Dialog } from "@base-ui/react/dialog";
import { useQueryClient } from "@tanstack/react-query";
import { Plus, X } from "lucide-react";
import { type ReactElement, useState } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import {
  type FieldDefinition,
  type FieldScope,
  type FieldType,
  type FieldVisibility,
  invalidateCustomFields,
  typeLabels,
  visibilityDescriptions,
  visibilityLabels,
} from "~/api/custom-fields";
import { applyProblemToForm, isProblem, type Problem } from "~/api/problem";
import { $api } from "~/api/queries";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { Checkbox } from "~/ui/Checkbox";
import { Field } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { RadioGroup } from "~/ui/RadioGroup";

interface FieldForm {
  label: string;
  type: FieldType;
  visibility: FieldVisibility;
  required: boolean;
  help: string;
  choices: { id?: number; label: string }[];
}

const types = Object.keys(typeLabels) as FieldType[];
const visibilities = Object.keys(visibilityLabels) as FieldVisibility[];

function initialValues(field?: FieldDefinition): FieldForm {
  return {
    label: field?.label ?? "",
    type: field?.type ?? "short_text",
    visibility: field?.visibility ?? "admin",
    required: field?.required ?? false,
    help: field?.help ?? "",
    choices: field?.choices?.map((c) => ({ id: c.id, label: c.label })) ?? [
      { label: "" },
      { label: "" },
    ],
  };
}

// Le serveur désigne un choix par « choices[1] » ; le formulaire par
// « choices.1.label ».
function choicePaths(problem: Problem): Problem {
  return {
    ...problem,
    errors: problem.errors?.map((e) => ({
      ...e,
      location: e.location?.replace(/^body\.choices\[(\d+)\]$/, "body.choices.$1.label"),
    })),
  };
}

// Ajout (field absent) ou modification d'un champ personnalisé
// (contracts/ui-routes.md 011 › Dialogue). Le serveur valide tout.
export function FieldDialog({
  scope,
  field,
  trigger,
  onDone,
}: {
  scope: FieldScope;
  field?: FieldDefinition;
  trigger: ReactElement;
  onDone: (message: string) => void;
}) {
  const qc = useQueryClient();
  const [isOpen, setOpen] = useState(false);
  const create = $api.useMutation("post", "/admin/custom-fields");
  const update = $api.useMutation("put", "/admin/custom-fields/{id}");
  const { register, control, handleSubmit, setError, reset, watch, setValue, formState } =
    useForm<FieldForm>({ defaultValues: initialValues(field) });
  const { fields: choiceRows, append, remove } = useFieldArray({ control, name: "choices" });
  const { errors } = formState;
  const type = watch("type");
  const typeLocked = (field?.valueCount ?? 0) > 0;

  const onSubmit = handleSubmit(async (form) => {
    const isChoice = form.type === "choice";
    const base = {
      label: form.label,
      type: form.type,
      visibility: form.visibility,
      required: form.type === "bool" ? false : form.required,
      help: form.help,
    };
    try {
      if (field) {
        await update.mutateAsync({
          params: { path: { id: field.id } },
          body: { ...base, choices: isChoice ? form.choices : [] },
        });
      } else {
        await create.mutateAsync({
          body: { ...base, scope, choices: isChoice ? form.choices.map((c) => c.label) : [] },
        });
      }
    } catch (error) {
      if (!isProblem(error)) throw error;
      const paths = form.choices.map((_, i) => `choices.${i}.label` as const);
      applyProblemToForm(choicePaths(error), setError, [
        "label",
        "type",
        "visibility",
        "required",
        "help",
        "choices",
        ...paths,
      ]);
      if (error.status === 404 || error.status === 409) await invalidateCustomFields(qc);
      return;
    }
    setOpen(false);
    await invalidateCustomFields(qc);
    onDone(field ? "Champ enregistré." : "Champ ajouté.");
  });

  return (
    <Dialog.Root
      open={isOpen}
      onOpenChange={(o) => {
        if (o) reset(initialValues(field));
        setOpen(o);
      }}
    >
      <Dialog.Trigger render={trigger} />
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-ink/50" />
        <Dialog.Popup className="fixed left-1/2 top-1/2 z-50 max-h-[calc(100vh-2rem)] w-[min(36rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-line bg-surface p-6 text-ink shadow-xl outline-none">
          <Dialog.Title className="font-display text-xl font-bold [overflow-wrap:anywhere]">
            {field ? `Modifier le champ ${field.label}` : "Ajouter un champ"}
          </Dialog.Title>
          {errors.root?.server ? (
            <Alert kind="danger" role="alert" className="mt-4">
              {errors.root.server.message}
            </Alert>
          ) : null}
          <form onSubmit={onSubmit} noValidate className="mt-4 flex flex-col gap-5">
            <Field
              label="Libellé"
              required
              maxLength={80}
              autoComplete="off"
              hint={`${watch("label").length}/80 caractères`}
              error={errors.label?.message}
              {...register("label")}
            />
            {typeLocked ? (
              <div>
                <p className="mb-1 font-bold">Type</p>
                <p>{typeLabels[type]}</p>
                <p className="mt-1 text-sm text-muted">
                  Le type ne peut plus changer : des valeurs sont enregistrées.
                </p>
              </div>
            ) : (
              <RadioGroup
                legend="Type"
                value={type}
                onValueChange={(v) => setValue("type", v, { shouldDirty: true })}
                options={types.map((t) => ({ value: t, label: typeLabels[t] }))}
                error={errors.type?.message}
              />
            )}
            {type === "choice" ? (
              <fieldset className="flex flex-col gap-3">
                <legend className="mb-1 font-bold">
                  Choix <span className="font-normal text-muted">(2 à 50)</span>
                </legend>
                {errors.choices?.message || errors.choices?.root?.message ? (
                  <p className="text-sm font-bold text-danger" role="alert">
                    {errors.choices?.message ?? errors.choices?.root?.message}
                  </p>
                ) : null}
                {choiceRows.map((row, i) => (
                  <div key={row.id} className="flex items-start gap-2">
                    <Field
                      label={`Choix ${i + 1}`}
                      maxLength={80}
                      autoComplete="off"
                      className="flex-1"
                      error={errors.choices?.[i]?.label?.message}
                      {...register(`choices.${i}.label`)}
                    />
                    <Button
                      type="button"
                      variant="tertiary"
                      iconOnly
                      className="mt-7"
                      onClick={() => remove(i)}
                    >
                      <Icon icon={X} label={`Retirer le choix ${i + 1}`} />
                    </Button>
                  </div>
                ))}
                <div>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => append({ label: "" })}
                  >
                    <Icon icon={Plus} />
                    Ajouter un choix
                  </Button>
                </div>
              </fieldset>
            ) : null}
            <RadioGroup
              legend="Visibilité"
              value={watch("visibility")}
              onValueChange={(v) => setValue("visibility", v, { shouldDirty: true })}
              options={visibilities.map((v) => ({
                value: v,
                label: visibilityLabels[v],
                description: visibilityDescriptions[v],
              }))}
              error={errors.visibility?.message}
            />
            {type !== "bool" ? (
              <div>
                <Checkbox
                  checked={watch("required")}
                  onCheckedChange={(c) => setValue("required", c, { shouldDirty: true })}
                >
                  Obligatoire
                </Checkbox>
                {errors.required?.message ? (
                  <p className="mt-1 text-sm font-bold text-danger">{errors.required.message}</p>
                ) : null}
              </div>
            ) : null}
            <Field
              label="Texte d'aide (facultatif)"
              maxLength={200}
              autoComplete="off"
              hint="Affiché sous le champ dans les formulaires."
              error={errors.help?.message}
              {...register("help")}
            />
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Dialog.Close render={<Button variant="secondary">Annuler</Button>} />
              <Button type="submit" aria-disabled={formState.isSubmitting}>
                {field ? "Enregistrer" : "Ajouter"}
              </Button>
            </div>
          </form>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
