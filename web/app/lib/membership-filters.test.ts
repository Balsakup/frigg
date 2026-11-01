import { describe, expect, it } from "vitest";
import {
  parseMembershipFilters,
  toMembershipApiQuery,
  toMembershipSearchParams,
} from "./membership-filters";

describe("critères des adhésions", () => {
  it("lit l'URL en ignorant les valeurs inconnues", () => {
    const f = parseMembershipFilters(
      new URLSearchParams("season=all&status=pending&payment=x&page=0"),
    );
    expect(f).toEqual({ season: "all", status: "pending", payment: "", q: "", page: 1 });
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
