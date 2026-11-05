import { CircleAlert } from "lucide-react";
import { useId } from "react";
import { type Control, Controller, type FieldErrors } from "react-hook-form";
import type { FieldValue, FormField } from "~/api/custom-fields";
import { Checkbox } from "~/ui/Checkbox";
import { Field, TextAreaField } from "~/ui/Field";
import { Icon } from "~/ui/Icon";
import { Select } from "~/ui/Select";
import { displayValue } from "./CustomFieldValues";
import { customFieldPath, fieldKey, type WithCustomFields } from "./form";

interface CustomFieldInputsProps {
  fields: (FormField | FieldValue)[];
  control: Control<WithCustomFields>;
  errors: FieldErrors<WithCustomFields>;
  legend?: string;
}

// Champs personnalisés d'un formulaire (contracts/ui-routes.md 011 ›
// Composants) : un contrôle par type, libellé, aide et erreur reliés ; un
// champ non modifiable dans l'espace est affiché en texte.
export function CustomFieldInputs({
  fields,
  control,
  errors,
  legend = "Informations complémentaires",
}: CustomFieldInputsProps) {
  if (fields.length === 0) return null;
  return (
    <fieldset className="flex flex-col gap-4">
      <legend className="mb-2 font-display text-lg font-bold">{legend}</legend>
      {fields.map((f) => (
        <CustomFieldInput
          key={f.id}
          field={f}
          control={control}
          error={errors.customFields?.[fieldKey(f.id)]?.message}
        />
      ))}
    </fieldset>
  );
}

function ReadOnly({ field }: { field: FormField | FieldValue }) {
  const value = "text" in field ? displayValue(field as FieldValue) : null;
  return (
    <div>
      <p className="mb-1 font-bold [overflow-wrap:anywhere]">{field.label}</p>
      <p className="whitespace-pre-line [overflow-wrap:anywhere]">
        {value ?? <span className="text-muted">Non renseigné</span>}
      </p>
      {field.help ? <p className="mt-1 text-sm text-muted">{field.help}</p> : null}
    </div>
  );
}

function CustomFieldInput({
  field: f,
  control,
  error,
}: {
  field: FormField | FieldValue;
  control: Control<WithCustomFields>;
  error?: string;
}) {
  const id = useId();
  if (!f.editable) return <ReadOnly field={f} />;
  const help = f.help ?? undefined;
  const name = customFieldPath(f.id);
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => {
        switch (f.type) {
          case "long_text":
            return (
              <TextAreaField
                label={f.label}
                required={f.required}
                maxLength={2000}
                hint={help}
                error={error}
                name={field.name}
                ref={field.ref}
                value={String(field.value ?? "")}
                onChange={field.onChange}
                onBlur={field.onBlur}
              />
            );
          case "bool":
            return (
              <div>
                <Checkbox checked={field.value === true} onCheckedChange={field.onChange}>
                  <span className="font-bold">{f.label}</span>
                  {help ? <span className="mt-0.5 block text-sm text-muted">{help}</span> : null}
                </Checkbox>
                <FieldError id={`${id}-erreur`} message={error} />
              </div>
            );
          case "choice":
            return (
              <div>
                <Select
                  label={f.required ? `${f.label} (obligatoire)` : f.label}
                  value={String(field.value ?? "")}
                  onValueChange={field.onChange}
                  items={[
                    ...(f.required && field.value ? [] : [{ value: "", label: "—" }]),
                    ...(f.choices ?? []).map((c) => ({ value: String(c.id), label: c.label })),
                  ]}
                />
                {help ? <p className="mt-1 text-sm text-muted">{help}</p> : null}
                <FieldError id={`${id}-erreur`} message={error} />
              </div>
            );
          default:
            return (
              <Field
                label={f.label}
                required={f.required}
                type={f.type === "date" ? "date" : "text"}
                inputMode={f.type === "number" ? "decimal" : undefined}
                maxLength={f.type === "short_text" ? 200 : undefined}
                autoComplete="off"
                hint={help}
                error={error}
                className={f.type === "short_text" ? undefined : "max-w-56"}
                name={field.name}
                ref={field.ref}
                value={String(field.value ?? "")}
                onChange={field.onChange}
                onBlur={field.onBlur}
              />
            );
        }
      }}
    />
  );
}

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null;
  return (
    <p id={id} className="mt-1 flex items-start gap-1.5 text-sm font-bold text-danger">
      <Icon icon={CircleAlert} className="mt-0.5 h-4 w-4" />
      <span>{message}</span>
    </p>
  );
}
