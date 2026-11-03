import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { RotateCcw } from "lucide-react";
import { useEffect, useState } from "react";
import { useBlocker } from "react-router";
import { fallbackIdentity, type Palette } from "~/api/identity";
import { isProblem, type Problem } from "~/api/problem";
import { $api } from "~/api/queries";
import { queryClient } from "~/api/query-client";
import {
  type Appearance,
  appearanceQuery,
  invalidateAppearance,
  palettePreviewQuery,
  uploadLogo,
} from "~/api/settings";
import { AppearancePreview } from "~/settings/AppearancePreview";
import { ColorField } from "~/settings/ColorField";
import { LogoField } from "~/settings/LogoField";
import { useSession } from "~/shell/useSession";
import { Alert } from "~/ui/Alert";
import { Button } from "~/ui/Button";
import { ConfirmDialog } from "~/ui/ConfirmDialog";
import { Icon } from "~/ui/Icon";
import { SettingsPage } from "./SettingsPage";

export const handle = { crumb: "Apparence" };

export async function clientLoader() {
  await queryClient.ensureQueryData(appearanceQuery()).catch(() => undefined);
  return null;
}

const HEX = /^#[0-9a-fA-F]{6}$/;

// Paramètres › Apparence (010, US3) : logo et couleurs, avec un aperçu dont la
// palette est calculée par le serveur (garantie de contraste, R3).
export default function SettingsAppearance() {
  const appearance = useQuery(appearanceQuery());
  if (appearance.data) return <AppearanceView appearance={appearance.data} />;
  if (appearance.isError) {
    return (
      <SettingsPage section="Apparence">
        <Alert kind="danger" role="alert" className="max-w-3xl">
          Impossible de charger l'apparence. Rechargez la page.
        </Alert>
      </SettingsPage>
    );
  }
  return null;
}

function fieldErrors(problem: Problem, location: string) {
  return (problem.errors ?? [])
    .filter((e) => e.location === location)
    .map((e) => e.message)
    .join(" ");
}

function AppearanceView({ appearance }: { appearance: Appearance }) {
  const qc = useQueryClient();
  const { association } = useSession();
  const update = $api.useMutation("put", "/admin/settings/appearance");
  const remove = $api.useMutation("delete", "/admin/settings/appearance/logo");

  const [primary, setPrimary] = useState(appearance.primaryColor ?? appearance.palette.brand);
  const [accent, setAccent] = useState(appearance.accentColor ?? appearance.palette.accent);
  const [file, setFile] = useState<File | null>(null);
  const [fileUrl, setFileUrl] = useState<string | null>(null);
  const [errors, setErrors] = useState<{ file?: string; primary?: string; accent?: string }>({});
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);

  const savedPrimary = appearance.primaryColor ?? appearance.palette.brand;
  const savedAccent = appearance.accentColor ?? appearance.palette.accent;
  const dirty =
    file !== null ||
    primary.toLowerCase() !== savedPrimary.toLowerCase() ||
    accent.toLowerCase() !== savedAccent.toLowerCase();

  // Aperçu du fichier choisi, libéré quand il change.
  useEffect(() => {
    if (!file) {
      setFileUrl(null);
      return;
    }
    const url = URL.createObjectURL(file);
    setFileUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  // Palette d'aperçu demandée 150 ms après la dernière modification.
  const [debounced, setDebounced] = useState({ primary, accent });
  useEffect(() => {
    const timer = setTimeout(() => setDebounced({ primary, accent }), 150);
    return () => clearTimeout(timer);
  }, [primary, accent]);
  const previewable = HEX.test(debounced.primary) && HEX.test(debounced.accent);
  const preview = useQuery({
    ...palettePreviewQuery(debounced.primary, debounced.accent),
    enabled: previewable,
    placeholderData: keepPreviousData,
  });
  const palette: Palette = preview.data ?? appearance.palette ?? fallbackIdentity.palette;

  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      dirty && !busy && currentLocation.pathname !== nextLocation.pathname,
  );
  useEffect(() => {
    if (!dirty) return;
    const warn = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  const saveColors = async (p: string | null, a: string | null) => {
    try {
      await update.mutateAsync({ body: { primaryColor: p, accentColor: a } });
      return true;
    } catch (e) {
      if (!isProblem(e)) throw e;
      setErrors({
        primary: fieldErrors(e, "body.primaryColor") || undefined,
        accent: fieldErrors(e, "body.accentColor") || undefined,
      });
      return false;
    }
  };

  const onSave = async () => {
    if (busy) return;
    setBusy(true);
    setStatus("");
    setErrors({});
    try {
      // Logo d'abord : en cas de refus, rien d'autre n'est envoyé. Pas de
      // dialogue au remplacement (plan.md 010 › Complexity Tracking).
      if (file) {
        try {
          await uploadLogo(file);
        } catch (e) {
          if (!isProblem(e)) throw e;
          setErrors({ file: fieldErrors(e, "form.file") || e.detail || undefined });
          return;
        }
        setFile(null);
      }
      // Couleur égale à la valeur par défaut : rien n'est forcé.
      const p =
        primary.toLowerCase() === savedPrimary.toLowerCase() ? appearance.primaryColor : primary;
      const a =
        accent.toLowerCase() === savedAccent.toLowerCase() ? appearance.accentColor : accent;
      if (!(await saveColors(p ?? null, a ?? null))) return;
      await invalidateAppearance(qc);
      setStatus("Apparence enregistrée.");
    } finally {
      setBusy(false);
    }
  };

  const onRemoveLogo = async () => {
    await remove.mutateAsync({});
    setFile(null);
    await invalidateAppearance(qc);
    setStatus("Logo retiré.");
  };

  const onReset = async () => {
    if (!(await saveColors(null, null))) return;
    await invalidateAppearance(qc);
    const defaults = fallbackIdentity.palette;
    setPrimary(defaults.brand);
    setAccent(defaults.accent);
    setStatus("Couleurs par défaut rétablies.");
  };

  const shownLogo = fileUrl ?? appearance.logo?.url ?? null;

  return (
    <SettingsPage section="Apparence">
      <p role="status" className="sr-only">
        {status}
      </p>
      {status ? (
        <Alert kind="success" className="mb-6 max-w-5xl">
          {status}
        </Alert>
      ) : null}
      <div className="grid max-w-5xl gap-6 xl:grid-cols-[minmax(0,1fr)_22rem] xl:items-start">
        <div className="flex min-w-0 flex-col gap-6">
          <fieldset className="flex min-w-0 flex-col gap-4 rounded-lg border border-line bg-surface p-5">
            <legend className="px-1 font-display text-lg font-bold">Logo</legend>
            <LogoField
              name={association.name}
              logoUrl={shownLogo}
              pendingName={file?.name}
              error={errors.file}
              onSelect={(f) => {
                setFile(f);
                setErrors((e) => ({ ...e, file: undefined }));
              }}
              onRemove={appearance.logo ? () => setConfirmRemove(true) : undefined}
            />
          </fieldset>

          <fieldset className="flex min-w-0 flex-col gap-4 rounded-lg border border-line bg-surface p-5">
            <legend className="px-1 font-display text-lg font-bold">Couleurs</legend>
            <p className="text-sm text-muted">
              Les textes posés sur ces couleurs, ou écrits dans ces teintes, sont ajustés
              automatiquement pour rester lisibles.
            </p>
            <ColorField
              label="Couleur principale"
              value={primary}
              onChange={setPrimary}
              error={errors.primary}
              hint="Boutons, élément actif du menu, en-tête des e-mails."
            />
            <ColorField
              label="Couleur d'accent"
              value={accent}
              onChange={setAccent}
              error={errors.accent}
              hint="Mises en avant."
            />
            <div>
              <Button type="button" variant="tertiary" onClick={() => setConfirmReset(true)}>
                <Icon icon={RotateCcw} className="h-5 w-5" />
                Rétablir les couleurs par défaut
              </Button>
            </div>
          </fieldset>

          <div>
            <Button type="button" onClick={onSave} aria-disabled={busy}>
              {busy ? "Enregistrement…" : "Enregistrer"}
            </Button>
          </div>
        </div>

        <div className="min-w-0 xl:sticky xl:top-6">
          <AppearancePreview
            name={association.name}
            tagline={association.tagline}
            logoUrl={shownLogo}
            palette={palette}
          />
        </div>
      </div>

      <ConfirmDialog
        open={confirmRemove}
        onOpenChange={setConfirmRemove}
        title="Retirer le logo ?"
        confirmLabel="Retirer le logo"
        onConfirm={onRemoveLogo}
      >
        Les initiales de l'association s'afficheront à la place.
      </ConfirmDialog>
      <ConfirmDialog
        open={confirmReset}
        onOpenChange={setConfirmReset}
        title="Rétablir les couleurs par défaut ?"
        confirmLabel="Rétablir"
        onConfirm={onReset}
      >
        Les couleurs de Frigg remplaceront celles de l'association, pour tous les utilisateurs.
      </ConfirmDialog>
      <ConfirmDialog
        open={blocker.state === "blocked"}
        onOpenChange={(o) => {
          if (!o && blocker.state === "blocked") blocker.reset();
        }}
        title="Quitter sans enregistrer ?"
        confirmLabel="Quitter"
        onConfirm={() => blocker.proceed?.()}
      >
        Le logo choisi et les couleurs modifiées seront perdus.
      </ConfirmDialog>
    </SettingsPage>
  );
}
