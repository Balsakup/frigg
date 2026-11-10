import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ImportAnalysis } from "~/api/memberImport";
import { MappingStep } from "./MappingStep";

const analysis: ImportAnalysis = {
  file: { encoding: "utf-8", separator: ";", rowCount: 2 },
  columns: [
    { index: 0, header: "Prénom", label: "Prénom", samples: ["Léa"] },
    { index: 1, header: "Nom", label: "Nom", samples: ["Lefèvre"] },
    { index: 2, header: "Mail", label: "Mail", samples: [] },
  ],
  suggested: ["firstName", "lastName", ""],
  targets: [
    { target: "firstName", label: "Prénom", requiredForImport: true, fieldType: null },
    { target: "lastName", label: "Nom", requiredForImport: true, fieldType: null },
    { target: "birthDate", label: "Date de naissance", requiredForImport: true, fieldType: null },
    { target: "email", label: "Adresse électronique", requiredForImport: false, fieldType: null },
  ],
  fieldsVersion: "0".repeat(32),
};

function Harness({
  errors = {},
  onPreview = () => {},
}: {
  errors?: Record<string, string[]>;
  onPreview?: () => void;
}) {
  const [mapping, setMapping] = useState(["firstName", "lastName", ""]);
  return (
    <MappingStep
      fileName="membres.csv"
      analysis={analysis}
      mapping={mapping}
      onChange={setMapping}
      errors={errors}
      busy={false}
      onBack={() => {}}
      onPreview={onPreview}
    />
  );
}

async function openSelect(name: string) {
  const trigger = screen.getByRole("combobox", { name });
  await act(async () => {
    fireEvent.click(trigger);
  });
  return screen.findByRole("listbox");
}

describe("MappingStep", () => {
  it("désactive une cible prise par une autre colonne", async () => {
    render(<Harness />);
    const list = await openSelect("Champ Frigg pour la colonne Mail");
    const taken = within(list).getByRole("option", { name: /^Prénom \*/ });
    expect(taken.getAttribute("aria-disabled")).toBe("true");
    expect(taken.textContent).toContain("(colonne « Prénom »)");
    const free = within(list).getByRole("option", { name: /^Adresse électronique/ });
    expect(free.getAttribute("aria-disabled")).not.toBe("true");
  });

  it("signale un champ exigé non associé et bloque l'aperçu", () => {
    const onPreview = vi.fn();
    render(<Harness onPreview={onPreview} />);
    expect(screen.getByText("Associez une colonne au champ « Date de naissance ».")).toBeTruthy();
    const button = screen.getByRole("button", { name: "Voir l'aperçu" });
    expect(button.getAttribute("aria-disabled")).toBe("true");
    expect(button.getAttribute("aria-describedby")).toBeTruthy();
    fireEvent.click(button);
    expect(onPreview).not.toHaveBeenCalled();
  });

  it("affiche une erreur du serveur sous le bon Select", () => {
    render(
      <Harness
        errors={{ "form.mapping[1]": ["Ce champ n'existe plus ou n'est pas importable."] }}
      />,
    );
    const trigger = screen.getByRole("combobox", { name: "Champ Frigg pour la colonne Nom" });
    const describedBy = trigger.getAttribute("aria-describedby") ?? "";
    expect(document.getElementById(describedBy)?.textContent).toContain("n'existe plus");
    expect(
      screen
        .getByRole("combobox", { name: "Champ Frigg pour la colonne Prénom" })
        .getAttribute("aria-describedby"),
    ).toBeNull();
  });
});
