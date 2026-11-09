import { describe, expect, it } from "vitest";
import { dateBlock, formatEventMoment, formatEventRange, formatHour } from "./date";

describe("dates des événements (heure de Paris)", () => {
  it("formate une plage sur un jour", () => {
    expect(formatEventRange("2026-12-12T13:00:00Z", "2026-12-12T16:00:00Z")).toBe(
      "sam. 12 déc. · 14 h – 17 h",
    );
  });
  it("formate une plage sur plusieurs jours", () => {
    expect(formatEventRange("2027-04-12T07:00:00Z", "2027-04-16T15:00:00Z")).toBe(
      "du lun. 12 avr. au ven. 16 avr.",
    );
  });
  it("garde les minutes et gère le passage à l'heure d'hiver", () => {
    expect(formatHour("2026-10-25T00:30:00Z")).toBe("2 h 30");
    expect(formatHour("2026-10-25T01:30:00Z")).toBe("2 h 30");
    expect(formatHour("2026-12-12T13:00:00Z")).toBe("14 h");
  });
  it("formate un instant en toutes lettres", () => {
    expect(formatEventMoment("2027-04-01T21:59:00Z")).toBe("jeudi 1er avril 2027 à 23 h 59");
  });
  it("découpe le bloc de date", () => {
    expect(dateBlock("2026-12-12T13:00:00Z")).toEqual(["12", "déc."]);
  });
});
