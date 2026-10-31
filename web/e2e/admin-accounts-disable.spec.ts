import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, PASSWORD, test } from "./test";

test("désactiver puis réactiver un compte (002, US2)", async ({ page, browser }) => {
  const email = "membre-a-desactiver@exemple.fr";
  const member = await (await browser.newContext({ locale: "fr-FR" })).newPage();
  await login(member, email);

  await login(page, "admin-desactivation@exemple.fr");
  await page.goto(`/admin/comptes?q=${email}`);
  await page.getByRole("link", { name: email }).click();
  const trigger = page.getByRole("button", { name: "Désactiver", exact: true });
  await trigger.click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toContainText("sera déconnecté de tous ses appareils");
  await expectNoBlockingA11yViolations(page);
  await dialog.getByRole("button", { name: "Désactiver" }).click();
  await expect(page.getByText(`Le compte ${email} a été désactivé.`).first()).toBeVisible();

  // La session du membre est fermée, la reconnexion échoue.
  await member.reload();
  await expect(member).toHaveURL(/\/connexion/);
  await member.getByLabel("Adresse électronique").fill(email);
  await member.getByLabel(/^Mot de passe/).fill(PASSWORD);
  await member.getByRole("button", { name: "Se connecter" }).click();
  await expect(member.getByText("Adresse électronique ou mot de passe incorrect")).toBeVisible();

  await page.getByRole("button", { name: "Réactiver" }).click();
  await expect(page.getByText(`Le compte ${email} a été réactivé.`).first()).toBeVisible();
  await login(member, email);
});
