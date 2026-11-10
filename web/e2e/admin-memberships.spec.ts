import { expectNoBlockingA11yViolations } from "./axe";
import { waitForMail } from "./mailpit";
import { expect, login, test } from "./test";

// Adhésions, administration (006 : US2, US6, US7, US8). Jeu de données :
// demandes en attente de Théo et Zoé Durand, adhésion à payer de Marc Durand,
// fiche sans compte de Jeanne Petit.

test.describe("adhésions — administration (006)", () => {
  test("validation et refus des demandes", async ({ page }) => {
    await login(page, "admin-adhesions@exemple.fr");
    const main = page.locator("#contenu");
    const since = new Date();

    await test.step("US2 — demandes en attente", async () => {
      await page.goto("/admin");
      await expect(main.getByText(/demandes? d'adhésion en attente/)).toBeVisible();
      await main.getByRole("link", { name: "Traiter les demandes" }).click();
      await expect(page).toHaveURL(/status=pending/);
      await expect(main.getByRole("link", { name: "Durand Théo" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("US2 — validation avec changement de type", async () => {
      await main.getByRole("link", { name: "Durand Théo" }).click();
      await expect(main.getByText("Inconnu")).toBeVisible();
      await expect(
        main.getByText("La saison précédente n'est pas gérée dans Frigg."),
      ).toBeVisible();
      await main.getByRole("button", { name: "Valider" }).click();
      const dialog = page.getByRole("dialog");
      const amount = dialog.getByLabel("Montant à payer (€)");
      await expect(amount).toHaveValue("40");
      await expect(dialog.getByText(/Montant théorique : 40,00/)).toBeVisible();
      await dialog.getByRole("radio", { name: "Première adhésion" }).click();
      await expect(amount).toHaveValue("60");
      await expect(dialog.getByText(/Montant théorique : 60,00/)).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await dialog.getByRole("button", { name: "Valider l'adhésion" }).click();
      await expect(main.getByText("Adhésion validée.").first()).toBeVisible();
      await expect(main.getByText(/^ADH-\d{4}-\d{4,}$/)).toBeVisible();
      const mail = await waitForMail(
        "membre-adhesions-decision@exemple.fr",
        since,
        "Théo Durand validée",
      );
      expect(mail).toContain("/adhesions/");
    });

    await test.step("US2 — refus motivé", async () => {
      await page.goto("/admin/adhesions?season=all&status=pending");
      await main.getByRole("link", { name: "Durand Zoé" }).click();
      await main.getByRole("button", { name: "Refuser" }).click();
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("button", { name: "Refuser la demande" }).click();
      await expect(dialog.getByText("Indiquez le motif.")).toBeVisible();
      await dialog.getByLabel(/Motif/).fill("Fiche en double");
      await expectNoBlockingA11yViolations(page);
      await dialog.getByRole("button", { name: "Refuser la demande" }).click();
      await expect(main.getByText("Demande refusée.").first()).toBeVisible();
      const mail = await waitForMail(
        "membre-adhesions-decision@exemple.fr",
        since,
        "Zoé Durand refusée",
      );
      expect(mail).toContain("Fiche en double");
    });
  });

  test("création directe avec règlement reçu", async ({ page }) => {
    await login(page, "admin-adhesions-creation@exemple.fr");
    const main = page.locator("#contenu");
    await page.goto("/admin/membres?q=Petit");
    await main.getByRole("link", { name: "Petit Jeanne" }).click();
    await main.getByRole("link", { name: "Créer une adhésion" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Créer une adhésion" })).toBeVisible();
    await expect(main.getByLabel(/^Membre/)).toHaveValue("Petit Jeanne");
    await main.getByRole("radio", { name: /Jeune/ }).click();
    await main.getByRole("radio", { name: "Première adhésion" }).click();
    await expect(main.getByText(/Montant théorique : 30,00/)).toBeVisible();
    await expect(main.getByLabel("Montant à payer (€)")).toHaveValue("30");
    await main.getByLabel("Règlement déjà reçu").check();
    // Don (014) : 35 € reçus pour 30 € à payer.
    await expect(main.getByLabel("Montant reçu (€)")).toHaveValue("30");
    await main.getByLabel("Montant reçu (€)").fill("35");
    await main.getByRole("radio", { name: "Chèque" }).click();
    await main.getByLabel(/Commentaire/).fill("Chèque n° 4521");
    await expectNoBlockingA11yViolations(page);
    await main.getByRole("button", { name: "Enregistrer l'adhésion" }).click();
    await expect(main.getByText("Adhésion enregistrée.").first()).toBeVisible();
    await expect(main.getByText(/^Payée le /).first()).toBeVisible();
    await expect(main.getByText(/dont 5,00\s€ de don/)).toBeVisible();

    await main.getByRole("link", { name: "Jeanne Petit" }).click();
    const section = main.getByRole("region", { name: "Adhésions" });
    await expect(section.getByRole("link", { name: /Saison/ })).toBeVisible();
    await main.getByRole("button", { name: "Supprimer" }).click();
    await expect(page.getByRole("alertdialog")).toContainText("Son adhésion sera aussi supprimée");
    await page.getByRole("alertdialog").getByRole("button", { name: "Annuler" }).click();
  });

  test("correction et annulation d'une adhésion validée", async ({ page }) => {
    await login(page, "admin-adhesions-correction@exemple.fr");
    const main = page.locator("#contenu");
    await page.goto("/admin/adhesions?q=Marc");
    await main.getByRole("link", { name: "Durand Marc" }).click();

    await main.getByRole("button", { name: "Corriger" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("radio", { name: /Jeune/ }).click();
    // Remise (014) : 25 € au lieu de 30 €, motif obligatoire.
    await dialog.getByLabel("Montant à payer (€)").fill("25");
    await dialog.getByRole("button", { name: "Corriger l'adhésion" }).click();
    await expect(dialog.getByText("Indiquez le motif de l'ajustement.")).toBeVisible();
    await dialog.getByLabel("Motif de l'ajustement").fill("Réduction accordée par le bureau");
    await expectNoBlockingA11yViolations(page);
    await dialog.getByRole("button", { name: "Corriger l'adhésion" }).click();
    await expect(main.getByText("Adhésion corrigée.").first()).toBeVisible();
    await expect(main.getByText("25,00 €").first()).toBeVisible();
    await expect(main.getByText(/Théorique 30,00\s€ · Remise 5,00\s€/)).toBeVisible();
    await expect(main.getByText("Motif : Réduction accordée par le bureau")).toBeVisible();

    await main.getByRole("button", { name: "Annuler l'adhésion" }).click();
    await page.getByRole("dialog").getByLabel(/Motif/).fill("Départ de l'association");
    await expectNoBlockingA11yViolations(page);
    await page.getByRole("dialog").getByRole("button", { name: "Annuler l'adhésion" }).click();
    await expect(main.getByText("Adhésion annulée.").first()).toBeVisible();
    await expect(main.getByText("Départ de l'association")).toBeVisible();
  });

  test("chiffres, filtres et recherche", async ({ page }) => {
    await login(page, "admin-adhesions-chiffres@exemple.fr");
    const main = page.locator("#contenu");
    await page.getByRole("link", { name: "Adhésions" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Adhésions" })).toBeVisible();
    await expect(main.getByRole("heading", { name: "Chiffres de la saison" })).toBeVisible();
    await expect(main.getByText("Adhésions validées")).toBeVisible();
    await expect(main.getByText("Restant à percevoir")).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    // Les autres tests modifient les mêmes adhésions en parallèle : on vérifie
    // le filtre par son effet sur chaque ligne, quelles qu'elles soient.
    await page.goto("/admin/adhesions?payment=paid");
    await expect(main.getByRole("status").filter({ hasText: /adhésion|Aucune/ })).toBeVisible();
    for (const row of await main.getByRole("table", { name: "Adhésions" }).getByRole("row").all()) {
      if ((await row.getByRole("columnheader").count()) > 0) continue;
      await expect(row.getByText(/^Payée/)).toBeVisible();
    }

    await page.goto("/admin/adhesions");
    await main.getByLabel("Nom ou référence").fill("0001");
    await expect(page).toHaveURL(/q=0001/);
    await expect(
      main.getByRole("table", { name: "Adhésions" }).getByText(/ADH-\d{4}-0001/),
    ).toBeVisible();
    await expect(main.getByRole("link", { name: "Bernard Hugo" })).toBeVisible();
  });
});
