import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Paramétrage de la carte bancaire (008, US2), serveur « carte » : identifiants
// fournis par la configuration, faux HelloAsso local.

test("paramètres HelloAsso : adresse, test de connexion, activation", async ({ page, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await login(page, "admin-carte-parametres@exemple.fr");
  await page.goto("/admin/parametres/paiements");
  const card = page.getByRole("group", { name: "Carte bancaire (HelloAsso)" });
  await expect(card.getByText("Mode test", { exact: true })).toBeVisible();
  await expect(card.getByText("sentiers-du-val")).toBeVisible();
  await expect(card.getByText(/\/api\/webhooks\/helloasso\/[0-9a-f]{32}/)).toBeVisible();
  await expect(card.getByText(/secret/i)).toHaveCount(0);
  await expectNoBlockingA11yViolations(page);

  await card.getByRole("button", { name: /Copier/ }).click();
  await expect(card.getByText("Copié.")).toBeAttached();

  await card.getByRole("button", { name: "Tester la connexion" }).click();
  await expect(
    card.getByText(/Connexion à HelloAsso réussie — Les Sentiers du Val \(test\)/),
  ).toBeVisible();

  const toggle = card.getByRole("switch", { name: "Proposer ce moyen" });
  await expect(toggle).toBeChecked();
  await toggle.click();
  await page.getByRole("button", { name: "Enregistrer" }).click();
  await expect(page.getByText("Paramètres enregistrés.").first()).toBeVisible();
  await expect(toggle).not.toBeChecked();
  await toggle.click();
  await page.getByRole("button", { name: "Enregistrer" }).click();
  await expect(toggle).toBeChecked();
  await expectNoBlockingA11yViolations(page);
});
