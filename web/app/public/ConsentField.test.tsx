import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { ConsentField } from "./ConsentField";

function Consent({ error }: { error?: string }) {
  const [checked, setChecked] = useState(false);
  return (
    <ConsentField
      association="Les Sentiers du Val"
      purpose="membership"
      checked={checked}
      onCheckedChange={setChecked}
      error={error}
    />
  );
}

describe("ConsentField", () => {
  it("nomme l'association et la finalité, et se coche au clic sur le texte", () => {
    render(<Consent />);
    const box = screen.getByRole("checkbox", {
      name: /Les Sentiers du Val.*adhésion.*7 jours.*12 mois/,
    });
    expect(box.getAttribute("aria-checked")).toBe("false");
    fireEvent.click(screen.getByText(/J'accepte que/));
    expect(box.getAttribute("aria-checked")).toBe("true");
  });

  it("relie l'erreur à la case", () => {
    render(<Consent error="Vous devez accepter le traitement de vos données." />);
    const box = screen.getByRole("checkbox");
    const describedBy = box.getAttribute("aria-describedby") ?? "";
    expect(document.getElementById(describedBy)?.textContent).toBe(
      "Vous devez accepter le traitement de vos données.",
    );
    expect(box.getAttribute("aria-invalid")).toBe("true");
  });
});
