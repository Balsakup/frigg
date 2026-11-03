import { describe, expect, it } from "vitest";
import {
  amountInput,
  formatAge,
  formatAmount,
  formatBytes,
  formatDay,
  formatIban,
  formatPhone,
  membershipKindLabel,
  membershipStatusLabel,
  paymentMethodLabel,
  paymentStatusLabel,
  previousSeasonLabel,
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

describe("adhésions", () => {
  it("traduit les états, types et moyens", () => {
    expect(membershipStatusLabel("pending")).toBe("En attente de validation");
    expect(membershipStatusLabel("cancelled")).toBe("Annulée");
    expect(membershipKindLabel("renewal")).toBe("Renouvellement");
    expect(paymentMethodLabel("transfer")).toBe("Virement");
    expect(previousSeasonLabel("unknown")).toBe("Inconnu");
  });

  it("décrit l'état du paiement", () => {
    expect(paymentStatusLabel({ status: "due" })).toBe("À payer");
    expect(paymentStatusLabel({ status: "announced", announcedMethod: "cheque" })).toBe(
      "Paiement annoncé — chèque",
    );
    expect(paymentStatusLabel({ status: "paid", paidOn: "2026-11-02" })).toBe(
      "Payée le 2 novembre 2026",
    );
  });

  it("groupe l'IBAN par quatre caractères", () => {
    expect(formatIban("FR7630006000011234567890189")).toBe("FR76 3000 6000 0112 3456 7890 189");
  });
});

describe("formatBytes", () => {
  it("formate les tailles de fichier en français", () => {
    expect(formatBytes(1)).toBe("1 octet");
    expect(formatBytes(850)).toBe("850 octets");
    expect(formatBytes(12 * 1024)).toBe("12 Ko");
    expect(formatBytes(1.2 * 1024 * 1024)).toBe("1,2 Mo");
    expect(formatBytes(10 * 1024 * 1024)).toBe("10,0 Mo");
  });
});
