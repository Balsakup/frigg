import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

test("supprimer définitivement un compte (002, US4)", async ({ page }) => {
  const email = "admin-a-supprimer@exemple.fr";
  await login(page, "admin-suppression@exemple.fr");
  await page.goto(`/admin/comptes?q=${email}`);
  await page.getByRole("link", { name: email }).click();
  await page.getByRole("button", { name: "Supprimer" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toContainText("Cette action est irréversible");
  await expect(dialog.getByRole("button", { name: "Désactiver plutôt" })).toBeVisible();
  await expectNoBlockingA11yViolations(page);
  await dialog.getByRole("button", { name: "Supprimer" }).click();

  await expect(page).toHaveURL(/\/admin\/comptes$/);
  await expect(page.getByText(`Le compte ${email} a été supprimé.`).first()).toBeVisible();
  const invitation = page.getByRole("row").filter({ hasText: "invite-par-supprime@exemple.fr" });
  await expect(invitation).toContainText("Compte supprimé");

  // L'adresse peut être invitée à nouveau.
  await page.getByLabel(/^Adresse électronique/).fill(email);
  await page.getByRole("button", { name: "Envoyer l'invitation", exact: true }).click();
  await expect(page.getByText(`Invitation envoyée à ${email}.`).first()).toBeVisible();
});
