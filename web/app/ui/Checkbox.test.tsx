import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { CheckboxGroup } from "./Checkbox";

function Group() {
  const [value, setValue] = useState<("a" | "b")[]>(["a"]);
  return (
    <>
      <CheckboxGroup
        legend="Situation de paiement"
        hint="Au moins une."
        value={value}
        onValueChange={setValue}
        options={[
          { value: "a", label: "À jour" },
          { value: "b", label: "À payer" },
        ]}
      />
      <output>{value.join(",")}</output>
    </>
  );
}

describe("CheckboxGroup", () => {
  it("expose un groupe nommé par sa légende et reflète l'état coché", () => {
    render(<Group />);
    const group = screen.getByRole("group", { name: "Situation de paiement" });
    expect(group.getAttribute("aria-describedby")).toBeTruthy();
    const [first, second] = screen.getAllByRole("checkbox");
    expect(first.getAttribute("aria-checked")).toBe("true");
    expect(second.getAttribute("aria-checked")).toBe("false");
  });

  it("coche une case focalisable et met la valeur à jour", () => {
    render(<Group />);
    const second = screen.getAllByRole("checkbox")[1];
    second.focus();
    expect(document.activeElement).toBe(second);
    fireEvent.click(second);
    expect(second.getAttribute("aria-checked")).toBe("true");
    expect(screen.getByRole("status").textContent).toBe("a,b");
  });
});
