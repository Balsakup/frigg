import { expectNoBlockingA11yViolations } from "./axe";
import { waitForLink } from "./mailpit";
import { expect, login, test } from "./test";

test("mot de passe oublié : nouveau mot de passe et fermeture des sessions (US5)", async ({
  page,
  browser,
}) => {
  const email = "membre-reinitialisation@exemple.fr";
  const other = await (await browser.newContext({ locale: "fr-FR" })).newPage();
  await login(other, email);

  await page.goto("/connexion");
  await page.getByRole("link", { name: "Mot de passe oublié ?" }).click();
  await expectNoBlockingA11yViolations(page);
  const since = new Date();
  await page.getByLabel(/^Adresse électronique/).fill(email);
  await page.getByRole("button", { name: "Recevoir le lien" }).click();
  await expect(page.getByText("Si un compte existe pour cette adresse")).toBeVisible();

  await page.goto(await waitForLink(email, since));
  await expect(page.getByRole("heading", { name: "Nouveau mot de passe" })).toBeVisible();
  await expectNoBlockingA11yViolations(page);
  await page.getByLabel(/^Mot de passe/).fill("Reinitialise2026");
  await page.getByLabel(/^Confirmez le mot de passe/).fill("Reinitialise2026");
  await page.getByRole("button", { name: "Enregistrer le mot de passe" }).click();
  await expect(page.getByText("Votre mot de passe a été modifié.")).toBeVisible();

  // L'autre session est fermée.
  await other.reload();
  await expect(other).toHaveURL(/\/connexion/);

  await page.getByRole("link", { name: "Se connecter" }).click();
  await login(page, email, "Reinitialise2026");
});
