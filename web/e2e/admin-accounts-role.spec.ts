import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

async function changeRole(page: import("@playwright/test").Page, email: string, role: string) {
  await page.goto(`/admin/comptes?q=${email}`);
  await page.getByRole("link", { name: email }).click();
  await page.getByRole("button", { name: "Changer le rôle" }).click();
  const dialog = page.getByRole("alertdialog");
  await dialog.getByRole("combobox", { name: "Nouveau rôle" }).click();
  await page.getByRole("option", { name: role }).click();
  await expectNoBlockingA11yViolations(page);
  await dialog.getByRole("button", { name: "Enregistrer" }).click();
  await expect(page.getByText(`${email} est maintenant`).first()).toBeVisible();
}

test.describe("changer le rôle d'un compte (002, US3)", () => {
  test("une personne promue accède à l'administration", async ({ page, browser }) => {
    await login(page, "admin-role@exemple.fr");
    await changeRole(page, "membre-a-promouvoir@exemple.fr", "Administrateur");

    const promoted = await (await browser.newContext({ locale: "fr-FR" })).newPage();
    await login(promoted, "membre-a-promouvoir@exemple.fr");
    await expect(promoted.getByRole("navigation", { name: "Changer d'espace" })).toBeVisible();
  });

  test("un administrateur rétrogradé est ramené à l'espace membre", async ({ page, browser }) => {
    const demoted = await (await browser.newContext({ locale: "fr-FR" })).newPage();
    await login(demoted, "admin-a-retrograder@exemple.fr");
    await demoted.goto("/admin/comptes?q=membre-liste-01");
    await expect(demoted.getByRole("heading", { level: 1, name: "Comptes" })).toBeVisible();

    await login(page, "admin-role@exemple.fr");
    await changeRole(page, "admin-a-retrograder@exemple.fr", "Membre");

    await demoted.getByRole("link", { name: "membre-liste-01@exemple.fr" }).click();
    await expect(demoted).toHaveURL(/\/$/);
    await expect(demoted.getByText("Vos droits d'administration ont été retirés.")).toBeVisible();
    await expect(demoted.getByRole("navigation", { name: "Changer d'espace" })).toHaveCount(0);
  });
});
