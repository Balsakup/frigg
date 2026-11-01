import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { invalidateMemberships, membershipOfferQuery } from "~/api/memberships";
import { isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import type { MembershipKind } from "~/lib/format";
import { AmountSummary, FeeChoice } from "~/memberships/FeeChoice";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { RadioGroup } from "~/ui/RadioGroup";
import { Select } from "~/ui/Select";

export const handle = { crumb: "Demander une adhésion" };

export async function clientLoader() {
  await queryClient.ensureQueryData(membershipOfferQuery()).catch(() => undefined);
  return null;
}

type Errors = Partial<Record<"memberId" | "seasonId" | "feeId" | "declaredKind" | "root", string>>;

const kindOptions: { value: MembershipKind; label: string }[] = [
  { value: "first", label: "Non, première adhésion" },
  { value: "renewal", label: "Oui, renouvellement" },
];

// Demande d'adhésion d'une fiche du compte (006, US1). Les saisons, tarifs,
// montants et fiches disponibles viennent de l'offre : aucun calcul côté client.
export default function MembershipRequest() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const offer = useQuery(membershipOfferQuery());
  const create = $api.useMutation("post", "/me/memberships");
  const seasons = offer.data?.seasons ?? [];
  const members = offer.data?.members ?? [];

  const preselected = Number(params.get("membre")) || null;
  const [seasonId, setSeasonId] = useState<number | null>(null);
  const [memberId, setMemberId] = useState<number | null>(preselected);
  const [feeId, setFeeId] = useState<number | null>(null);
  const [kind, setKind] = useState<MembershipKind | "">("");
  const [errors, setErrors] = useState<Errors>({});

  const season = seasons.find((s) => s.id === seasonId) ?? seasons[0];
  const available = members.filter(
    (m) => season && (m.availableSeasonIds ?? []).includes(season.id),
  );
  const unavailable = members.filter(
    (m) => season && !(m.availableSeasonIds ?? []).includes(season.id),
  );
  const fee = season?.fees?.find((f) => f.id === feeId);
  const chosenMember = available.some((m) => m.id === memberId) ? memberId : null;

  if (offer.isPending) return null;
  if (!season || available.length === 0) {
    return (
      <Page title="Demander une adhésion">
        <Alert kind="info" className="max-w-3xl">
          {members.length === 0
            ? "Ajoutez d'abord un membre pour demander son adhésion."
            : seasons.length === 0
              ? "Les adhésions ne sont pas ouvertes pour le moment."
              : "Tous vos membres ont déjà une adhésion en cours pour les saisons ouvertes."}
        </Alert>
        <Link to="/adhesions" className={`${buttonClasses({ variant: "secondary" })} mt-4`}>
          Retour aux adhésions
        </Link>
      </Page>
    );
  }

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const missing: Errors = {};
    if (chosenMember === null) missing.memberId = "Choisissez le membre.";
    if (!fee) missing.feeId = "Choisissez un tarif.";
    if (!kind) missing.declaredKind = "Indiquez s'il s'agit d'un renouvellement.";
    setErrors(missing);
    if (Object.keys(missing).length > 0 || chosenMember === null || !fee || !kind) return;
    try {
      await create.mutateAsync({
        body: { memberId: chosenMember, seasonId: season.id, feeId: fee.id, declaredKind: kind },
      });
    } catch (err) {
      if (!isProblem(err)) throw err;
      const next: Errors = {};
      for (const fe of err.errors ?? []) {
        const field = fe.location?.replace("body.", "") as keyof Errors | undefined;
        if (field && field !== "root") next[field] = fe.message;
      }
      if (Object.keys(next).length === 0)
        next.root = err.detail ?? "Une erreur inattendue s'est produite.";
      setErrors(next);
      await qc.invalidateQueries({ queryKey: membershipOfferQuery().queryKey });
      return;
    }
    await invalidateMemberships(qc);
    navigate("/adhesions", { state: { flash: "Demande envoyée." } });
  };

  return (
    <Page title="Demander une adhésion">
      {errors.root ? (
        <Alert kind="danger" role="alert" className="mb-6 max-w-3xl">
          {errors.root}
        </Alert>
      ) : null}
      <form onSubmit={onSubmit} noValidate className="flex max-w-3xl flex-col gap-6">
        {seasons.length > 1 ? (
          <Select
            label="Saison"
            value={String(season.id)}
            items={seasons.map((s) => ({ value: String(s.id), label: s.label }))}
            onValueChange={(v) => {
              setSeasonId(Number(v));
              setFeeId(null);
            }}
            className="max-w-xs"
          />
        ) : (
          <p>
            <span className="font-bold">Saison : </span>
            {season.label}
          </p>
        )}

        <div>
          <RadioGroup
            legend="Membre"
            required
            value={chosenMember === null ? "" : String(chosenMember)}
            onValueChange={(v) => setMemberId(Number(v))}
            error={errors.memberId}
            options={available.map((m) => ({
              value: String(m.id),
              label: `${m.firstName} ${m.lastName}`,
            }))}
          />
          {unavailable.map((m) => (
            <p key={m.id} className="mt-2 text-sm text-muted">
              {m.firstName} a déjà une adhésion en cours pour cette saison.
            </p>
          ))}
        </div>

        <FeeChoice
          fees={season.fees ?? []}
          value={fee?.id ?? null}
          onChange={setFeeId}
          error={errors.feeId}
        />

        <RadioGroup
          legend="Était-il adhérent la saison passée ?"
          required
          value={kind}
          onValueChange={setKind}
          error={errors.declaredKind}
          options={kindOptions}
        />

        <div className="rounded-lg border border-line bg-surface p-4">
          <AmountSummary fee={fee} kind={kind} />
          <p className="text-muted">
            Votre demande sera validée par le bureau ; vous recevrez ensuite les informations de
            paiement.
          </p>
        </div>

        <div className="flex flex-col-reverse gap-2 sm:flex-row">
          <Link to="/adhesions" className={buttonClasses({ variant: "secondary" })}>
            Annuler
          </Link>
          <Button type="submit" aria-disabled={create.isPending}>
            Envoyer la demande
          </Button>
        </div>
      </form>
    </Page>
  );
}
