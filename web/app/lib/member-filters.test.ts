import { describe, expect, it } from "vitest";
import {
  hasActiveMemberFilters,
  parseMemberFilters,
  toApiQuery,
  toMemberSearchParams,
} from "./member-filters";

describe("critères des membres", () => {
  it("lit et transmet les filtres par champ personnalisé (011)", () => {
    const f = parseMemberFilters(new URLSearchParams("champ=4:true&champ=7:empty&champ=bad"));
    expect(f.fields).toEqual(["4:true", "7:empty"]);
    expect(hasActiveMemberFilters(f)).toBe(true);
    expect(toApiQuery(f)).toEqual({ field: ["4:true", "7:empty"], page: 1 });
    expect(toMemberSearchParams(f).toString()).toBe("champ=4%3Atrue&champ=7%3Aempty");
  });

  it("sans filtre de champ, rien n'est envoyé", () => {
    const f = parseMemberFilters(new URLSearchParams(""));
    expect(hasActiveMemberFilters(f)).toBe(false);
    expect(toApiQuery(f)).toEqual({ page: 1 });
  });

  it("lit et transmet l'origine publique et les fiches inactives (016)", () => {
    const pub = parseMemberFilters(
      new URLSearchParams("origine=publique&rattachement=sans-compte"),
    );
    expect(pub.origin).toBe("public");
    expect(hasActiveMemberFilters(pub)).toBe(true);
    expect(toApiQuery(pub)).toEqual({ account: "none", origin: "public", page: 1 });
    expect(toMemberSearchParams(pub).toString()).toBe("rattachement=sans-compte&origine=publique");

    const inactive = parseMemberFilters(new URLSearchParams("origine=inactives"));
    expect(toApiQuery(inactive)).toEqual({ origin: "public", inactive: true, page: 1 });
    expect(toMemberSearchParams(inactive).toString()).toBe("origine=inactives");

    expect(parseMemberFilters(new URLSearchParams("origine=autre")).origin).toBe("");
  });
});
