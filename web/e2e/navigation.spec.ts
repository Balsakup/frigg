import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

test.describe("navigation adaptée au rôle (US2)", () => {
  test("un membre ne voit ni sélecteur ni administration", async ({ page }) => {
    await login(page, "membre-navigation-1@exemple.fr");
    await expect(page.getByRole("navigation", { name: "Changer d'espace" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Tableau de bord" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Comptes" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Paramètres" })).toHaveCount(0);
    await expect(page.getByRole("link", { name: "Accueil" })).toHaveAttribute(
      "aria-current",
      "page",
    );

    await page.goto("/admin");
    await expect(page.getByRole("heading", { level: 1, name: "Accès refusé" })).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    await page.getByRole("link", { name: "Mes documents" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Mes documents" })).toBeVisible();
    await expect(page.getByText("Bientôt disponible")).toBeVisible();
    await expectNoBlockingA11yViolations(page);
  });

  test("un administrateur bascule entre les espaces et retrouve le dernier choisi", async ({
    page,
  }) => {
    await login(page, "admin-navigation@exemple.fr");
    await expect(page).toHaveURL("/admin");
    await expect(page.getByRole("heading", { level: 1, name: "Tableau de bord" })).toBeVisible();
    await expect(page.getByText("Espace administration", { exact: true })).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    const switcher = page.getByRole("navigation", { name: "Changer d'espace" });
    await switcher.getByRole("link", { name: /Membre/ }).click();
    await expect(page).toHaveURL("/");
    await expect(page.getByText("Espace membre", { exact: true })).toBeVisible();
    await expect(page.getByRole("link", { name: "Mes membres" })).toBeVisible();
    await expect(switcher.getByRole("link", { name: /Membre/ })).toHaveAttribute(
      "aria-current",
      "true",
    );

    // Ouvrir une adresse d'administration fait basculer le sélecteur.
    await page.goto("/admin/finances");
    await expect(switcher.getByRole("link", { name: /Admin/ })).toHaveAttribute(
      "aria-current",
      "true",
    );
    await switcher.getByRole("link", { name: /Membre/ }).click();

    // Après reconnexion, le dernier espace choisi sur l'appareil est conservé.
    await page.getByRole("button", { name: /menu du compte/ }).click();
    await page.getByRole("menuitem", { name: "Se déconnecter" }).click();
    await login(page, "admin-navigation@exemple.fr");
    await expect(page).toHaveURL("/");
  });

  test("la barre latérale repliée est mémorisée", async ({ page }) => {
    await login(page, "membre-navigation-2@exemple.fr");
    await page.getByRole("button", { name: "Menu de navigation" }).click();
    await expect(page.getByRole("button", { name: "Menu de navigation" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    await page.reload();
    await expect(page.getByRole("button", { name: "Menu de navigation" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    // Les libellés restent accessibles en mode icônes.
    await expect(page.getByRole("link", { name: "Mes membres" })).toBeVisible();
  });

  test("sur téléphone, le menu s'ouvre en surimpression et se ferme avec Échap", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 320, height: 640 });
    await login(page, "membre-navigation-3@exemple.fr");
    await expect(page.getByRole("navigation", { name: "Navigation principale" })).toHaveCount(0);

    await page.getByRole("button", { name: "Menu de navigation" }).click();
    const drawer = page.getByRole("dialog", { name: "Menu de navigation" });
    await expect(drawer).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(drawer).toBeHidden();

    const scrollWidth = await page.evaluate(() => document.documentElement.scrollWidth);
    expect(scrollWidth).toBeLessThanOrEqual(320);
  });
});
