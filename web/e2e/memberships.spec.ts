import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, PASSWORD, test } from "./test";

// Adhésions, espace membre (006 : US1, US3, US4). Serveur des adhésions :
// saison ouverte, tarifs « Adulte » (60 €, renouvellement 40 €), « Jeune »
// (30 €), « Bienfaiteur » (gratuit), trois moyens de paiement activés.

test.describe("adhésions — espace membre (006)", () => {
  test("demande, suivi et annulation", async ({ page }) => {
    await login(page, "membre-adhesions-demande@exemple.fr");
    const main = page.locator("#contenu");

    await test.step("US1 — rubrique et formulaire", async () => {
      await page.getByRole("link", { name: "Adhésions" }).click();
      await expect(page.getByRole("heading", { level: 1, name: "Adhésions" })).toBeVisible();
      await expect(main.getByText("Aucune adhésion pour le moment.")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("link", { name: "Demander une adhésion" }).click();
      await expect(
        page.getByRole("heading", { level: 1, name: "Demander une adhésion" }),
      ).toBeVisible();
    });

    await test.step("US1 — montant selon le tarif et le type déclaré", async () => {
      await main.getByRole("radio", { name: "Léo Martin" }).click();
      await main.getByRole("radio", { name: /Adulte/ }).click();
      await main.getByRole("radio", { name: "Oui, renouvellement" }).click();
      await expect(main.getByText("Montant : 40,00 €")).toBeVisible();
      await main.getByRole("radio", { name: "Non, première adhésion" }).click();
      await expect(main.getByText("Montant : 60,00 €")).toBeVisible();
      await main.getByRole("radio", { name: "Oui, renouvellement" }).click();
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("button", { name: "Envoyer la demande" }).click();
      await expect(page).toHaveURL("/adhesions");
      await expect(main.getByText("Demande envoyée.").first()).toBeVisible();
      const card = main.getByRole("listitem").filter({ hasText: "Léo Martin" });
      await expect(card.getByText("En attente de validation")).toBeVisible();
      await expect(card.getByText(/40,00/)).toBeVisible();
    });

    await test.step("US1 — une fiche déjà en cours n'est plus proposée", async () => {
      await main.getByRole("link", { name: "Demander une adhésion" }).click();
      await expect(main.getByRole("radio", { name: "Léo Martin" })).toHaveCount(0);
      await expect(
        main.getByText("Léo a déjà une adhésion en cours pour cette saison."),
      ).toBeVisible();
      await main.getByRole("radio", { name: "Inès Martin" }).click();
      await main.getByRole("radio", { name: /Jeune/ }).click();
      await main.getByRole("radio", { name: "Non, première adhésion" }).click();
      await main.getByRole("button", { name: "Envoyer la demande" }).click();
      await expect(main.getByText("Demande envoyée.").first()).toBeVisible();
    });

    await test.step("US3 — annulation d'une demande en attente", async () => {
      await main.getByRole("link", { name: "Inès Martin" }).click();
      await expect(page.getByRole("heading", { level: 1, name: /Inès Martin/ })).toBeVisible();
      await main.getByRole("button", { name: "Annuler la demande" }).click();
      const dialog = page.getByRole("alertdialog");
      await expect(dialog).toContainText("Annuler la demande d'adhésion de Inès ?");
      await expectNoBlockingA11yViolations(page);
      await dialog.getByRole("button", { name: "Annuler la demande" }).click();
      await expect(page).toHaveURL("/adhesions");
      await expect(main.getByText("Demande annulée.").first()).toBeVisible();
      await expect(
        main.getByRole("listitem").filter({ hasText: "Inès Martin" }).getByText("Annulée"),
      ).toBeVisible();
    });

    await test.step("US3 — section « Adhésions » de la fiche", async () => {
      await page.getByRole("link", { name: "Mes membres" }).click();
      await expect(page.getByRole("heading", { level: 1, name: "Mes membres" })).toBeVisible();
      await page.locator("#contenu").getByRole("link", { name: "Inès Martin" }).click();
      const section = main.getByRole("region", { name: "Adhésions" });
      await expect(section.getByText("Non adhérent")).toBeVisible();
      await expect(section.getByRole("link", { name: "Demander l'adhésion" })).toBeVisible();
    });
  });

  test("paiement d'une adhésion validée", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);
    const main = page.locator("#contenu");

    await test.step("US4 — le lien de l'e-mail ramène sur la page après connexion", async () => {
      await page.goto("/adhesions/1");
      await expect(page).toHaveURL(/\/connexion/);
      await page.getByLabel("Adresse électronique").fill("membre-adhesions-paiement@exemple.fr");
      await page.getByLabel(/^Mot de passe/).fill(PASSWORD);
      await page.getByRole("button", { name: "Se connecter" }).click();
      await expect(page).toHaveURL("/adhesions/1");
      await expect(page.getByRole("heading", { level: 1, name: /Hugo Bernard/ })).toBeVisible();
    });

    await test.step("US4 — informations de chaque moyen", async () => {
      const section = main.getByRole("region", { name: "Paiement" });
      await expect(section.getByText(/Montant : 60,00/)).toBeVisible();
      await section.getByRole("radio", { name: "Espèces" }).click();
      await expect(section.getByText(/le samedi de 10 h à 12 h/)).toBeVisible();
      await section.getByRole("radio", { name: "Chèque" }).click();
      await expect(section.getByText("Les Sentiers du Val")).toBeVisible();
      await expect(section.getByText(/au dos du chèque/)).toBeVisible();
      await section.getByRole("radio", { name: "Virement" }).click();
      await expect(section.getByText("FR76 3000 6000 0112 3456 7890 189")).toBeVisible();
      await expect(section.getByText("BNPAFRPP")).toBeVisible();
      await section.getByRole("button", { name: "Copier IBAN" }).click();
      await expect(section.getByRole("button", { name: "Copié IBAN" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("US4 — confirmation du moyen", async () => {
      await main.getByRole("button", { name: "Confirmer ce moyen de paiement" }).click();
      await expect(main.getByText("Moyen de paiement enregistré.").first()).toBeVisible();
      await expect(main.getByText("Paiement annoncé — virement").first()).toBeVisible();
      await expect(main.getByRole("button", { name: "Changer de moyen" })).toBeVisible();
    });
  });

  test("une adhésion d'un autre compte est introuvable", async ({ page }) => {
    await login(page, "membre-adhesions-voisin@exemple.fr");
    await page.goto("/adhesions/1");
    await expect(page.getByRole("heading", { level: 1, name: "Page introuvable" })).toBeVisible();
  });
});
