import { describe, expect, it } from "vitest";
import { entriesFor, entryForPath, navigation, spaceFromPath } from "./navigation";

describe("navigation", () => {
  it("déduit l'espace de l'URL", () => {
    expect(spaceFromPath("/admin")).toBe("admin");
    expect(spaceFromPath("/admin/comptes")).toBe("admin");
    expect(spaceFromPath("/")).toBe("member");
    expect(spaceFromPath("/administration")).toBe("member");
    expect(spaceFromPath("/mon-compte")).toBe("member");
  });

  it("sépare les rubriques des deux espaces", () => {
    expect(entriesFor("member", "main").every((e) => !e.path.startsWith("/admin"))).toBe(true);
    expect(entriesFor("admin", "main").every((e) => e.path.startsWith("/admin"))).toBe(true);
    expect(entriesFor("admin", "resources").map((e) => e.id)).toContain("comptes");
  });

  it("a des identifiants et des chemins uniques", () => {
    expect(new Set(navigation.map((e) => e.id)).size).toBe(navigation.length);
    expect(new Set(navigation.map((e) => e.path)).size).toBe(navigation.length);
    expect(entryForPath("/admin/finances")?.label).toBe("Finances");
  });
});
