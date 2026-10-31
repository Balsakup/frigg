import { expectNoBlockingA11yViolations } from "./axe";
import { waitForLink } from "./mailpit";
import { expect, login, test } from "./test";

test.describe("invitations (US4)", () => {
  test("un administrateur invite une personne qui crée son compte", async ({ page, browser }) => {
    const email = `invite-${Date.now()}@exemple.fr`;
    await login(page, "admin-invitation@exemple.fr");
    await page.getByRole("link", { name: "Comptes" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Comptes" })).toBeVisible();

    const since = new Date();
    await page.getByLabel(/^Adresse électronique/).fill(email);
    await page.getByRole("button", { name: "Envoyer l'invitation", exact: true }).click();
    await expect(page.getByText(`Invitation envoyée à ${email}.`).first()).toBeVisible();
    await expect(page.getByRole("cell", { name: email, exact: true })).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    const link = await waitForLink(email, since);
    const guest = await browser.newContext({ locale: "fr-FR" });
    const guestPage = await guest.newPage();
    await guestPage.goto(link);
    await expect(guestPage.getByRole("heading", { name: "Créer votre compte" })).toBeVisible();
    await expect(guestPage).not.toHaveURL(/#/);
    await expect(guestPage.getByLabel("Adresse électronique")).toHaveValue(email);
    await expectNoBlockingA11yViolations(guestPage);

    await guestPage.getByLabel(/^Mot de passe/).fill("faible");
    await guestPage.getByLabel(/^Confirmez le mot de passe/).fill("faible");
    await guestPage.getByRole("button", { name: "Créer mon compte" }).click();
    await expect(
      guestPage.getByText("Le mot de passe doit contenir au moins 8 caractères.", { exact: false }),
    ).toBeVisible();

    await guestPage.getByLabel(/^Mot de passe/).fill("Nouveau2026");
    await guestPage.getByLabel(/^Confirmez le mot de passe/).fill("Nouveau2026");
    await guestPage.getByRole("button", { name: "Créer mon compte" }).click();
    await expect(guestPage).toHaveURL(/\/$/);
    await expect(guestPage.getByRole("heading", { level: 1, name: "Accueil" })).toBeVisible();
    await expect(guestPage.getByRole("navigation", { name: "Changer d'espace" })).toHaveCount(0);

    // Le lien ne sert qu'une fois.
    const again = await (await browser.newContext()).newPage();
    await again.goto(link);
    await expect(again.getByText("Cette invitation n'est plus valable.")).toBeVisible();
    await guest.close();
  });

  test("une invitation peut être annulée", async ({ page }) => {
    const email = `annulee-${Date.now()}@exemple.fr`;
    await login(page, "admin@exemple.fr");
    await page.goto("/admin/comptes");
    await page.getByLabel(/^Adresse électronique/).fill(email);
    await page.getByRole("button", { name: "Envoyer l'invitation", exact: true }).click();
    await expect(page.getByRole("cell", { name: email, exact: true })).toBeVisible();

    await page.getByRole("button", { name: `Annuler l'invitation de ${email}` }).click();
    const dialog = page.getByRole("alertdialog", { name: "Annuler l'invitation ?" });
    await expect(dialog).toBeVisible();
    await expectNoBlockingA11yViolations(page);
    await dialog.getByRole("button", { name: "Annuler l'invitation" }).click();
    await expect(page.getByRole("cell", { name: email, exact: true })).toHaveCount(0);
    await expect(page.getByText(`Invitation de ${email} annulée.`).first()).toBeVisible();
  });
});
