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
});
