import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

test.describe("comptes : consulter et retrouver (002, US1)", () => {
  test("recherche, filtres, pagination et fiche", async ({ page }) => {
    await login(page, "admin-comptes@exemple.fr");
    await page.goto("/admin/comptes");
    await expect(page.getByRole("heading", { level: 1, name: "Comptes" })).toBeVisible();
    const table = page.getByRole("table", { name: "Comptes de l'association" });
    await expect(table).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    // Moins de 3 caractères : indication, liste non filtrée.
    const search = page.getByLabel("Rechercher une adresse");
    await search.fill("du");
    await expect(page.getByText("Saisissez au moins 3 caractères.")).toBeVisible();
    await expect(page).not.toHaveURL(/q=/);

    // Total et pagination sur un ensemble stable (30 comptes membre-liste-NN).
    await search.fill("membre-liste");
    await expect(page).toHaveURL(/q=membre-liste/);
    await expect(page.getByRole("status").filter({ hasText: "30 comptes" })).toBeVisible();
    await expect(page.getByText("Page 1 sur 2")).toBeVisible();
    const rows = table.getByRole("row");
    await expect(rows.nth(1)).toContainText("membre-liste-01@exemple.fr");
    await page.getByRole("link", { name: /Suivante/ }).click();
    await expect(page).toHaveURL(/page=2/);
    await expect(page.getByText("Page 2 sur 2")).toBeVisible();
    await expect(rows.nth(1)).toContainText("membre-liste-26@exemple.fr");

    // Fragment au milieu d'un mot, sans accents ni casse.
    await search.fill("LEFE");
    await expect(table.getByRole("link", { name: "hélène.lefèvre@exemple.fr" })).toBeVisible();
    await expect(page.getByRole("status").filter({ hasText: "1 compte" })).toBeVisible();

    // Filtres combinés, conservés au rechargement, puis effacés.
    await search.fill("");
    const criteria = page.getByRole("search");
    await criteria.getByRole("combobox", { name: "État" }).click();
    await page.getByRole("option", { name: "Désactivés" }).click();
    await criteria.getByRole("combobox", { name: "Rôle" }).click();
    await page.getByRole("option", { name: "Membres" }).click();
    await expect(page).toHaveURL(/statut=desactive/);
    await expect(page).toHaveURL(/role=membre/);
    await page.reload();
    await expect(table.getByRole("link", { name: "membre-desactive@exemple.fr" })).toBeVisible();
    await expect(table.getByText("Désactivé").first()).toBeVisible();
    await page.getByRole("button", { name: "Effacer les critères" }).click();
    await expect(page).not.toHaveURL(/statut=/);

    // Fiche d'un compte et fil d'Ariane.
    await search.fill("membre-liste-07");
    await table.getByRole("link", { name: "membre-liste-07@exemple.fr" }).click();
    await expect(
      page.getByRole("heading", { level: 1, name: "membre-liste-07@exemple.fr" }),
    ).toBeVisible();
    const crumbs = page.getByRole("navigation", { name: "Fil d'Ariane" });
    await expect(crumbs.getByRole("link", { name: "Comptes" })).toHaveAttribute(
      "href",
      "/admin/comptes",
    );
    await expect(page.getByText("Jamais")).toBeVisible();
    await expectNoBlockingA11yViolations(page);
  });

  test("sa propre fiche ne propose aucune action", async ({ page }) => {
    await login(page, "admin-comptes@exemple.fr");
    await page.goto("/admin/comptes?q=admin-comptes");
    await page.getByRole("link", { name: "admin-comptes@exemple.fr" }).click();
    await expect(
      page.getByText("Vous ne pouvez pas modifier votre propre compte ici."),
    ).toBeVisible();
    await expect(page.getByRole("button", { name: "Désactiver" })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Supprimer" })).toHaveCount(0);
  });

  test("320 px : aucun défilement horizontal de la page", async ({ page }) => {
    await page.setViewportSize({ width: 320, height: 800 });
    await login(page, "admin-comptes@exemple.fr");
    await page.goto("/admin/comptes");
    await expect(page.getByRole("table", { name: "Comptes de l'association" })).toBeVisible();
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await expectNoBlockingA11yViolations(page);
  });

  test("un membre n'accède pas à la liste", async ({ page }) => {
    await login(page, "membre-navigation-3@exemple.fr");
    await page.goto("/admin/comptes");
    await expect(page.getByRole("heading", { level: 1, name: "Accès refusé" })).toBeVisible();
  });
});
