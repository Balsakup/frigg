import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { readFragmentToken, useFragmentToken } from "./useFragmentToken";

afterEach(() => {
  window.history.replaceState(null, "", "/");
});

describe("useFragmentToken", () => {
  it("lit le jeton du fragment puis l'efface en gardant chemin et requête", () => {
    window.history.replaceState(null, "", "/suivi/adhesions/3?tentative=7#jeton=abc_DEF-123");
    const { result, rerender } = renderHook(() => useFragmentToken());
    expect(result.current).toBe("abc_DEF-123");
    expect(window.location.hash).toBe("");
    expect(window.location.pathname).toBe("/suivi/adhesions/3");
    expect(window.location.search).toBe("?tentative=7");
    rerender();
    expect(result.current).toBe("abc_DEF-123");
  });

  it("renvoie null sans fragment", () => {
    window.history.replaceState(null, "", "/suivi");
    const { result } = renderHook(() => useFragmentToken());
    expect(result.current).toBeNull();
  });

  it("ignore un fragment sans jeton", () => {
    expect(readFragmentToken("#autre=1")).toBeNull();
    expect(readFragmentToken("#jeton=")).toBeNull();
    expect(readFragmentToken("")).toBeNull();
  });
});
