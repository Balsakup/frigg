import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { identityQuery } from "~/api/identity";
import { BrandStyle, triplet } from "./BrandStyle";

describe("BrandStyle", () => {
  afterEach(() => {
    document.documentElement.removeAttribute("style");
  });

  it("convertit un code couleur en triplet", () => {
    expect(triplet("#1f5278")).toBe("31 82 120");
  });

  it("n'écrit que les six variables de marque", () => {
    const qc = new QueryClient();
    qc.setQueryData(identityQuery.queryKey, {
      name: "Club",
      logoUrl: null,
      palette: {
        brand: "#fff3b0",
        brandInk: "#7a6f2f",
        onBrand: "#181f29",
        accent: "#000000",
        accentInk: "#000000",
        onAccent: "#ffffff",
      },
    });
    render(
      <QueryClientProvider client={qc}>
        <BrandStyle />
      </QueryClientProvider>,
    );
    const style = document.documentElement.style;
    expect(style.getPropertyValue("--brand")).toBe("255 243 176");
    expect(style.getPropertyValue("--brand-ink")).toBe("122 111 47");
    expect(style.getPropertyValue("--on-brand")).toBe("24 31 41");
    expect(style.getPropertyValue("--on-accent")).toBe("255 255 255");
    const written = Array.from({ length: style.length }, (_, i) => style.item(i)).sort();
    expect(written).toEqual(
      ["--accent", "--accent-ink", "--brand", "--brand-ink", "--on-accent", "--on-brand"].sort(),
    );
    for (const semantic of ["--ok", "--warn", "--danger", "--info"]) {
      expect(style.getPropertyValue(semantic)).toBe("");
    }
  });
});
