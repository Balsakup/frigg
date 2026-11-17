import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Honeypot } from "./Honeypot";

describe("Honeypot", () => {
  it("est absent de l'arbre d'accessibilité et de l'ordre de tabulation", () => {
    render(
      <form>
        <label htmlFor="nom">Nom</label>
        <input id="nom" />
        <Honeypot />
      </form>,
    );
    expect(screen.getAllByRole("textbox")).toHaveLength(1);
    expect(screen.queryByRole("textbox", { name: "Site web" })).toBeNull();
    const trap = screen.getByRole("textbox", { name: "Site web", hidden: true });
    expect(trap.getAttribute("name")).toBe("website");
    expect(trap.tabIndex).toBe(-1);
    expect(trap.getAttribute("autocomplete")).toBe("off");
    expect(trap.closest("[aria-hidden='true']")).not.toBeNull();
  });
});
