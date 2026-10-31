import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router";
import { applyProblemToForm, isProblem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import { invalidateSeasons, type SeasonList, seasonListQuery } from "~/api/seasons";
import { Page } from "~/shell/Page";
import { Alert } from "~/ui/Alert";
import { Button, buttonClasses } from "~/ui/Button";
import { SeasonDatesFields, type SeasonDatesForm } from "./SeasonDatesFields";

export const handle = { crumb: "Nouvelle saison" };

export async function clientLoader() {
  await queryClient.ensureQueryData(seasonListQuery()).catch(() => undefined);
  return null;
}

// Création d'une saison (US1, US2) : début libre pour la première, imposé
// ensuite (lendemain de la dernière fin, FR-007).
export default function SeasonNew() {
  const list = useQuery(seasonListQuery());
  if (list.isError) {
    return (
      <Alert kind="danger" role="alert" className="max-w-3xl">
        Impossible de charger les saisons. Rechargez la page.
      </Alert>
    );
  }
  if (!list.data) return null;
  return <SeasonNewForm list={list.data} />;
}

function SeasonNewForm({ list }: { list: SeasonList }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const create = $api.useMutation("post", "/admin/seasons");
  const { register, handleSubmit, setError, formState } = useForm<SeasonDatesForm>({
    defaultValues: { startsOn: "", endsOn: "" },
  });
  const imposed = list.nextStartsOn;
  const latest = list.items?.[0];

  const onSubmit = handleSubmit(async (form) => {
    try {
      const season = await create.mutateAsync({
        body: { startsOn: imposed ?? form.startsOn, endsOn: form.endsOn },
      });
      await invalidateSeasons(qc);
      navigate(`/admin/saisons/${season.id}`, {
        state: { flash: `Saison ${season.label} créée.` },
      });
    } catch (error) {
      if (!isProblem(error)) throw error;
      applyProblemToForm(error, setError, ["startsOn", "endsOn"]);
      // Liste changée entre-temps : le début imposé est relu.
      if (error.status === 409) await invalidateSeasons(qc);
    }
  });

  return (
    <Page
      title="Nouvelle saison"
      intro="Une saison délimite une période d'activité. Elle est créée en brouillon : vous l'ouvrirez quand elle sera prête."
    >
      <section className="max-w-3xl rounded-lg border border-line bg-surface p-5">
        {formState.errors.root?.server ? (
          <Alert kind="danger" role="alert" className="mb-4">
            {formState.errors.root.server.message}
          </Alert>
        ) : null}
        <form onSubmit={onSubmit} noValidate className="flex flex-col gap-5">
          <SeasonDatesFields
            register={register}
            errors={formState.errors}
            startsOn={
              imposed && latest
                ? {
                    value: imposed,
                    hint: `Le lendemain de la fin de la saison ${latest.label}.`,
                  }
                : "editable"
            }
            startsOnHint="Elle peut être passée si la saison a déjà commencé."
            endsOnMin={list.nextEndsOnMin ?? list.today}
            endsOnMax={list.nextEndsOnMax ?? undefined}
          />
          <div className="flex flex-wrap gap-2">
            <Button type="submit" aria-disabled={formState.isSubmitting}>
              Créer la saison
            </Button>
            <Link to="/admin/saisons" className={buttonClasses({ variant: "secondary" })}>
              Annuler
            </Link>
          </div>
        </form>
      </section>
    </Page>
  );
}
