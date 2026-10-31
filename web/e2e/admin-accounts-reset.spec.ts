import { waitForLink } from "./mailpit";
import { expect, login, test } from "./test";

test("envoyer un lien de réinitialisation à une personne (002, US5)", async ({ page, browser }) => {
  const email = "membre-lien@exemple.fr";
  await login(page, "admin-lien@exemple.fr");
  await page.goto(`/admin/comptes?q=${email}`);
  await page.getByRole("link", { name: email }).click();
  const since = new Date();
  await page.getByRole("button", { name: "Envoyer un lien de réinitialisation" }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Envoyer" }).click();
  await expect(page.getByText(`Lien de réinitialisation envoyé à ${email}.`).first()).toBeVisible();

  const guest = await (await browser.newContext({ locale: "fr-FR" })).newPage();
  await guest.goto(await waitForLink(email, since));
  await guest.getByLabel(/^Mot de passe/).fill("Envoyepar2026");
  await guest.getByLabel(/^Confirmez le mot de passe/).fill("Envoyepar2026");
  await guest.getByRole("button", { name: "Enregistrer le mot de passe" }).click();
  await expect(guest.getByText("Votre mot de passe a été modifié.")).toBeVisible();
  await login(guest, email, "Envoyepar2026");

  // Pas d'envoi possible vers un compte désactivé.
  await page.goto("/admin/comptes?q=membre-desactive");
  await page.getByRole("link", { name: "membre-desactive@exemple.fr" }).click();
  await expect(page.getByRole("button", { name: "Réactiver" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Envoyer un lien de réinitialisation" }),
  ).toHaveCount(0);
});
