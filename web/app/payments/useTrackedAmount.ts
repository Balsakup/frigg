import { useState } from "react";
import { amountInput } from "~/lib/format";

// Montant à payer saisi par un administrateur (contracts/ui-routes.md 014 ›
// AmountToPayField). Tant qu'il est « suivi », le champ affiche le montant
// théorique fourni par l'API et le suit quand le tarif ou le type change ; il
// cesse de l'être à la première saisie. Le frontend ne compare ni n'analyse
// les montants : le serveur décide (constitution VI).

export interface InitialAmount {
  amountCents: number;
  adjustmentReason?: string | null;
}

export interface TrackedAmount {
  // Texte affiché dans le champ.
  text: string;
  reason: string;
  tracked: boolean;
  setText: (text: string) => void;
  setReason: (reason: string) => void;
  // Revenir au montant théorique (suivi rétabli, motif effacé).
  reset: () => void;
  // Réinitialiser à l'ouverture d'un formulaire.
  restart: (initial?: InitialAmount | null) => void;
  // Champs à envoyer : rien quand le champ est suivi (le serveur applique le
  // montant théorique).
  payload: () => { amount?: string; adjustmentReason?: string };
}

interface State {
  text: string;
  reason: string;
  tracked: boolean;
}

function initialState(initial?: InitialAmount | null): State {
  if (initial?.adjustmentReason) {
    return {
      text: amountInput(initial.amountCents),
      reason: initial.adjustmentReason,
      tracked: false,
    };
  }
  return { text: "", reason: "", tracked: true };
}

export function useTrackedAmount(
  theoreticalCents: number | undefined,
  initial?: InitialAmount | null,
): TrackedAmount {
  const [state, setState] = useState<State>(() => initialState(initial));
  const text = state.tracked
    ? theoreticalCents === undefined
      ? ""
      : amountInput(theoreticalCents)
    : state.text;
  return {
    text,
    reason: state.reason,
    tracked: state.tracked,
    setText: (t) => setState((s) => ({ ...s, text: t, tracked: false })),
    setReason: (r) => setState((s) => ({ ...s, reason: r })),
    reset: () => setState({ text: "", reason: "", tracked: true }),
    restart: (i) => setState(initialState(i)),
    payload: () => (state.tracked ? {} : { amount: state.text, adjustmentReason: state.reason }),
  };
}
