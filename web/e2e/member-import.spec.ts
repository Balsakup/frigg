import { readFile } from "node:fs/promises";
import path from "node:path";
import type { Page } from "@playwright/test";
import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

const FIXTURE = path.join(import.meta.dirname, "fixtures", "member-import.csv");
const ADMIN = "admin-membres@exemple.fr";

// Jeu du quickstart 013 : « Marie Dupont » existe déjà, et deux champs
// personnalisés de membre actifs (créés par l'API).
async function prepare(page: Page) {
  // Avant les champs : « Numéro de licence » est obligatoire à la saisie.
  const marie = await page.request.post("/api/admin/members", {
    data: { firstName: "Marie", lastName: "Dupont", birthDate: "2012-03-12", accountId: null },
  });
  expect(marie.status()).toBe(201);
  const fields = await (await page.request.get("/api/admin/custom-fields")).json();
  const has = (label: string) => fields.member.some((f: { label: string }) => f.label === label);
  if (!has("Numéro de licence")) {
    const res = await page.request.post("/api/admin/custom-fields", {
      data: {
        scope: "member",
        label: "Numéro de licence",
        type: "short_text",
        visibility: "admin",
        required: true,
      },
    });
    expect(res.status()).toBe(201);
  }
  if (!has("Catégorie")) {
    const res = await page.request.post("/api/admin/custom-fields", {
      data: {
        scope: "member",
        label: "Catégorie",
        type: "choice",
        visibility: "admin",
        required: false,
        choices: ["Poussin", "Benjamin", "Minime"],
      },
    });
    expect(res.status()).toBe(201);
  }
}

test.describe("importation des membres (013)", () => {
  test.describe.configure({ mode: "serial" });

  test("importer un fichier, comprendre les rejets (US1 à US3)", async ({ page }) => {
    await login(page, ADMIN);
    await prepare(page);
    const main = page.locator("#contenu");

    await test.step("étape 1 : fichier", async () => {
      await page.goto("/admin/membres");
      await main.getByRole("link", { name: "Importer" }).click();
      await expect(
        page.getByRole("heading", { level: 1, name: "Importer des membres" }),
      ).toBeVisible();
      await expect(page.getByRole("list", { name: "Étapes de l'import" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await page.getByLabel("Fichier CSV").setInputFiles(FIXTURE);
    });

    await test.step("étape 2 : colonnes, au clavier", async () => {
      await expect(page.getByRole("heading", { level: 2, name: "2. Colonnes" })).toBeFocused();
      await expect(main.getByText("7 lignes")).toBeVisible();
      await expect(main.getByText("Windows-1252")).toBeVisible();
      await expect(main.getByText("Léa · Marie · Paul")).toBeVisible();
      await expect(
        page.getByRole("combobox", { name: "Champ Frigg pour la colonne NOM" }),
      ).toHaveText(/Nom \*/);
      await expect(
        page.getByRole("combobox", { name: "Champ Frigg pour la colonne Ville" }),
      ).toHaveText(/Commune/);
      await expect(
        page.getByRole("combobox", { name: "Champ Frigg pour la colonne Catégorie" }),
      ).toHaveText(/Catégorie/);
      const licence = page.getByRole("combobox", { name: "Champ Frigg pour la colonne Licence" });
      await expect(licence).toHaveText(/Ne pas importer/);

      // Association au clavier : focus, ouverture, choix.
      await licence.focus();
      await page.keyboard.press("Enter");
      const option = page.getByRole("option", { name: /Numéro de licence/ });
      await expect(option).toBeVisible();
      await option.focus();
      await page.keyboard.press("Enter");
      await expect(licence).toHaveText(/Numéro de licence/);

      // La cible prise est désactivée pour les autres colonnes.
      await page.getByRole("combobox", { name: "Champ Frigg pour la colonne Remarques" }).click();
      await expect(page.getByRole("option", { name: /Numéro de licence/ })).toHaveAttribute(
        "aria-disabled",
        "true",
      );
      await page.keyboard.press("Escape");

      // Champ exigé non associé : alerte et bouton bloqué.
      const birth = page.getByRole("combobox", {
        name: "Champ Frigg pour la colonne Date de naissance",
      });
      await birth.click();
      await page.getByRole("option", { name: "Ne pas importer" }).click();
      await expect(
        main.getByText("Associez une colonne au champ « Date de naissance »."),
      ).toBeVisible();
      await expect(main.getByRole("button", { name: "Voir l'aperçu" })).toHaveAttribute(
        "aria-disabled",
        "true",
      );
      await birth.click();
      await page.getByRole("option", { name: /^Date de naissance/ }).click();
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("button", { name: "Voir l'aperçu" }).click();
    });

    await test.step("étape 3 : aperçu et lignes rejetées (US2)", async () => {
      await expect(page.getByRole("heading", { level: 2, name: "3. Aperçu" })).toBeFocused();
      await expect(main.getByText("2 à importer")).toBeVisible();
      await expect(main.getByText("3 en erreur")).toBeVisible();
      await expect(main.getByText("2 doublons")).toBeVisible();
      await expect(
        main.getByText(/1 fiche importée n'aura pas de valeur.*Numéro de licence \(1\)/),
      ).toBeVisible();

      // Filtre « En erreur » par défaut : les trois lignes et leurs raisons.
      await expect(page.getByRole("radio", { name: /En erreur/ })).toBeChecked();
      const rows = main.getByRole("table").getByRole("row");
      await expect(rows).toHaveCount(4);
      await expect(rows.nth(1)).toContainText("Indiquez l'année sur 4 chiffres");
      await expect(rows.nth(2)).toContainText("Choix non reconnu");
      await expect(rows.nth(3)).toContainText("Code postal");
      await expectNoBlockingA11yViolations(page);

      await page.getByRole("radio", { name: /Doublons/ }).click();
      await expect(main.getByRole("link", { name: "Dupont Marie (12/03/2012)" })).toHaveAttribute(
        "href",
        /\/admin\/membres\/\d+$/,
      );
      await expect(main.getByText("Même personne qu'à la ligne 5")).toBeVisible();

      const download = page.waitForEvent("download");
      await main.getByRole("button", { name: "Télécharger les lignes rejetées" }).click();
      const file = await download;
      expect(file.suggestedFilename()).toBe("lignes-rejetees.csv");
      const text = await readFile(await file.path(), "utf8");
      expect(text.startsWith("﻿NOM;Prénom;Date de naissance")).toBe(true);
      expect(text).toContain("Raison du rejet");
      expect(text).toContain("'+32 470 12 34 56");

      await main.getByRole("button", { name: "Importer 2 membres" }).click();
    });

    await test.step("étape 4 : rapport (US3)", async () => {
      await expect(page.getByRole("heading", { level: 2, name: "4. Rapport" })).toBeFocused();
      await expect(main.getByText("2 membres importés.").first()).toBeVisible();
      await expect(main.getByText("1 fiche à compléter")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("link", { name: "Voir les membres sans compte" }).click();
      await expect(page).toHaveURL(/rattachement=sans-compte/);
      await expect(main.getByRole("link", { name: "Lefèvre Léa" })).toBeVisible();
      await main.getByRole("link", { name: "Lefèvre Léa" }).click();
      await expect(main.getByRole("combobox", { name: "Catégorie" })).toHaveText(/Poussin/);
      await expect(main.getByRole("textbox", { name: /Numéro de licence/ })).toHaveValue("L-001");
    });
  });

  test("partir du modèle de fichier (US4)", async ({ page }) => {
    await login(page, ADMIN);
    await page.goto("/admin/membres/importer");
    const download = page.waitForEvent("download");
    await page.getByRole("button", { name: "Télécharger le modèle" }).click();
    const file = await download;
    expect(file.suggestedFilename()).toBe("modele-import-membres.csv");
    const text = await readFile(await file.path(), "utf8");
    expect(text).toContain("Prénom *");
    expect(text).toContain("Date de naissance * (JJ/MM/AAAA)");
  });

  test("champ archivé pendant l'aperçu", async ({ page }) => {
    await login(page, ADMIN);
    const created = await page.request.post("/api/admin/custom-fields", {
      data: {
        scope: "member",
        label: "Taille",
        type: "short_text",
        visibility: "admin",
        required: false,
      },
    });
    expect(created.status()).toBe(201);
    const field = await created.json();
    const main = page.locator("#contenu");
    await page.goto("/admin/membres/importer");
    await page.getByLabel("Fichier CSV").setInputFiles({
      name: "taille.csv",
      mimeType: "text/csv",
      buffer: Buffer.from("Prénom;Nom;Date de naissance;Taille\r\nNina;Archivée;01/02/2015;M\r\n"),
    });
    await expect(
      page.getByRole("combobox", { name: "Champ Frigg pour la colonne Taille" }),
    ).toHaveText(/Taille/);
    await main.getByRole("button", { name: "Voir l'aperçu" }).click();
    await expect(main.getByText("1 à importer")).toBeVisible();

    const archived = await page.request.post(`/api/admin/custom-fields/${field.id}/archive`);
    expect(archived.status()).toBe(200);
    await main.getByRole("button", { name: "Importer 1 membre" }).click();
    await expect(page.getByRole("heading", { level: 2, name: "2. Colonnes" })).toBeFocused();
    await expect(main.getByText("Ce champ n'existe plus ou n'est pas importable.")).toBeVisible();
  });

  test("lisible à 320 px", async ({ page }) => {
    await login(page, ADMIN);
    await page.setViewportSize({ width: 320, height: 800 });
    await page.goto("/admin/membres/importer");
    await page.getByLabel("Fichier CSV").setInputFiles(FIXTURE);
    await expect(page.getByRole("heading", { level: 2, name: "2. Colonnes" })).toBeVisible();
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await expectNoBlockingA11yViolations(page);
  });

  test("refusé à un membre", async ({ page }) => {
    await login(page, "membre@exemple.fr");
    const res = await page.request.post("/api/admin/members/import/analyze", {
      multipart: { file: { name: "a.csv", mimeType: "text/csv", buffer: Buffer.from("Nom\nA\n") } },
    });
    expect(res.status()).toBe(403);
  });
});
