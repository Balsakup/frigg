import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

test.describe("accessibilité transverse (SC-005, SC-007)", () => {
  test("parcours complet au clavier seul", async ({ page }) => {
    await login(page, "admin-clavier@exemple.fr");

    // Lien d'évitement vers le contenu.
    await page.keyboard.press("Tab");
    const skip = page.getByRole("link", { name: "Aller au contenu" });
    await expect(skip).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page.locator("#contenu")).toBeFocused();

    // Sélecteur d'espace au clavier.
    const memberSpace = page
      .getByRole("navigation", { name: "Changer d'espace" })
      .getByRole("link", { name: /Membre/ });
    await memberSpace.focus();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL("/");
    await expect(page.getByRole("heading", { level: 1, name: "Accueil" })).toBeVisible();

    // Menu du compte : ouverture, flèches, déconnexion.
    await page.getByRole("button", { name: /menu du compte/ }).focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("menuitem", { name: "Mon compte" })).toBeFocused();
    await page.keyboard.press("ArrowDown");
    await expect(page.getByRole("menuitem", { name: "Se déconnecter" })).toBeFocused();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL("/connexion");
  });

  test("aucun défilement horizontal à 320 px", async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 640 });
    for (const path of ["/connexion", "/mot-de-passe-oublie"]) {
      await page.goto(path);
      await expect(page.locator("h1")).toBeVisible();
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth),
        path,
      ).toBeLessThanOrEqual(320);
    }
    await login(page, "membre-clavier@exemple.fr");
    for (const path of ["/", "/mon-compte", "/mes-membres", "/admin", "/page-inconnue"]) {
      await page.goto(path);
      await expect(page.locator("#contenu h1")).toBeVisible();
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth),
        path,
      ).toBeLessThanOrEqual(320);
      await expectNoBlockingA11yViolations(page);
    }
  });

  test("curseur main sur tous les éléments cliquables", async ({ page }) => {
    const cursor = (selector: ReturnType<typeof page.locator>) =>
      selector.evaluate((el) => getComputedStyle(el).cursor);

    await page.goto("/connexion");
    expect(await cursor(page.getByRole("button", { name: "Se connecter" }))).toBe("pointer");
    expect(await cursor(page.getByRole("link", { name: "Mot de passe oublié ?" }))).toBe("pointer");

    await login(page, "membre-clavier@exemple.fr");
    expect(await cursor(page.getByRole("button", { name: "Menu de navigation" }))).toBe("pointer");
    expect(await cursor(page.getByRole("link", { name: "Mes membres" }))).toBe("pointer");
    await page.getByRole("button", { name: /menu du compte/ }).click();
    expect(await cursor(page.getByRole("menuitem", { name: "Se déconnecter" }))).toBe("pointer");
    await page.keyboard.press("Escape");

    // Un bouton occupé (aria-disabled) garde le curseur « interdit ».
    await page.evaluate(() => {
      const b = document.createElement("button");
      b.id = "occupe";
      b.className = "aria-disabled:cursor-not-allowed";
      b.setAttribute("aria-disabled", "true");
      document.body.append(b);
    });
    expect(await cursor(page.locator("#occupe"))).toBe("not-allowed");
  });
});
