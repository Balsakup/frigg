import { expectNoBlockingA11yViolations } from "./axe";
import { waitForLink } from "./mailpit";
import { expect, login, test } from "./test";

test("confirmation de l'adresse électronique (US6)", async ({ page }) => {
  const email = "membre-non-confirme@exemple.fr";
  await login(page, email);
  const banner = page.getByText("Confirmez votre adresse électronique");
  await expect(banner).toBeVisible();
  await expectNoBlockingA11yViolations(page);

  // L'alerte reste en tête de chaque page.
  await page.getByRole("link", { name: "Mes membres" }).click();
  await expect(banner).toBeVisible();

  const since = new Date();
  await page.getByRole("button", { name: "Renvoyer le lien" }).click();
  await expect(page.getByText("Un nouveau lien vient d'être envoyé.")).toBeVisible();

  await page.goto(await waitForLink(email, since));
  await expect(page.getByText("Votre adresse électronique est confirmée.")).toBeVisible();
  await expectNoBlockingA11yViolations(page);

  await page.getByRole("link", { name: "Aller à l'accueil" }).click();
  await expect(page.getByRole("heading", { level: 1, name: "Accueil" })).toBeVisible();
  await expect(banner).toHaveCount(0);
});
