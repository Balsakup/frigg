import { describe, expect, it } from "vitest";
import {
  amountInput,
  formatAge,
  formatAmount,
  formatDay,
  formatPhone,
  seasonStatusLabel,
} from "./format";

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

describe("formatAmount", () => {
  it("affiche les euros au format français", () => {
    expect(formatAmount(1550)).toMatch(/^15,50\s€$/);
    expect(formatAmount(100000)).toMatch(/^1\s000,00\s€$/);
  });

  it("affiche « Gratuit » pour 0", () => {
    expect(formatAmount(0)).toBe("Gratuit");
  });
});

describe("amountInput", () => {
  it("préremplit le champ sans décimales inutiles", () => {
    expect(amountInput(3000)).toBe("30");
    expect(amountInput(1550)).toBe("15,50");
    expect(amountInput(5)).toBe("0,05");
  });
});

describe("formatPhone", () => {
  it("groupe un numéro français par paires", () => {
    expect(formatPhone("0612345678")).toBe("06 12 34 56 78");
  });

  it("laisse un numéro international tel quel", () => {
    expect(formatPhone("+32470123456")).toBe("+32470123456");
  });
});

describe("formatAge", () => {
  it("accorde l'âge", () => {
    expect(formatAge(0)).toBe("moins d'un an");
    expect(formatAge(1)).toBe("1 an");
    expect(formatAge(11)).toBe("11 ans");
  });
});
