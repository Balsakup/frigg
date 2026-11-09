import { useQueryClient } from "@tanstack/react-query";
import { useId, useState } from "react";
import { Link, useNavigate } from "react-router";
import { invalidateEvents, type MyEvent } from "~/api/events";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { formatAmount } from "~/lib/format";
import { Alert } from "~/ui/Alert";
import { Badge } from "~/ui/Badge";
import { Button } from "~/ui/Button";
import { Checkbox } from "~/ui/Checkbox";

const reasons = {
  not_season_member: (season: string) => `Pas d'adhésion validée pour la saison ${season}.`,
  already_registered: () => "Déjà inscrit(e).",
} as const;

// Inscription des fiches du compte (US3, US4) : une case par fiche gérée,
// raison affichée pour une fiche non éligible.
export function RegisterMembers({
  event: e,
  onStatus,
}: {
  event: MyEvent;
  onStatus: (m: string) => void;
}) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const register = $api.useMutation("post", "/me/events/{id}/registrations");
  const [selected, setSelected] = useState<number[]>([]);
  const [error, setError] = useState("");
  const [announce, setAnnounce] = useState("");
  const legend = useId();
  const full = e.placesLeft === 0;
  const eligible = (e.members ?? []).filter((m) => m.eligible);
  const total = e.priceCents * selected.length;

  const submit = async () => {
    setError("");
    if (selected.length === 0) {
      setError("Cochez au moins une personne à inscrire.");
      return;
    }
    try {
      const out = await register.mutateAsync({
        params: { path: { id: e.id } },
        body: { memberIds: selected },
      });
      setSelected([]);
      await invalidateEvents(qc);
      const first = out.createdRegistrationIds?.[0];
      if (e.priceCents > 0 && first) {
        navigate(`/mes-inscriptions/${first}`, {
          state: { flash: "Inscription enregistrée : il reste à payer." },
        });
        return;
      }
      onStatus(selected.length > 1 ? "Inscriptions enregistrées." : "Inscription enregistrée.");
    } catch (err) {
      const message = isProblem(err)
        ? err.errors?.map((x) => x.message).join(" ") || err.detail || ""
        : "Une erreur inattendue s'est produite.";
      setError(message);
      if (isProblem(err) && err.status === 409) setAnnounce(message);
      await invalidateEvents(qc);
    }
  };

  return (
    <div>
      <p className="sr-only" aria-live="polite">
        {announce}
      </p>
      {full ? (
        <Alert kind="warning" className="mb-4">
          Complet : il n'y a plus de place. Aucune liste d'attente n'est proposée.
        </Alert>
      ) : null}
      <fieldset aria-describedby={error ? `${legend}-erreur` : undefined}>
        <legend id={legend} className="mb-2 font-bold">
          Qui inscrire ?
        </legend>
        <ul className="flex flex-col gap-3">
          {(e.members ?? []).map((m) => {
            const name = `${m.member.firstName} ${m.member.lastName}`;
            const reasonId = `${legend}-${m.member.id}`;
            if (m.registration) {
              return (
                <li key={m.member.id} className="flex flex-wrap items-center gap-2">
                  <span className="font-bold">{name}</span>
                  <Badge tone="success">Inscrit(e)</Badge>
                  <Link
                    to={`/mes-inscriptions/${m.registration.id}`}
                    className="text-brand-ink underline underline-offset-2"
                  >
                    Voir l'inscription
                  </Link>
                </li>
              );
            }
            if (!m.eligible || full || !e.registrationOpen) {
              return (
                <li key={m.member.id} className="text-muted">
                  <span className="font-bold">{name}</span>
                  {m.reason ? (
                    <span id={reasonId} className="block text-sm">
                      {reasons[m.reason](e.season.label)}
                    </span>
                  ) : null}
                </li>
              );
            }
            return (
              <li key={m.member.id}>
                <Checkbox
                  checked={selected.includes(m.member.id)}
                  onCheckedChange={(c) =>
                    setSelected((s) =>
                      c ? [...s, m.member.id] : s.filter((x) => x !== m.member.id),
                    )
                  }
                >
                  {name}
                </Checkbox>
              </li>
            );
          })}
        </ul>
        {error ? (
          <p id={`${legend}-erreur`} role="alert" className="mt-2 text-sm font-bold text-danger">
            {error}
          </p>
        ) : null}
      </fieldset>
      {e.registrationOpen && !full && eligible.length > 0 ? (
        <div className="mt-4">
          <Button onClick={submit} aria-disabled={register.isPending}>
            {e.priceCents > 0 && selected.length > 0
              ? `Inscrire et payer ${formatAmount(total)}`
              : "Inscrire"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
