import { describe, expect, it } from "vitest";
import type { FieldValue, FormField } from "~/api/custom-fields";
import { adminFormFields, customFieldPaths, toFormValues, toInputs } from "./form";

const base = { help: null, required: false, editable: true, choices: [] };

const values: FieldValue[] = [
  {
    ...base,
    id: 1,
    label: "Licence",
    type: "short_text",
    text: "1234567",
    number: null,
    date: null,
    checked: null,
    choiceId: null,
  },
  {
    ...base,
    id: 2,
    label: "Pointure",
    type: "number",
    text: null,
    number: 42.5,
    date: null,
    checked: null,
    choiceId: null,
  },
  {
    ...base,
    id: 3,
    label: "Certificat",
    type: "date",
    text: null,
    number: null,
    date: "2026-09-15",
    checked: null,
    choiceId: null,
  },
  {
    ...base,
    id: 4,
    label: "Bénévole",
    type: "bool",
    text: null,
    number: null,
    date: null,
    checked: true,
    choiceId: null,
  },
  {
    ...base,
    id: 5,
    label: "Niveau",
    type: "choice",
    choices: [{ id: 9, label: "Débutant" }],
    text: null,
    number: null,
    date: null,
    checked: null,
    choiceId: 9,
  },
  {
    ...base,
    id: 6,
    label: "Groupe",
    type: "short_text",
    editable: false,
    text: "A",
    number: null,
    date: null,
    checked: null,
    choiceId: null,
  },
];

describe("champs personnalisés dans les formulaires", () => {
  it("convertit les valeurs pour le formulaire (virgule décimale, choix en chaîne)", () => {
    expect(toFormValues(values)).toEqual({
      f1: "1234567",
      f2: "42,5",
      f3: "2026-09-15",
      f4: true,
      f5: "9",
      f6: "A",
    });
  });

  it("initialise les définitions d'un formulaire de création", () => {
    const defs: FormField[] = [
      { ...base, id: 1, label: "Licence", type: "short_text" },
      { ...base, id: 4, label: "Bénévole", type: "bool" },
      { ...base, id: 5, label: "Niveau", type: "choice" },
    ];
    expect(toFormValues(defs)).toEqual({ f1: "", f4: false, f5: "" });
  });

  it("n'envoie que les champs modifiables, choix vide sans choiceId", () => {
    const inputs = toInputs(values, {
      f1: "7654321",
      f2: "12,5",
      f3: "",
      f4: false,
      f5: "",
      f6: "B",
    });
    expect(inputs).toEqual([
      { fieldId: 1, text: "7654321" },
      { fieldId: 2, text: "12,5" },
      { fieldId: 3, date: "" },
      { fieldId: 4, checked: false },
      { fieldId: 5 },
    ]);
  });

  it("donne les chemins d'erreur et les définitions actives de l'administration", () => {
    expect(customFieldPaths(values.slice(0, 2))).toEqual(["customFields.f1", "customFields.f2"]);
    const list = {
      member: [
        {
          id: 1,
          scope: "member" as const,
          label: "A",
          type: "date" as const,
          visibility: "admin" as const,
          required: true,
          help: null,
          choices: [],
          archived: false,
          valueCount: 0,
          missingCount: null,
        },
        {
          id: 2,
          scope: "member" as const,
          label: "B",
          type: "date" as const,
          visibility: "admin" as const,
          required: false,
          help: null,
          choices: [],
          archived: true,
          valueCount: 0,
          missingCount: null,
        },
      ],
      membership: [],
    };
    expect(adminFormFields(list, "member")).toEqual([
      { id: 1, label: "A", type: "date", help: null, required: true, editable: true, choices: [] },
    ]);
  });
});
