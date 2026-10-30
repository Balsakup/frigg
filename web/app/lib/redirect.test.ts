import { describe, expect, it } from "vitest";
import { safeRedirect } from "./redirect";

describe("safeRedirect", () => {
  it("accepte un chemin interne", () => {
    expect(safeRedirect("/admin/comptes?page=2")).toBe("/admin/comptes?page=2");
  });

  it("refuse les destinations externes ou vides", () => {
    for (const value of [
      null,
      undefined,
      "",
      "https://exemple.fr",
      "//exemple.fr",
      "/\\exemple.fr",
      "admin",
    ]) {
      expect(safeRedirect(value)).toBeNull();
    }
  });
});
