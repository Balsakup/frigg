import { expectNoBlockingA11yViolations } from "./axe";
import { waitForMail } from "./mailpit";
import { expect, login, test } from "./test";

// Paiement par carte via HelloAsso (008, US1, US3). Serveur « carte » : jeu des
// adhésions, carte activée en sandbox, faux HelloAsso local. Chaque compte a
// une adhésion « Jeune » (30 €) validée à payer.

async function openPayment(page: import("@playwright/test").Page, member: string) {
  await page.getByRole("link", { name: "Adhésions" }).click();
  await page.locator("#contenu").getByRole("link", { name: member }).click();
  await expect(page.getByRole("heading", { level: 2, name: "Paiement" })).toBeVisible();
}

test.describe("paiement par carte (008)", () => {
  test("payer par carte et revenir sur Frigg", async ({ page }) => {
    const email = "membre-carte-paiement@exemple.fr";
    await login(page, email);
    const main = page.locator("#contenu");
    await openPayment(page, "Nina Leroy");

    await test.step("la carte est proposée en premier, en mode test", async () => {
      const radios = main.getByRole("radio");
      await expect(radios.first()).toHaveAccessibleName("Carte bancaire");
      await expect(main.getByText("Mode test — aucun débit réel.")).toBeVisible();
      await main.getByRole("radio", { name: "Carte bancaire" }).click();
      await expect(main.getByText(/redirigé vers HelloAsso/)).toBeVisible();
      await expect(main.getByText(/contribution volontaire facultative/)).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    const since = new Date();
    await test.step("paiement chez HelloAsso puis retour", async () => {
      await main.getByRole("button", { name: "Payer 30,00 € par carte" }).click();
      await expect(page.getByRole("heading", { name: "HelloAsso (test)" })).toBeVisible();
      await expect(page.getByText(/Jeune/)).toBeVisible();
      await expect(page.getByText(/Nina/)).toHaveCount(0);
      await page.getByRole("button", { name: "Payer", exact: true }).click();
      await expect(page).toHaveURL(/\/adhesions\/\d+$/);
      await expect(main.getByText(/^Paiement reçu, merci\. L'adhésion de Nina/)).toBeVisible();
      await expect(main.getByText(/carte bancaire/i).first()).toBeVisible();
      await expect(main.getByText("Mode test", { exact: true }).first()).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("e-mail de constatation", async () => {
      const text = await waitForMail(email, since, "Paiement de l'adhésion");
      expect(text).toContain("carte bancaire");
    });
  });

  test("paiement refusé puis retour arrière", async ({ page }) => {
    await login(page, "membre-carte-refus@exemple.fr");
    const main = page.locator("#contenu");
    await openPayment(page, "Paul Morel");
    await main.getByRole("radio", { name: "Carte bancaire" }).click();
    await main.getByRole("button", { name: "Payer 30,00 € par carte" }).click();
    await page.getByRole("button", { name: "Refuser" }).click();
    await expect(
      main.getByText(/Aucune somme n'a été débitée\. Vous pouvez réessayer/),
    ).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: "Paiement" })).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    await main.getByRole("radio", { name: "Carte bancaire" }).click();
    await main.getByRole("button", { name: "Payer 30,00 € par carte" }).click();
    await page.getByRole("button", { name: "Retour" }).click();
    await expect(page.getByRole("heading", { level: 2, name: "Paiement" })).toBeVisible();
    await expect(page).toHaveURL(/\/adhesions\/\d+$/);
  });

  test("paiement constaté par la notification, sans retour", async ({ page }) => {
    await login(page, "membre-carte-notification@exemple.fr");
    const main = page.locator("#contenu");
    await openPayment(page, "Lina Fabre");
    const detail = page.url();
    await main.getByRole("radio", { name: "Carte bancaire" }).click();
    await main.getByRole("button", { name: "Payer 30,00 € par carte" }).click();
    await page.getByRole("button", { name: "Payer sans revenir" }).click();
    await expect(page.getByRole("heading", { name: "Paiement accepté" })).toBeVisible();
    await page.goto(detail);
    await expect(main.getByText(/^Payée le .* \(carte bancaire\)/)).toBeVisible();
    await expect(page.getByRole("heading", { level: 2, name: "Paiement" })).toHaveCount(0);
  });

  test("parcours au clavier", async ({ page }) => {
    await login(page, "membre-carte-clavier@exemple.fr");
    const main = page.locator("#contenu");
    await openPayment(page, "Sacha Blanc");
    const card = main.getByRole("radio", { name: "Carte bancaire" });
    await card.focus();
    await page.keyboard.press("Space");
    await expect(card).toBeChecked();
    const pay = main.getByRole("button", { name: "Payer 30,00 € par carte" });
    await pay.focus();
    await expect(pay).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("heading", { name: "HelloAsso (test)" })).toBeVisible();
  });

  test("administration : paiement par carte, signalement et traitement", async ({ page }) => {
    await login(page, "admin-carte@exemple.fr");
    const main = page.locator("#contenu");
    await page.goto("/admin/adhesions?season=all&paidMethod=card");
    await expect(main.getByRole("link", { name: "Garnier Rémi" })).toBeVisible();
    await main.getByRole("link", { name: "Garnier Rémi" }).click();
    await expect(page.getByRole("heading", { level: 2, name: "Paiements en ligne" })).toBeVisible();
    await expect(main.getByText("n° 77777").first()).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    await main.getByRole("button", { name: "Annuler l'adhésion" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toContainText("espace HelloAsso");
    await dialog.getByLabel("Motif").fill("Départ de l'association");
    await dialog.getByRole("button", { name: "Annuler l'adhésion" }).click();
    await expect(main.getByText(/remboursement à effectuer dans HelloAsso/)).toBeVisible();
    await main.getByRole("button", { name: "Marquer comme traité" }).click();
    await page
      .getByRole("alertdialog")
      .getByRole("button", { name: "Marquer comme traité" })
      .click();
    await expect(main.getByText("Signalement traité.").first()).toBeVisible();
    await expect(main.getByText(/remboursement à effectuer dans HelloAsso/)).toHaveCount(0);
  });
});
