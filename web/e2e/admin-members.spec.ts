import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Les noms « Quéméner » et « Rattaché » ne sont utilisés que par ce fichier :
// la liste d'administration montre les fiches de tous les tests.
test.describe("membres (005)", () => {
  test.describe.configure({ mode: "serial" });

  test("retrouver et consulter les membres (US3)", async ({ page, context }) => {
    // Fiches préparées dans l'espace d'un membre.
    await login(page, "membre-rattachement@exemple.fr");
    for (const [first, birth] of [
      ["Élodie", "2012-05-01"],
      ["Zoé", "2014-06-01"],
    ]) {
      await page.goto("/mes-membres/nouveau");
      await page.getByLabel(/^Prénom/).fill(first);
      await page.getByLabel(/^Nom/).fill("Quéméner");
      await page.getByLabel(/^Date de naissance/).fill(birth);
      await page.getByRole("button", { name: "Ajouter le membre" }).click();
      await expect(page.getByText("Membre ajouté.").first()).toBeVisible();
    }

    await context.clearCookies();
    await login(page, "admin-membres@exemple.fr");
    await page.getByRole("link", { name: "Membres", exact: true }).first().click();
    await expect(page.getByRole("heading", { level: 1, name: "Membres" })).toBeVisible();
    const main = page.locator("#contenu");

    await page.getByLabel("Rechercher un membre").fill("qu");
    await expect(main.getByText("Saisissez au moins 3 caractères.")).toBeVisible();
    await page.getByLabel("Rechercher un membre").fill("quemen");
    await expect(page).toHaveURL(/q=quemen/);
    const rows = main.getByRole("row");
    await expect(rows).toHaveCount(3);
    await expect(rows.nth(1)).toContainText("Quéméner Élodie");
    await expect(rows.nth(1)).toContainText("membre-rattachement@exemple.fr");
    await expect(rows.nth(2)).toContainText("Quéméner Zoé");
    await expectNoBlockingA11yViolations(page);

    await main.getByRole("link", { name: "Quéméner Élodie" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Élodie Quéméner" })).toBeVisible();
    await expect(
      main.getByRole("link", { name: "membre-rattachement@exemple.fr" }),
    ).toHaveAttribute("href", /\/admin\/comptes\/\d+$/);
    await expectNoBlockingA11yViolations(page);
  });

  test("créer, rattacher, modifier et supprimer un membre (US4)", async ({ page, context }) => {
    await login(page, "admin-membres@exemple.fr");
    const main = page.locator("#contenu");

    // Fiche sans compte.
    await page.goto("/admin/membres/nouveau");
    const newPicker = page.getByLabel("Compte de rattachement");
    await expect(newPicker).toHaveValue("");
    await expect(newPicker).toHaveAttribute("placeholder", "Sans compte");
    await page.getByLabel(/^Prénom/).fill("Jeanne");
    await page.getByLabel(/^Nom/).fill("Rattaché");
    await page.getByLabel(/^Date de naissance/).fill("2010-01-01");
    await page.getByRole("button", { name: "Ajouter le membre" }).click();
    await expect(main.getByText("Membre ajouté.").first()).toBeVisible();

    await page.goto("/admin/membres?rattachement=sans-compte&q=rattach");
    await expect(main.getByRole("row").nth(1)).toContainText("Rattaché Jeanne");
    await expect(main.getByRole("row").nth(1)).toContainText("Sans compte");

    // Rattachement au compte du membre par le Combobox, au clavier.
    await main.getByRole("link", { name: "Rattaché Jeanne" }).click();
    const picker = page.getByLabel("Compte de rattachement");
    await picker.fill("membre-ratt");
    const option = page.getByRole("option", { name: "membre-rattachement@exemple.fr" });
    await expect(option).toBeVisible();
    await picker.press("ArrowDown");
    await option.click();
    await expect(picker).toHaveValue("membre-rattachement@exemple.fr");
    // Contrôle avec la liste refermée : ouverte, base-ui masque le reste de la
    // page (aria-hidden) pendant que le focus y est piégé, ce qu'axe signale.
    await expectNoBlockingA11yViolations(page);
    await page.getByLabel(/^Date de naissance/).fill("2010-02-01");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(main.getByText("Modifications enregistrées.").first()).toBeVisible();

    // La fiche apparaît dans l'espace du membre.
    await context.clearCookies();
    await login(page, "membre-rattachement@exemple.fr");
    await page.goto("/mes-membres");
    await expect(page.locator("#contenu").getByText("Jeanne Rattaché")).toBeVisible();

    // Suppression par l'administration.
    await context.clearCookies();
    await login(page, "admin-membres@exemple.fr");
    await page.goto("/admin/membres?q=rattach");
    await main.getByRole("link", { name: "Rattaché Jeanne" }).click();
    await page.getByRole("button", { name: "Supprimer" }).click();
    const dialog = page.getByRole("alertdialog");
    await expectNoBlockingA11yViolations(page);
    await dialog.getByRole("button", { name: "Supprimer" }).click();
    await expect(main.getByText("Membre supprimé.").first()).toBeVisible();
  });
});
