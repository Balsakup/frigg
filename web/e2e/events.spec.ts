import { expectNoBlockingA11yViolations } from "./axe";
import { waitForMail } from "./mailpit";
import { expect, login, test } from "./test";

// Événements (012 : US1 à US4). Serveur des événements : saison ouverte,
// « Assemblée générale » gratuite réservée aux adhérents, « Stage de printemps »
// ouvert à tous (35 €, 20 places), Claire Martin adhérente qui gère la fiche de
// son fils Lucas, non adhérent ; espèces, chèque et virement activés.

function localInput(daysFromNow: number, hour: number) {
  const d = new Date();
  d.setDate(d.getDate() + daysFromNow);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(hour)}:00`;
}

test.describe("événements — administration (012 US1)", () => {
  test("créer, publier et modifier", async ({ page }) => {
    await login(page, "admin-evenements@exemple.fr");
    const main = page.locator("#contenu");

    await test.step("liste", async () => {
      await page.getByRole("link", { name: "Événements" }).click();
      await expect(page.getByRole("heading", { level: 1, name: "Événements" })).toBeVisible();
      await expect(main.getByRole("link", { name: "Stage de printemps" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("création, fin avant début refusée", async () => {
      await main.getByRole("link", { name: "Créer un événement" }).click();
      await main.getByLabel(/^Titre/).fill("Sortie en forêt");
      await main.getByLabel(/^Lieu/).fill("Parking du lac");
      await main.getByLabel(/^Début/).fill(localInput(15, 10));
      await main.getByLabel(/^Fin/).fill(localInput(15, 9));
      await main.getByRole("button", { name: "Enregistrer le brouillon" }).click();
      await expect(main.getByText("La fin doit suivre le début.")).toBeVisible();
      await main.getByLabel(/^Fin/).fill(localInput(15, 16));
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("button", { name: "Enregistrer le brouillon" }).click();
      await expect(page.getByRole("heading", { level: 1, name: "Sortie en forêt" })).toBeVisible();
      await expect(main.getByText("Événement enregistré en brouillon.").first()).toBeVisible();
      await expect(main.getByText(/Brouillon : les membres ne voient pas/)).toBeVisible();
    });

    await test.step("publication et modification", async () => {
      await page.getByRole("button", { name: "Publier" }).click();
      await expect(
        main.getByText("Événement publié : les membres le voient.").first(),
      ).toBeVisible();
      await page.getByRole("link", { name: "Modifier" }).click();
      await main.getByLabel(/^Lieu/).fill("Parking de la forêt");
      await main.getByRole("button", { name: "Enregistrer" }).click();
      await expect(main.getByText("Parking de la forêt")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });
  });

  test("un membre n'accède pas à l'administration des événements", async ({ page }) => {
    await login(page, "membre-evenements-paiement@exemple.fr");
    await page.goto("/admin/evenements");
    await expect(page.getByRole("heading", { level: 1, name: "Accès refusé" })).toBeVisible();
  });
});

test.describe("événements — espace membre (012 US2 à US4)", () => {
  test("consulter, s'inscrire et annuler", async ({ page }) => {
    await login(page, "membre-evenements@exemple.fr");
    const main = page.locator("#contenu");

    await test.step("US2 — événements à venir", async () => {
      await page.getByRole("link", { name: "Événements à venir" }).click();
      await expect(
        page.getByRole("heading", { level: 1, name: "Événements à venir" }),
      ).toBeVisible();
      const titles = await main.getByRole("heading", { level: 2 }).allTextContents();
      expect(titles.indexOf("Assemblée générale")).toBeLessThan(
        titles.indexOf("Stage de printemps"),
      );
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("link", { name: "Assemblée générale" }).click();
      await expect(main.getByText("rapport financier")).toBeVisible();
    });

    await test.step("US3 — inscription, fiche non adhérente signalée", async () => {
      await expect(main.getByText(/Pas d'adhésion validée pour la saison/)).toBeVisible();
      await expect(main.getByRole("checkbox", { name: "Lucas Martin" })).toHaveCount(0);
      await expectNoBlockingA11yViolations(page);
      const since = new Date();
      await main.getByRole("checkbox", { name: "Claire Martin" }).click();
      await main.getByRole("button", { name: "Inscrire" }).click();
      await expect(main.getByText("Inscription enregistrée.").first()).toBeVisible();
      await expect(main.getByText("Inscrit(e)")).toBeVisible();
      const mail = await waitForMail(
        "membre-evenements@exemple.fr",
        since,
        "Inscription confirmée",
      );
      expect(mail).toContain("Claire Martin");
    });

    await test.step("US3 — mes inscriptions et annulation", async () => {
      await page.getByRole("link", { name: "Mes inscriptions" }).click();
      await expect(page.getByRole("heading", { level: 1, name: "Mes inscriptions" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("link", { name: "Assemblée générale" }).click();
      await main.getByRole("button", { name: "Annuler mon inscription" }).click();
      const dialog = page.getByRole("alertdialog");
      await dialog.getByRole("button", { name: "Annuler l'inscription" }).click();
      await expect(
        main.getByText("Inscription annulée : la place est libérée.").first(),
      ).toBeVisible();
      await expect(main.getByText("Annulée", { exact: true })).toBeVisible();
    });
  });

  test("payer une inscription et « À régler »", async ({ page }) => {
    await login(page, "membre-evenements@exemple.fr");
    const main = page.locator("#contenu");
    await page.getByRole("link", { name: "Événements à venir" }).click();
    await main.getByRole("link", { name: "Stage de printemps" }).click();
    await main.getByRole("checkbox", { name: "Claire Martin" }).click();
    await main.getByRole("checkbox", { name: "Lucas Martin" }).click();
    await main.getByRole("button", { name: /Inscrire et payer 70,00/ }).click();
    await expect(page).toHaveURL(/\/mes-inscriptions\/\d+/);
    await expect(
      main.getByText("Inscription enregistrée : il reste à payer.").first(),
    ).toBeVisible();
    await expect(main.getByText(/Référence : EVT-/)).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    await main.getByRole("radio", { name: "Chèque" }).click();
    await main.getByRole("button", { name: "Confirmer ce moyen de paiement" }).click();
    await expect(main.getByText("Moyen de paiement enregistré.").first()).toBeVisible();
    await expect(main.getByText(/Paiement annoncé — chèque/).first()).toBeVisible();

    await page.getByRole("link", { name: "À régler" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "À régler" })).toBeVisible();
    await expect(main.getByRole("listitem").filter({ hasText: "Stage de printemps" })).toHaveCount(
      2,
    );
    await expectNoBlockingA11yViolations(page);
  });

  test("320 px : aucun défilement horizontal", async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 800 });
    await login(page, "membre-evenements-paiement@exemple.fr");
    for (const path of ["/evenements", "/mes-inscriptions", "/a-regler"]) {
      await page.goto(path);
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      );
      expect(overflow, path).toBeLessThanOrEqual(0);
    }
  });
});
