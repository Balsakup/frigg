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

// Fiches gérées par le compte supprimé (005, US5).
async function addMember(page: import("@playwright/test").Page, first: string, last: string) {
  await page.goto("/mes-membres/nouveau");
  await page.getByLabel(/^Prénom/).fill(first);
  await page.getByLabel(/^Nom/).fill(last);
  await page.getByLabel(/^Date de naissance/).fill("2012-01-01");
  await page.getByRole("button", { name: "Ajouter le membre" }).click();
  await expect(page.getByText("Membre ajouté.").first()).toBeVisible();
}

test("supprimer un compte qui gère des fiches (005, US5)", async ({ page, context }) => {
  await login(page, "membre-fiches-conservees@exemple.fr");
  await addMember(page, "Inès", "Conservée");
  await addMember(page, "Hugo", "Conservé");
  await context.clearCookies();
  await login(page, "membre-fiches-effacees@exemple.fr");
  await addMember(page, "Nina", "Effacée");
  await context.clearCookies();

  await login(page, "admin-suppression-fiches@exemple.fr");
  const dialog = page.getByRole("alertdialog");

  // Conserver les fiches sans compte.
  await page.goto("/admin/comptes?q=membre-fiches-conservees");
  await page.getByRole("link", { name: "membre-fiches-conservees@exemple.fr" }).click();
  await expect(
    page.getByRole("heading", { level: 1, name: "membre-fiches-conservees@exemple.fr" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Supprimer", exact: true }).click();
  await expect(dialog).toContainText("Ce compte gère 2 fiches de membre");
  await expect(dialog).toContainText("Inès Conservée");
  await expect(dialog).toContainText("Hugo Conservé");
  await expectNoBlockingA11yViolations(page);
  await dialog.getByRole("button", { name: "Supprimer" }).click();
  await expect(
    dialog.getByText("Choisissez le sort des fiches gérées par ce compte."),
  ).toBeVisible();
  await dialog.getByRole("radio", { name: "Conserver ces fiches sans compte" }).click();
  await dialog.getByRole("button", { name: "Supprimer" }).click();
  await expect(page).toHaveURL(/\/admin\/comptes$/);

  await page.goto("/admin/membres?rattachement=sans-compte&q=conserv");
  const rows = page.locator("#contenu").getByRole("row");
  await expect(rows).toHaveCount(3);
  await expect(rows.nth(1)).toContainText("Sans compte");

  // Supprimer aussi les fiches.
  await page.goto("/admin/comptes?q=membre-fiches-effacees");
  await page.getByRole("link", { name: "membre-fiches-effacees@exemple.fr" }).click();
  await expect(
    page.getByRole("heading", { level: 1, name: "membre-fiches-effacees@exemple.fr" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Supprimer", exact: true }).click();
  await expect(dialog).toContainText("Nina Effacée");
  await dialog.getByRole("radio", { name: "Supprimer aussi ces fiches" }).click();
  await dialog.getByRole("button", { name: "Supprimer" }).click();
  await expect(page).toHaveURL(/\/admin\/comptes$/);
  await page.goto("/admin/membres?q=effac");
  await expect(page.locator("#contenu").getByText("Aucun membre ne correspond")).toBeVisible();
});
