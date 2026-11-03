import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Paramètres › Général (010, US1, US2, US4). Projet « parametres-generaux » :
// instance dédiée, car le nom et la tagline sont globaux.

test.describe.configure({ mode: "serial" });

test.describe("paramètres généraux (010)", () => {
  test("nom et tagline (US1)", async ({ page, browser }) => {
    await login(page, "admin@exemple.fr");
    await page.goto("/admin/parametres");
    await expect(page).toHaveURL("/admin/parametres/general");
    await expect(
      page
        .getByRole("navigation", { name: "Sections des paramètres" })
        .getByRole("link", { name: "Général" }),
    ).toHaveAttribute("aria-current", "page");

    const name = page.getByLabel("Nom de l'association");
    await expect(name).toHaveValue("Les Sentiers du Val");
    await name.fill("Les Sentiers du Val d'Oise");
    await page.getByLabel(/^Tagline/).fill("Randonnées et sorties nature depuis 1987");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(page.getByText("Paramètres enregistrés.", { exact: true }).last()).toBeVisible();

    // Barre latérale et titre à jour sans rechargement.
    const sidebar = page.locator("aside").first();
    await expect(sidebar.getByText("Les Sentiers du Val d'Oise")).toBeVisible();
    await expect(sidebar.getByText("Randonnées et sorties nature depuis 1987")).toBeVisible();
    await expect(page).toHaveTitle(/· Les Sentiers du Val d'Oise$/);
    await expectNoBlockingA11yViolations(page);

    // Nom vide : refusé, message sous le champ.
    await name.fill("   ");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(page.getByText("Indiquez le nom de l'association.")).toBeVisible();
    await expect(name).toBeFocused();

    // Quitter avec des modifications : confirmation.
    await page
      .getByRole("navigation", { name: "Sections des paramètres" })
      .getByRole("link", { name: "Paiements" })
      .click();
    const dialog = page.getByRole("alertdialog", { name: "Quitter sans enregistrer ?" });
    await expect(dialog).toBeVisible();
    await dialog.getByRole("button", { name: "Annuler" }).click();
    await expect(page).toHaveURL("/admin/parametres/general");
    await name.fill("Les Sentiers du Val d'Oise");

    // Écran étroit : pas de défilement horizontal.
    await page.setViewportSize({ width: 320, height: 800 });
    await expect
      .poll(() =>
        page.evaluate(
          () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
        ),
      )
      .toBe(false);

    // Membre : tagline dans la barre latérale et sur l'accueil, paramètres refusés.
    const ctx = await browser.newContext();
    const member = await ctx.newPage();
    await login(member, "membre@exemple.fr");
    await expect(
      member.locator("#contenu").getByText("Randonnées et sorties nature depuis 1987"),
    ).toBeVisible();
    await expect(
      member.locator("aside").first().getByText("Randonnées et sorties nature depuis 1987"),
    ).toBeVisible();
    await member.goto("/admin/parametres/general");
    await expect(member.getByLabel("Nom de l'association")).toHaveCount(0);
    await ctx.close();
  });

  test("identité légale et coordonnées (US2)", async ({ page }) => {
    await login(page, "admin@exemple.fr");
    await page.goto("/admin/parametres/general");
    await page.getByLabel(/^Numéro RNA/).fill("w751234567");
    await page.getByLabel(/^Numéro SIRET/).fill("732 829 320 00074");
    await page.getByLabel(/^Date de création/).fill("1987-05-12");
    await page.getByLabel(/^Adresse du siège/).fill("1 rue du Moulin");
    await page.getByLabel("Code postal").fill("95000");
    await page.getByLabel("Ville").fill("Cergy");
    await page.getByLabel(/^Téléphone/).fill("+33 1 23 45 67 89");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(page.getByText("Paramètres enregistrés.", { exact: true }).last()).toBeVisible();
    await page.reload();
    await expect(page.getByLabel(/^Numéro RNA/)).toHaveValue("W751234567");
    await expect(page.getByLabel(/^Numéro SIRET/)).toHaveValue("73282932000074");
    await expect(page.getByLabel(/^Téléphone/)).toHaveValue("01 23 45 67 89");

    // Trois erreurs ensemble, rien d'enregistré.
    await page.getByLabel(/^Numéro SIRET/).fill("73282932000075");
    await page.getByLabel(/^Date de création/).fill("2999-01-01");
    await page.getByLabel(/^Site web/).fill("www.exemple.fr");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(
      page.getByText("Ce numéro SIRET n'est pas valide.", { exact: false }),
    ).toBeVisible();
    await expect(
      page.getByText("La date de création ne peut pas être dans le futur."),
    ).toBeVisible();
    await expect(
      page.getByText("Saisissez une adresse web complète, commençant par https://."),
    ).toBeVisible();
    await expectNoBlockingA11yViolations(page);
    await page.reload();
    await expect(page.getByLabel(/^Numéro SIRET/)).toHaveValue("73282932000074");
    await expect(page.getByLabel(/^Site web/)).toHaveValue("");
  });

  test("contact présenté aux membres (US4)", async ({ page, browser }) => {
    await login(page, "admin@exemple.fr");
    await page.goto("/admin/parametres/general");
    await page.getByLabel(/^Adresse électronique de contact/).fill("bureau@sentiers.example");
    await page.getByLabel(/^Lien Instagram/).fill("instagram.com/sentiers");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(
      page.getByText("Saisissez une adresse web complète, commençant par https://."),
    ).toBeVisible();
    await page.getByLabel(/^Lien Instagram/).fill("https://instagram.com/sentiers");
    await page.getByLabel(/^Lien Facebook/).fill("https://facebook.com/sentiers");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(page.getByText("Paramètres enregistrés.", { exact: true }).last()).toBeVisible();

    const ctx = await browser.newContext();
    const member = await ctx.newPage();
    await login(member, "membre@exemple.fr");
    const card = member.getByRole("region", { name: "Contacter l'association" });
    await expect(card).toBeVisible();
    await expect(card.getByRole("link", { name: "bureau@sentiers.example" })).toHaveAttribute(
      "href",
      "mailto:bureau@sentiers.example",
    );
    await expect(card.getByRole("link", { name: /01 23 45 67 89/ })).toHaveAttribute(
      "href",
      "tel:+33123456789",
    );
    const instagram = card.getByRole("link", { name: /Instagram de/ });
    await expect(instagram).toHaveAttribute("href", "https://instagram.com/sentiers");
    await expect(instagram).toHaveAttribute("rel", "noopener noreferrer");
    await expectNoBlockingA11yViolations(member);

    // Tout vider : encart absent.
    for (const label of [
      /^Adresse électronique de contact/,
      /^Lien Instagram/,
      /^Lien Facebook/,
      /^Téléphone/,
      /^Adresse du siège/,
      "Code postal",
      "Ville",
    ]) {
      await page.getByLabel(label).fill("");
    }
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(page.getByText("Paramètres enregistrés.", { exact: true }).last()).toBeVisible();
    await member.reload();
    await expect(member.getByRole("heading", { level: 1, name: "Accueil" })).toBeVisible();
    await expect(member.getByRole("region", { name: "Contacter l'association" })).toHaveCount(0);
    await ctx.close();
  });
});
