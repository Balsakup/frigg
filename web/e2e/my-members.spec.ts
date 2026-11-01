import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Âge attendu à la date d'exécution pour une naissance le 12 avril 2015.
function ageOn(birth: Date, today = new Date()) {
  let age = today.getFullYear() - birth.getFullYear();
  const before =
    today.getMonth() < birth.getMonth() ||
    (today.getMonth() === birth.getMonth() && today.getDate() < birth.getDate());
  if (before) age--;
  return age;
}

test.describe("mes membres (005)", () => {
  test.describe.configure({ mode: "serial" });

  test("ajouter, modifier et supprimer ses fiches (US1)", async ({ page }) => {
    await login(page, "membre-fiches@exemple.fr");
    const main = page.locator("#contenu");

    await page.getByRole("link", { name: "Mes membres" }).first().click();
    await expect(page.getByRole("heading", { level: 1, name: "Mes membres" })).toBeVisible();
    await expect(main.getByText("Un membre est une personne que vous inscrivez")).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    // Sa propre fiche, avec l'adresse du compte proposée.
    await main.getByRole("link", { name: "Ajouter un membre" }).click();
    await page.getByLabel("Il s'agit de moi").check();
    await expect(page.getByLabel("Adresse électronique")).toHaveValue("membre-fiches@exemple.fr");
    await page.getByLabel(/^Prénom/).fill("Sophie");
    await page.getByLabel(/^Nom/).fill("Lambert");
    await page.getByLabel(/^Date de naissance/).fill("1984-03-02");
    await expectNoBlockingA11yViolations(page);
    await page.getByRole("button", { name: "Ajouter le membre" }).click();
    await expect(main.getByText("Membre ajouté.").first()).toBeVisible();

    // Un enfant ; erreurs de champ d'abord.
    await main.getByRole("link", { name: "Ajouter un membre" }).click();
    await expect(page.getByLabel("Il s'agit de moi")).toHaveCount(0);
    await page.getByLabel(/^Nom/).fill("Lambert");
    await page.getByLabel(/^Date de naissance/).fill("2015-04-12");
    await page.getByLabel("Adresse", { exact: true }).fill("1 rue de la Paix");
    await page.getByLabel("Code postal").fill("7500");
    await page.getByLabel("Commune").fill("Paris");
    await page.getByRole("button", { name: "Ajouter le membre" }).click();
    await expect(main.getByText("Saisissez le prénom.")).toBeVisible();
    await expect(main.getByText("Saisissez un code postal à 5 chiffres.")).toBeVisible();
    await expect(page.getByLabel(/^Prénom/)).toHaveAttribute("aria-invalid", "true");
    await page.getByLabel(/^Prénom/).fill("Léo");
    await page.getByLabel("Code postal").fill("75001");
    await page.getByRole("button", { name: "Ajouter le membre" }).click();
    await expect(main.getByText("Membre ajouté.").first()).toBeVisible();

    const cards = main.getByRole("list", { name: "Mes membres" }).getByRole("listitem");
    await expect(cards).toHaveCount(2);
    await expect(cards.first()).toContainText("Sophie Lambert");
    await expect(cards.first()).toContainText("Vous");
    await expect(cards.nth(1)).toContainText("Léo Lambert");
    await expect(cards.nth(1)).toContainText(`${ageOn(new Date(2015, 3, 12))} ans`);
    await expectNoBlockingA11yViolations(page);

    // Modification du téléphone : normalisé puis affiché par paires.
    await main.getByRole("link", { name: "Léo Lambert" }).click();
    await page.getByLabel("Téléphone").fill("06.12.34.56.78");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(main.getByText("Modifications enregistrées.").first()).toBeVisible();
    await main.getByRole("link", { name: "Léo Lambert" }).click();
    await expect(page.getByLabel("Téléphone")).toHaveValue("06 12 34 56 78");

    // Doublon probable : confirmation demandée.
    await page.goto("/mes-membres/nouveau");
    await page.getByLabel(/^Prénom/).fill("léo");
    await page.getByLabel(/^Nom/).fill("LAMBERT");
    await page.getByLabel(/^Date de naissance/).fill("2015-04-12");
    await page.getByRole("button", { name: "Ajouter le membre" }).click();
    const confirm = page.getByLabel("Enregistrer quand même");
    await expect(confirm).toBeFocused();
    await confirm.check();
    await page.getByRole("button", { name: "Ajouter le membre" }).click();
    await expect(main.getByText("Membre ajouté.").first()).toBeVisible();
    await expect(cards).toHaveCount(3);

    // Suppression confirmée.
    await cards.nth(2).getByRole("link").click();
    await page.getByRole("button", { name: "Supprimer" }).click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toContainText("effacées définitivement");
    await expectNoBlockingA11yViolations(page);
    await dialog.getByRole("button", { name: "Supprimer" }).click();
    await expect(main.getByText("Membre supprimé.").first()).toBeVisible();
    await expect(cards).toHaveCount(2);
  });

  test("ne jamais voir les fiches d'un autre compte (US2)", async ({ page, context }) => {
    await login(page, "membre-fiches@exemple.fr");
    await page.goto("/mes-membres");
    await page.locator("#contenu").getByRole("link", { name: "Léo Lambert" }).click();
    await expect(page).toHaveURL(/\/mes-membres\/\d+$/);
    const url = page.url();

    await context.clearCookies();
    await login(page, "membre-fiches-voisin@exemple.fr");
    await page.goto(url);
    await expect(page.getByRole("heading", { level: 1, name: "Page introuvable" })).toBeVisible();
    await page.goto("/mes-membres");
    await expect(page.locator("#contenu").getByText("Léo Lambert")).toHaveCount(0);
    await page.goto("/admin/membres");
    await expect(page.getByRole("heading", { level: 1, name: "Accès refusé" })).toBeVisible();
  });
});
