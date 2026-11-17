import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { isProblem } from "~/api/problem";
import { invalidateTracking, useOpenTrackingLink } from "~/api/tracking";
import { useFragmentToken } from "./useFragmentToken";

export type TrackingAccess =
  | { kind: "ready" }
  | { kind: "opening" }
  | { kind: "gone" }
  | { kind: "error"; message: string };

// Accès aux pages de suivi (016, US2) : si l'adresse porte un jeton
// (`#jeton=…`, lien d'e-mail), il est effacé de la barre d'adresse puis envoyé
// une seule fois dans le corps de `open-tracking-link` ; les lectures par lien
// partent ensuite. Sans jeton, l'accès déjà ouvert en session est utilisé.
export function useTrackingAccess(): TrackingAccess {
  const token = useFragmentToken();
  const open = useOpenTrackingLink();
  const qc = useQueryClient();
  const [state, setState] = useState<TrackingAccess>(
    token ? { kind: "opening" } : { kind: "ready" },
  );
  const started = useRef(false);

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true;
    open
      .mutateAsync({ body: { token } })
      .then(async () => {
        await invalidateTracking(qc);
        setState({ kind: "ready" });
      })
      .catch((e: unknown) => {
        if (isProblem(e) && e.status === 410) {
          setState({ kind: "gone" });
          return;
        }
        setState({
          kind: "error",
          message:
            (isProblem(e) && e.detail) ||
            "Une erreur inattendue s'est produite. Réessayez dans quelques minutes.",
        });
      });
  }, [token, open, qc]);

  return state;
}
