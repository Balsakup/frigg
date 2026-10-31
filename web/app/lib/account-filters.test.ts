import { describe, expect, it } from "vitest";
import { hasSearchableTerm, parseFilters, toApiQuery, toSearchParams } from "./account-filters";

describe("critères de la liste des comptes", () => {
  it("lit les valeurs françaises de l'URL", () => {
    expect(parseFilters(new URLSearchParams("q=pont&role=membre&statut=desactive&page=3"))).toEqual(
      {
        q: "pont",
        role: "member",
        status: "disabled",
        page: 3,
      },
    );
    expect(parseFilters(new URLSearchParams("role=admin&statut=actif"))).toMatchObject({
      role: "admin",
      status: "active",
    });
  });

  it("ignore les valeurs inconnues ou invalides", () => {
    expect(parseFilters(new URLSearchParams("role=owner&statut=x&page=0"))).toEqual({
      q: "",
      role: "",
      status: "",
      page: 1,
    });
    expect(parseFilters(new URLSearchParams("page=abc")).page).toBe(1);
  });

  it("reconnaît une saisie exploitable par la recherche", () => {
    expect(hasSearchableTerm("de pont")).toBe(true);
    expect(hasSearchableTerm("hél")).toBe(true);
    expect(hasSearchableTerm("du")).toBe(false);
    expect(hasSearchableTerm("a b")).toBe(false);
    expect(hasSearchableTerm("  ")).toBe(false);
  });

  it("n'envoie pas une recherche trop courte à l'API", () => {
    expect(toApiQuery({ q: "du", role: "", status: "", page: 1 })).toEqual({ page: 1 });
    expect(toApiQuery({ q: " pont ", role: "admin", status: "disabled", page: 2 })).toEqual({
      q: "pont",
      role: "admin",
      status: "disabled",
      page: 2,
    });
  });

  it("omet les valeurs par défaut dans l'URL", () => {
    expect(toSearchParams({ q: "", role: "", status: "", page: 1 }).toString()).toBe("");
    expect(toSearchParams({ q: "du", role: "member", status: "active", page: 2 }).toString()).toBe(
      "q=du&role=membre&statut=actif&page=2",
    );
  });
});
