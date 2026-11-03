import { describe, expect, it } from "vitest";
import {
  parseMembershipFilters,
  toMembershipApiQuery,
  toMembershipSearchParams,
} from "./membership-filters";

describe("critères des adhésions", () => {
  it("lit l'URL en ignorant les valeurs inconnues", () => {
    const f = parseMembershipFilters(
      new URLSearchParams(
        "season=all&status=pending&payment=x&paidMethod=bitcoin&alerts=all&page=0",
      ),
    );
    expect(f).toEqual({
      season: "all",
      status: "pending",
      payment: "",
      paidMethod: "",
      alerts: "",
      q: "",
      page: 1,
    });
  });

  it("filtre par moyen constaté et par signalement", () => {
    const f = parseMembershipFilters(new URLSearchParams("paidMethod=card&alerts=open"));
    expect(toMembershipApiQuery(f)).toEqual({ paidMethod: "card", alerts: "open", page: 1 });
    expect(toMembershipSearchParams(f).toString()).toBe("paidMethod=card&alerts=open");
  });

  it("n'envoie pas une recherche trop courte", () => {
    const f = parseMembershipFilters(new URLSearchParams("q=ab&season=3"));
    expect(toMembershipApiQuery(f)).toEqual({ season: "3", page: 1 });
  });

  it("reconstruit l'URL", () => {
    const f = parseMembershipFilters(new URLSearchParams("q=0042&payment=due&page=2"));
    expect(toMembershipSearchParams(f).toString()).toBe("payment=due&q=0042&page=2");
  });
});
