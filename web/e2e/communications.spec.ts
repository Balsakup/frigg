import type { Page } from "@playwright/test";
import { expectNoBlockingA11yViolations } from "./axe";
import { waitForMessages } from "./mailpit";
import { expect, login, test } from "./test";

// Communications (009 : US1 à US4). Jeu de données (e2eseed, mode
// « communication ») : Léo et Inès Martin (compte Claire, à payer), Paul Durand
// (paiement annoncé), Anne Petit (sans compte, payée), Hugo Roux (sans compte
// ni adresse, à payer), Zoé Blanc (gratuite), Marc Noir (demande en attente).
// L'historique est partagé : les tests s'enchaînent dans l'ordre.
test.describe.configure({ mode: "serial" });

const stamp = Date.now().toString(36);
const relanceSubject = `Rappel cotisation ${stamp}`;
const piecesSubject = `Convocation AG ${stamp}`;

const pdf = Buffer.from(`%PDF-1.7\n1 0 obj\n<<>>\nendobj\n${"x".repeat(2000)}\n%%EOF\n`);
const png = Buffer.concat([
  Buffer.from("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR", "binary"),
  Buffer.alloc(64),
]);
const exe = Buffer.concat([Buffer.from("MZ\x90\x00", "binary"), Buffer.alloc(200)]);

async function noHorizontalScroll(page: Page) {
  const width = await page.evaluate(() => document.documentElement.scrollWidth);
  expect(width).toBeLessThanOrEqual(320);
}

test.describe("communications (009)", () => {
  test("US1 — relance des adhérents en impayé", async ({ page }) => {
    await login(page, "admin-communication-relance@exemple.fr");
    const main = page.locator("#contenu");

    await test.step("historique vide", async () => {
      await page.goto("/admin/communication");
      await expect(main.getByText("Aucune communication pour l'instant")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("link", { name: "Nouvelle communication" }).first().click();
      await expect(page).toHaveURL(/\/admin\/communication\/nouvelle$/);
    });

    await test.step("cible « En impayé » et non joignables", async () => {
      await expect(main.getByRole("radio", { name: "En impayé" })).toBeChecked();
      await expect(main.getByRole("status").filter({ hasText: "adhérents ciblés" })).toHaveText(
        /4 adhérents ciblés · 2 destinataires/,
      );
      await expect(main.getByText("1 adhérent n'a aucune adresse")).toBeVisible();
      await main.getByRole("button", { name: "Voir les non joignables" }).click();
      const dialog = page.getByRole("dialog");
      await expect(dialog.getByRole("cell", { name: "Hugo", exact: true })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await page.keyboard.press("Escape");
      await main.getByRole("button", { name: "Voir les adhérents ciblés" }).click();
      await expect(
        page.getByRole("dialog").getByRole("cell", { name: "Paul", exact: true }),
      ).toBeVisible();
      await page.keyboard.press("Escape");
    });

    await test.step("rédaction et aperçu", async () => {
      await main.getByRole("textbox", { name: /^Objet/ }).fill(relanceSubject);
      await main
        .getByRole("textbox", { name: /^Message/ })
        .fill(
          "Bonjour,\n\n<b>test</b> : merci de régler votre cotisation.\nInfos : https://exemple.test/infos",
        );
      await main.getByRole("button", { name: "Aperçu" }).click();
      const dialog = page.getByRole("dialog", { name: "Aperçu du message" });
      const frame = dialog.frameLocator("iframe[title='Aperçu du message']");
      await expect(frame.getByText("<b>test</b>", { exact: false })).toBeVisible();
      await expect(frame.getByRole("link", { name: "https://exemple.test/infos" })).toBeVisible();
      await expect(frame.getByText(/en tant qu'adhérent de Les Sentiers du Val/)).toBeVisible();
      await page.keyboard.press("Escape");
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("confirmation et envoi", async () => {
      const since = new Date();
      await main.getByRole("button", { name: "Envoyer", exact: true }).click();
      const dialog = page.getByRole("alertdialog", { name: "Envoyer cette communication ?" });
      await expect(dialog.getByText(relanceSubject)).toBeVisible();
      await expect(dialog.getByText("2 destinataires", { exact: true })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await dialog.getByRole("button", { name: "Envoyer à 2 destinataires" }).click();
      await expect(page).toHaveURL(/\/admin\/communication\/\d+$/);
      await expect(
        main.getByText("Communication envoyée à 2 destinataires.").first(),
      ).toBeVisible();
      await expectNoBlockingA11yViolations(page);

      const messages = await waitForMessages(relanceSubject, since, 2);
      const to = messages.flatMap((m) => m.To.map((t) => t.Address)).sort();
      expect(to).toEqual(["claire.communication@exemple.fr", "paul.communication@exemple.fr"]);
      for (const m of messages) {
        expect(m.From.Name).toBe("Les Sentiers du Val");
        expect(m.To).toHaveLength(1);
      }
    });
  });

  test("US2 — pièces jointes", async ({ page }) => {
    await login(page, "admin-communication-pieces@exemple.fr");
    const main = page.locator("#contenu");
    await page.goto("/admin/communication/nouvelle");
    const input = main.locator("input[type=file]");

    await test.step("ajout, refus et retrait de fichiers", async () => {
      await input.setInputFiles([
        { name: "Convocation AG.pdf", mimeType: "application/pdf", buffer: pdf },
        { name: "plan.png", mimeType: "image/png", buffer: png },
      ]);
      const files = main.getByRole("list", { name: "Fichiers joints" });
      await expect(files.getByText("Convocation AG.pdf")).toBeVisible();
      await expect(main.getByText("2 fichiers ·")).toBeVisible();
      await input.setInputFiles({ name: "facture.pdf", mimeType: "application/pdf", buffer: exe });
      await expect(
        main.getByRole("alert").filter({ hasText: "Ce type de fichier n'est pas accepté" }),
      ).toBeVisible();
      await input.setInputFiles({
        name: "gros.pdf",
        mimeType: "application/pdf",
        buffer: Buffer.concat([pdf, Buffer.alloc(6 * 1024 * 1024)]),
      });
      await expect(
        main.getByRole("alert").filter({ hasText: "Le fichier dépasse 5 Mo." }),
      ).toBeVisible();
      await main.getByRole("button", { name: "Retirer plan.png" }).click();
      await expect(files.getByText("plan.png")).toHaveCount(0);
      await expect(main.getByText("1 fichier ·")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("envoi et réception des pièces", async () => {
      await main.getByRole("textbox", { name: /^Objet/ }).fill(piecesSubject);
      await main
        .getByRole("textbox", { name: /^Message/ })
        .fill("Vous trouverez la convocation en pièce jointe.");
      const since = new Date();
      await main.getByRole("button", { name: "Envoyer", exact: true }).click();
      const dialog = page.getByRole("alertdialog");
      await expect(dialog.getByText(/Convocation AG\.pdf/)).toBeVisible();
      await dialog.getByRole("button", { name: "Envoyer à 2 destinataires" }).click();
      await expect(page).toHaveURL(/\/admin\/communication\/\d+$/);

      const messages = await waitForMessages(piecesSubject, since, 2);
      expect(messages).toHaveLength(2);
      for (const m of messages) {
        expect(m.Attachments.map((a) => [a.FileName, a.Size])).toEqual([
          ["Convocation AG.pdf", pdf.length],
        ]);
      }
    });

    await test.step("téléchargement depuis le détail", async () => {
      const download = page.waitForEvent("download");
      await main.getByRole("link", { name: /Convocation AG\.pdf/ }).click();
      expect((await download).suggestedFilename()).toBe("Convocation AG.pdf");
      await expectNoBlockingA11yViolations(page);
    });
  });

  test("US3 — choix de la cible", async ({ page }) => {
    await login(page, "admin-communication-cibles@exemple.fr");
    const main = page.locator("#contenu");
    await page.goto("/admin/communication/nouvelle");
    const summary = main.getByRole("status").filter({ hasText: /ciblés?/ });

    await main.getByRole("radio", { name: "Tous les adhérents" }).click();
    await expect(summary).toHaveText(/6 adhérents ciblés · 4 destinataires/);
    await main.getByRole("radio", { name: "À jour" }).click();
    await expect(summary).toHaveText(/2 adhérents ciblés · 2 destinataires/);

    await main.getByRole("radio", { name: "Personnalisée" }).click();
    const situations = main.getByRole("group", { name: "Situation de paiement" });
    await expect(situations.getByRole("checkbox", { name: "À jour" })).toBeChecked();
    // Au clavier : Espace coche et décoche.
    await situations.getByRole("checkbox", { name: "À jour" }).focus();
    await page.keyboard.press("Space");
    await situations.getByRole("checkbox", { name: "À payer" }).focus();
    await page.keyboard.press("Space");
    await expect(situations.getByRole("checkbox", { name: "À payer" })).toBeChecked();
    await expect(summary).toHaveText(/3 adhérents ciblés · 1 destinataire/);
    await main
      .getByRole("group", { name: "Tarifs" })
      .getByRole("checkbox", { name: "Adulte" })
      .click();
    await expect(summary).toHaveText(/0 adhérent ciblé · 0 destinataire/);
    await expectNoBlockingA11yViolations(page);

    await main.getByRole("textbox", { name: /^Objet/ }).fill("Cible vide");
    await main.getByRole("textbox", { name: /^Message/ }).fill("Ce message ne partira pas.");
    await main.getByRole("button", { name: "Envoyer", exact: true }).click();
    await expect(main.getByText("Aucun destinataire ne correspond à cette cible.")).toBeVisible();
    await expect(page.getByRole("alertdialog")).toHaveCount(0);
  });

  test("US4 — historique et suivi des envois", async ({ page }) => {
    await login(page, "admin-communication-historique@exemple.fr");
    const main = page.locator("#contenu");
    await page.goto("/admin/communication");
    const links = main.getByRole("table", { name: "Communications envoyées" }).getByRole("link");
    await expect(links.nth(0)).toHaveText(piecesSubject);
    await expect(links.nth(1)).toHaveText(relanceSubject);
    await expect(main.getByText("2 communications")).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    await links.nth(1).click();
    await expect(main.getByRole("heading", { name: relanceSubject })).toBeVisible();
    await expect(async () => {
      await page.reload();
      await expect(main.getByText("2 envoyés")).toBeVisible({ timeout: 1000 });
    }).toPass({ timeout: 20_000 });
    await expect(
      main.getByRole("table", { name: "Destinataires" }).getByText("Envoyé", { exact: true }),
    ).toHaveCount(2);
    await expectNoBlockingA11yViolations(page);

    await test.step("320 px de large", async () => {
      await page.setViewportSize({ width: 320, height: 800 });
      await page.reload();
      await expect(main.getByRole("heading", { name: relanceSubject })).toBeVisible();
      await noHorizontalScroll(page);
      await page.goto("/admin/communication");
      await expect(links.first()).toBeVisible();
      await noHorizontalScroll(page);
      await page.goto("/admin/communication/nouvelle");
      await expect(main.getByRole("textbox", { name: /^Objet/ })).toBeVisible();
      await noHorizontalScroll(page);
    });
  });
});
