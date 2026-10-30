import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, PASSWORD, test } from "./test";

test("changer son mot de passe depuis « Mon compte » (US7)", async ({ page, browser }) => {
  const email = "membre-compte@exemple.fr";
  const other = await (await browser.newContext({ locale: "fr-FR" })).newPage();
  await login(other, email);
  await login(page, email);

  await page.getByRole("button", { name: /menu du compte/ }).click();
  await page.getByRole("menuitem", { name: "Mon compte" }).click();
  await expect(page.getByRole("heading", { level: 1, name: "Mon compte" })).toBeVisible();
  await expect(page.getByText(email).first()).toBeVisible();
  await expectNoBlockingA11yViolations(page);

  await page.getByLabel(/^Mot de passe actuel/).fill("Faux2026");
  await page.getByLabel(/^Nouveau mot de passe/).fill("Change2026");
  await page.getByLabel(/^Confirmez le nouveau mot de passe/).fill("Change2026");
  await page.getByRole("button", { name: "Enregistrer le mot de passe" }).click();
  await expect(page.getByText("Le mot de passe actuel est incorrect.")).toBeVisible();

  await page.getByLabel(/^Mot de passe actuel/).fill(PASSWORD);
  await page.getByRole("button", { name: "Enregistrer le mot de passe" }).click();
  await expect(page.getByText("Votre mot de passe a été modifié.").first()).toBeVisible();

  await other.reload();
  await expect(other).toHaveURL(/\/connexion/);
  await login(other, email, "Change2026");
});
