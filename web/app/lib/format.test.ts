import { describe, expect, it } from "vitest";
import { formatDay, seasonStatusLabel } from "./format";

describe("formatDay", () => {
  it("affiche la date civile sans décalage de fuseau", () => {
    expect(formatDay("2026-09-01")).toBe("1 septembre 2026");
    expect(formatDay("2027-08-31")).toBe("31 août 2027");
  });

  it("construit la date en heure locale, quel que soit le fuseau", () => {
    // new Date("2026-09-01") vaudrait le 31 août à l'ouest de Greenwich ;
    // formatDay ne passe pas par l'analyse d'une chaîne ISO.
    const [y, m, d] = "2026-01-01".split("-").map(Number);
    expect(new Date(y, m - 1, d).getDate()).toBe(1);
    expect(formatDay("2026-01-01")).toBe("1 janvier 2026");
  });
});

describe("seasonStatusLabel", () => {
  it("traduit les états", () => {
    expect(seasonStatusLabel("draft")).toBe("Brouillon");
    expect(seasonStatusLabel("open")).toBe("Ouverte");
    expect(seasonStatusLabel("closed")).toBe("Close");
  });
});
