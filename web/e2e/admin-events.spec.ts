import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Événements, administration des inscrits (012 : US5, US6). Serveur des
// événements : Paula Martin est inscrite au « Stage de printemps » et a payé par
// chèque ; Jeanne Petit est une fiche sans compte.

function localInput(daysFromNow: number, hour: number) {
  const d = new Date();
  d.setDate(d.getDate() + daysFromNow);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(hour)}:00`;
}

test.describe("événements — inscrits (012 US5)", () => {
  test("inscrire, exporter, retirer, rechercher", async ({ page }) => {
    await login(page, "admin-evenements-inscrits@exemple.fr");
    const main = page.locator("#contenu");
    await page.goto("/admin/evenements");
    await main.getByRole("link", { name: "Stage de printemps" }).click();

    await test.step("inscrits et chiffres", async () => {
      const table = main.getByRole("table", { name: /Inscrits à Stage de printemps/ });
      await expect(table.getByRole("row", { name: /Martin Paula/ })).toContainText("Payée");
      await expect(main.getByText("Encaissé")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("inscription manuelle déjà payée", async () => {
      await main.getByRole("button", { name: "Inscrire un membre" }).click();
      const dialog = page.getByRole("dialog", { name: "Inscrire un membre" });
      await dialog.getByLabel(/^Membre/).fill("Petit");
      await page.getByRole("option", { name: /Petit Jeanne/ }).click();
      // Remise (014) : 20 € au lieu du prix, avec motif.
      await dialog.getByLabel("Montant à payer (€)").fill("20");
      await dialog.getByLabel("Motif de l'ajustement").fill("Bénévole encadrant");
      await dialog.getByRole("checkbox", { name: /Règlement déjà reçu/ }).click();
      await expect(dialog.getByLabel("Montant reçu (€)")).toHaveValue("20");
      await dialog.getByRole("radio", { name: "Espèces" }).click();
      await expectNoBlockingA11yViolations(page);
      await dialog.getByRole("button", { name: "Inscrire" }).click();
      await expect(main.getByText("Petit Jeanne est inscrit(e).").first()).toBeVisible();
      const row = main.getByRole("row", { name: /Petit Jeanne/ });
      await expect(row).toContainText("Payée");
      await expect(row).toContainText("Non joignable par e-mail");
    });

    await test.step("export CSV", async () => {
      const download = page.waitForEvent("download");
      await main.getByRole("button", { name: "Exporter (CSV)" }).click();
      const file = await download;
      expect(file.suggestedFilename()).toMatch(/^inscrits-\d+\.csv$/);
    });

    await test.step("retrait avec rappel de remboursement", async () => {
      const row = main.getByRole("row", { name: /Petit Jeanne/ });
      await row.getByRole("button", { name: "Retirer" }).click();
      const dialog = page.getByRole("alertdialog");
      await expect(dialog).toContainText("le remboursement se fait hors de Frigg");
      await dialog.getByRole("button", { name: "Retirer l'inscription" }).click();
      await expect(main.getByText("Inscription retirée.").first()).toBeVisible();
      await expect(main.getByRole("row", { name: /Petit Jeanne/ })).toHaveCount(0);
    });

    await test.step("rubrique Inscriptions", async () => {
      await page.getByRole("link", { name: "Inscriptions", exact: true }).click();
      await expect(page.getByRole("heading", { level: 1, name: "Inscriptions" })).toBeVisible();
      await main.getByLabel("Nom ou référence").fill("EVT-");
      await expect(main.getByRole("link", { name: "Martin Paula" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });
  });
});

test.describe("événements — annuler et supprimer (012 US6)", () => {
  test("annuler, rétablir, supprimer un brouillon", async ({ page }) => {
    await login(page, "admin-evenements-inscrits@exemple.fr");
    const main = page.locator("#contenu");

    await page.goto("/admin/evenements/nouveau");
    await main.getByLabel(/^Titre/).fill("Tournoi de pétanque");
    await main.getByLabel(/^Lieu/).fill("Boulodrome");
    await main.getByLabel(/^Début/).fill(localInput(25, 14));
    await main.getByLabel(/^Fin/).fill(localInput(25, 18));
    await main.getByRole("button", { name: "Enregistrer le brouillon" }).click();
    await page.getByRole("button", { name: "Publier" }).click();
    await expect(main.getByText("Événement publié : les membres le voient.").first()).toBeVisible();

    await page.getByRole("button", { name: "Plus d'actions sur l'événement" }).click();
    await page.getByRole("menuitem", { name: "Annuler l'événement" }).click();
    const dialog = page.getByRole("alertdialog");
    await dialog.getByLabel(/Message aux inscrits/).fill("Terrain inondé.");
    await expectNoBlockingA11yViolations(page);
    await dialog.getByRole("button", { name: "Annuler l'événement" }).click();
    await expect(
      main.getByText("Événement annulé : les inscrits sont prévenus.").first(),
    ).toBeVisible();
    await expect(main.getByText("Terrain inondé.")).toBeVisible();

    await page.getByRole("button", { name: "Plus d'actions sur l'événement" }).click();
    await page.getByRole("menuitem", { name: "Rétablir l'événement" }).click();
    await expect(
      main.getByText("Événement rétabli : les inscrits sont prévenus.").first(),
    ).toBeVisible();

    await page.getByRole("button", { name: "Plus d'actions sur l'événement" }).click();
    await page.getByRole("menuitem", { name: "Supprimer" }).click();
    await page
      .getByRole("alertdialog")
      .getByRole("button", { name: "Supprimer définitivement" })
      .click();
    await expect(page).toHaveURL("/admin/evenements");
    await expect(
      main.getByText("Événement « Tournoi de pétanque » supprimé.").first(),
    ).toBeVisible();
  });
});
