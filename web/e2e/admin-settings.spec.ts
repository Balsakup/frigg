import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Paramètres des moyens de paiement (006, US5). Projet « adhesions-parametres » :
// passe après les tests des adhésions, car les paramètres sont globaux.

test.describe("paramètres — moyens de paiement (006)", () => {
  test("paramétrage et encart du tableau de bord", async ({ page }) => {
    await login(page, "admin-adhesions-parametres@exemple.fr");
    const main = page.locator("#contenu");

    // Point de départ : aucun moyen activé.
    const res = await page.request.put("/api/admin/settings/payment", {
      data: { cash: { enabled: false }, cheque: { enabled: false }, transfer: { enabled: false } },
    });
    expect(res.status()).toBe(200);

    await page.goto("/admin");
    await expect(
      main.getByText("Aucun moyen de paiement n'est proposé aux adhérents."),
    ).toBeVisible();
    await main.getByRole("link", { name: "Paramétrer" }).click();
    await expect(page).toHaveURL("/admin/parametres/paiements");
    await expect(page.getByRole("heading", { level: 1, name: "Paramètres" })).toBeVisible();
    await expect(page).toHaveTitle(/^Paiements · Paramètres/);
    await expect(
      page
        .getByRole("navigation", { name: "Sections des paramètres" })
        .getByRole("link", { name: "Paiements" }),
    ).toHaveAttribute("aria-current", "page");
    await expect(
      page.getByRole("navigation", { name: "Fil d'Ariane" }).getByText("Paiements"),
    ).toHaveAttribute("aria-current", "page");
    const settings = page.getByRole("navigation", { name: "Configuration" });
    await expect(settings.getByRole("link", { name: "Paramètres" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    await expectNoBlockingA11yViolations(page);

    const transfer = main.getByRole("group", { name: "Virement" });
    // Activation au clavier.
    await transfer.getByRole("switch", { name: "Proposer ce moyen" }).focus();
    await page.keyboard.press("Space");
    await expect(transfer.getByRole("switch", { name: "Proposer ce moyen" })).toBeChecked();
    await transfer.getByLabel(/Titulaire du compte/).fill("Les Sentiers du Val");
    await transfer.getByLabel(/^IBAN/).fill("");
    await main.getByRole("button", { name: "Enregistrer" }).click();
    await expect(transfer.getByText("Saisissez l'IBAN.")).toBeVisible();
    await transfer.getByLabel(/^IBAN/).fill("FR76 3000 6000 0112 3456 7890 188");
    await main.getByRole("button", { name: "Enregistrer" }).click();
    await expect(transfer.getByText(/Cet IBAN n'est pas valide/)).toBeVisible();
    await transfer.getByLabel(/^IBAN/).fill("FR76 3000 6000 0112 3456 7890 189");
    await main.getByRole("button", { name: "Enregistrer" }).click();
    await expect(main.getByText("Paramètres enregistrés.").first()).toBeVisible();
    await expect(transfer.getByLabel(/^IBAN/)).toHaveValue("FR7630006000011234567890189");
    await expectNoBlockingA11yViolations(page);

    await page.goto("/admin");
    await expect(
      main.getByText("Aucun moyen de paiement n'est proposé aux adhérents."),
    ).toHaveCount(0);

    // L'adresse de la rubrique mène à sa première section.
    await page.goto("/admin/parametres");
    await expect(page).toHaveURL("/admin/parametres/paiements");
  });
});
