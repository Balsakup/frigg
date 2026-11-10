import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useTrackedAmount } from "./useTrackedAmount";

describe("useTrackedAmount", () => {
  it("suit le montant théorique tant qu'il n'est pas saisi", () => {
    const { result, rerender } = renderHook(({ cents }) => useTrackedAmount(cents), {
      initialProps: { cents: 7500 },
    });
    expect(result.current.text).toBe("75");
    expect(result.current.tracked).toBe(true);
    expect(result.current.payload()).toEqual({});
    rerender({ cents: 4550 });
    expect(result.current.text).toBe("45,50");
  });

  it("cesse de suivre après une saisie, puis revient au théorique", () => {
    const { result, rerender } = renderHook(({ cents }) => useTrackedAmount(cents), {
      initialProps: { cents: 7500 },
    });
    act(() => result.current.setText("70"));
    act(() => result.current.setReason("réduction"));
    rerender({ cents: 4500 });
    expect(result.current.text).toBe("70");
    expect(result.current.tracked).toBe(false);
    expect(result.current.payload()).toEqual({ amount: "70", adjustmentReason: "réduction" });
    act(() => result.current.reset());
    expect(result.current.text).toBe("45");
    expect(result.current.reason).toBe("");
    expect(result.current.payload()).toEqual({});
  });

  it("démarre non suivi sur une adhésion déjà ajustée", () => {
    const { result } = renderHook(() =>
      useTrackedAmount(7500, { amountCents: 7000, adjustmentReason: "remise" }),
    );
    expect(result.current.tracked).toBe(false);
    expect(result.current.text).toBe("70");
    expect(result.current.reason).toBe("remise");
    act(() => result.current.restart(null));
    expect(result.current.tracked).toBe(true);
    expect(result.current.text).toBe("75");
  });
});
