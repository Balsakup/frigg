import type {
  CustomFieldList,
  FieldDefinition,
  FieldInput,
  FieldScope,
  FieldValue,
  FormField,
} from "~/api/custom-fields";

// Valeurs des champs personnalisés dans un formulaire react-hook-form, sous
// `customFields.f<id>` : le serveur place ses erreurs sur
// `body.customFields.f<id>` (contracts/api.md 011).
export type CustomFieldFormValues = Record<string, string | boolean>;

export interface WithCustomFields {
  customFields: CustomFieldFormValues;
}

export function fieldKey(id: number) {
  return `f${id}`;
}

export function customFieldPath(id: number) {
  return `customFields.${fieldKey(id)}` as const;
}

// Chemins d'erreur des champs, pour applyProblemToForm.
export function customFieldPaths(fields: FormField[]) {
  return fields.map((f) => customFieldPath(f.id));
}

function isValue(f: FormField | FieldValue): f is FieldValue {
  return "text" in f || "checked" in f || "choiceId" in f || "number" in f || "date" in f;
}

// Saisie d'un nombre : virgule décimale, sans séparateur de milliers, pour
// que la valeur renvoyée telle quelle soit relue par le serveur.
const numberInput = new Intl.NumberFormat("fr-FR", {
  useGrouping: false,
  maximumFractionDigits: 10,
});

export function toFormValues(fields: (FormField | FieldValue)[]): CustomFieldFormValues {
  const out: CustomFieldFormValues = {};
  for (const f of fields) {
    const key = fieldKey(f.id);
    const v = isValue(f) ? f : undefined;
    switch (f.type) {
      case "bool":
        out[key] = v?.checked ?? false;
        break;
      case "number":
        out[key] = v?.number != null ? numberInput.format(v.number) : "";
        break;
      case "date":
        out[key] = v?.date ?? "";
        break;
      case "choice":
        out[key] = v?.choiceId != null ? String(v.choiceId) : "";
        break;
      default:
        out[key] = v?.text ?? "";
    }
  }
  return out;
}

// Valeurs envoyées à l'API : seulement les champs modifiables dans l'espace.
export function toInputs(
  fields: FormField[],
  values: CustomFieldFormValues | undefined,
): FieldInput[] {
  const out: FieldInput[] = [];
  for (const f of fields) {
    if (!f.editable) continue;
    const raw = values?.[fieldKey(f.id)];
    switch (f.type) {
      case "bool":
        out.push({ fieldId: f.id, checked: raw === true });
        break;
      case "date":
        out.push({ fieldId: f.id, date: typeof raw === "string" ? raw : "" });
        break;
      case "choice":
        out.push(raw ? { fieldId: f.id, choiceId: Number(raw) } : { fieldId: f.id });
        break;
      default:
        out.push({ fieldId: f.id, text: typeof raw === "string" ? raw : "" });
    }
  }
  return out;
}

// Définitions actives d'une portée, vues par l'administration (tout modifiable).
export function adminFormFields(list: CustomFieldList | undefined, scope: FieldScope): FormField[] {
  return (list?.[scope] ?? [])
    .filter((f: FieldDefinition) => !f.archived)
    .map((f) => ({
      id: f.id,
      label: f.label,
      type: f.type,
      help: f.help,
      required: f.required,
      editable: true,
      choices: f.choices,
    }));
}
