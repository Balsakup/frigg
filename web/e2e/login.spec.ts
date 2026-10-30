import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, PASSWORD, test } from "./test";

test.describe("connexion (US1)", () => {
  test("redirige vers la connexion puis revient sur la page demandée", async ({ page }) => {
    await page.goto("/mon-compte?onglet=1");
    await expect(page).toHaveURL(/\/connexion\?suite=%2Fmon-compte%3Fonglet%3D1$/);
    await expect(page.getByRole("heading", { name: "Connexion" })).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    await page.getByLabel("Adresse électronique").fill("membre-connexion-1@exemple.fr");
    await page.getByLabel(/^Mot de passe/).fill(PASSWORD);
    await page.getByRole("button", { name: "Se connecter" }).click();
    await expect(page).toHaveURL(/\/mon-compte\?onglet=1$/);
  });

  test("affiche le même message pour un mauvais mot de passe et une adresse inconnue", async ({
    page,
  }) => {
    await page.goto("/connexion");
    const message = "Adresse électronique ou mot de passe incorrect.";
    for (const email of ["membre@exemple.fr", "inconnu@exemple.fr"]) {
      await page.getByLabel("Adresse électronique").fill(email);
      await page.getByLabel(/^Mot de passe/).fill("Mauvais1");
      await page.getByRole("button", { name: "Se connecter" }).click();
      await expect(page.getByRole("alert")).toContainText(message);
    }
  });

  test("arrive sur l'accueil dans le shell, puis se déconnecte", async ({ page }) => {
    await login(page, "membre-connexion-2@exemple.fr");
    await expect(page).toHaveURL("/");
    await expect(page.getByRole("heading", { level: 1, name: "Accueil" })).toBeVisible();
    await expect(page.getByText("Les Sentiers du Val").first()).toBeVisible();
    await expect(page).toHaveTitle("Accueil · Les Sentiers du Val");
    await expectNoBlockingA11yViolations(page);

    await page.getByRole("link", { name: "Mes membres" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Mes membres" })).toBeVisible();
    await page.getByRole("button", { name: /menu du compte/ }).click();
    await page.getByRole("menuitem", { name: "Se déconnecter" }).click();
    await expect(page).toHaveURL("/connexion");

    // « Précédent » ramène vers une page protégée : retour à la connexion.
    await page.goBack();
    await expect(page).toHaveURL(/\/connexion\?suite=%2F$/);
  });

  test("affiche « Page introuvable » dans le shell", async ({ page }) => {
    await login(page, "membre-connexion-3@exemple.fr");
    await page.goto("/cette-page-n-existe-pas");
    await expect(page.getByRole("heading", { level: 1, name: "Page introuvable" })).toBeVisible();
    await expect(page.getByRole("navigation", { name: "Navigation principale" })).toBeVisible();
    await expectNoBlockingA11yViolations(page);
  });
});
