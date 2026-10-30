import { test as base, expect, type Page } from "@playwright/test";

export { expect };

export const PASSWORD = "Motdepasse1";

// Une IP fictive par test : le serveur e2e fait confiance au proxy local et lit
// X-Forwarded-For, ce qui isole les tests vis-à-vis de la limitation de débit.
export const test = base.extend({
  // biome-ignore lint/correctness/noEmptyPattern: signature imposée par les fixtures Playwright
  extraHTTPHeaders: async ({}, use) => {
    const ip = `198.51.100.${Math.floor(Math.random() * 250) + 1}`;
    await use({
      "X-Forwarded-For": `10.${Math.floor(Math.random() * 250)}.0.${Math.floor(Math.random() * 250)}, ${ip}`,
    });
  },
});

// Connexion par l'interface ; attend l'arrivée dans le shell.
export async function login(page: Page, email: string, password = PASSWORD) {
  await page.goto("/connexion");
  await page.getByLabel("Adresse électronique").fill(email);
  await page.getByLabel(/^Mot de passe/).fill(password);
  await page.getByRole("button", { name: "Se connecter" }).click();
  await expect(page).not.toHaveURL(/\/connexion/);
  await expect(
    page
      .getByRole("button", { name: /menu du compte/ })
      .or(page.getByRole("button", { name: "Menu de navigation" }))
      .first(),
  ).toBeVisible();
}
