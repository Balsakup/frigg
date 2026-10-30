import { describe, expect, it } from "vitest";
import { cn } from "./cn";

describe("cn", () => {
  it("ignore les valeurs conditionnelles fausses", () => {
    expect(cn("px-4", false && "hidden", undefined, { "font-bold": true })).toBe("px-4 font-bold");
  });

  it("garde la dernière classe d'un même groupe", () => {
    expect(cn("bg-brand", "bg-brand/10")).toBe("bg-brand/10");
    expect(cn("text-muted", "text-brand-ink")).toBe("text-brand-ink");
    expect(cn("border-line", "border-line-strong")).toBe("border-line-strong");
  });

  it("ne confond pas les groupes voisins des tokens du design system", () => {
    expect(cn("font-bold", "font-display")).toBe("font-bold font-display");
    expect(cn("text-sm", "text-brand-ink")).toBe("text-sm text-brand-ink");
  });
});
