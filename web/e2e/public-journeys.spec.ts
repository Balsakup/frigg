import { expectNoBlockingA11yViolations } from "./axe";
import { waitForMail } from "./mailpit";
import { expect, login, test } from "./test";

function localInput(daysFromNow: number, hour: number) {
  const d = new Date();
  d.setDate(d.getDate() + daysFromNow);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(hour)}:00`;
}

// Parcours publics (016). Serveur dédié (jeu des adhésions : saison ouverte,
// tarifs « Adulte » 60 € / renouvellement 40 €, « Jeune », « Bienfaiteur »),
// car l'accueil des demandes publiques est un paramètre global.

test.describe("parcours publics (016)", () => {
  test("Adhésion publique", async ({ page, browser }) => {
    // Adresse propre à chaque lancement : Mailpit est partagé.
    const email = `claire.publique.${Date.now()}@exemple.fr`;
    const admin = await (
      await browser.newContext({
        locale: "fr-FR",
        permissions: ["clipboard-read", "clipboard-write"],
        extraHTTPHeaders: { "X-Forwarded-For": "203.0.113.20, 198.51.100.120" },
      })
    ).newPage();
    const main = page.locator("#contenu");

    await test.step("l'accueil est fermé par défaut", async () => {
      await page.goto("/adherer");
      await expect(
        page.getByRole("heading", { level: 1, name: "Demande d'adhésion" }),
      ).toBeVisible();
      await expect(
        main.getByText("Les demandes d'adhésion ne sont pas ouvertes pour le moment."),
      ).toBeVisible();
      await expect(main.getByRole("button", { name: "Envoyer ma demande" })).toHaveCount(0);
    });

    await test.step("l'administrateur active l'accueil et copie l'adresse", async () => {
      await login(admin, "admin@exemple.fr");
      await admin.goto("/admin/parametres/general");
      await admin
        .getByRole("switch", { name: "Accepter les demandes d'adhésion publiques" })
        .click();
      const address = admin.getByLabel("Adresse de la page d'adhésion");
      await expect(address).toHaveValue(/\/adherer$/);
      await admin.getByRole("button", { name: "Enregistrer" }).click();
      await expect(admin.getByText("Paramètres enregistrés.").first()).toBeVisible();
      await admin.getByRole("button", { name: /Copier/ }).click();
      await expect(admin.getByRole("status").filter({ hasText: "Copié." })).toHaveCount(1);
      expect(await admin.evaluate(() => navigator.clipboard.readText())).toMatch(/\/adherer$/);
      await expectNoBlockingA11yViolations(admin);
    });

    await test.step("dépôt anonyme puis écran neutre", async () => {
      await page.goto("/adherer");
      await expect(main.getByRole("button", { name: "Envoyer ma demande" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);

      // Au clavier, le piège n'est jamais atteint.
      await main.getByLabel(/^Prénom/).focus();
      for (let i = 0; i < 40; i++) {
        await page.keyboard.press("Tab");
        expect(await page.evaluate(() => document.activeElement?.getAttribute("name"))).not.toBe(
          "website",
        );
      }

      await main.getByLabel(/^Prénom/).fill("Claire");
      await main.getByLabel(/^Nom/).fill("Publique");
      await main.getByLabel(/^Date de naissance/).fill("1988-06-14");
      await main.getByLabel(/^Adresse électronique/).fill(email);
      await main.getByRole("radio", { name: /Adulte/ }).click();
      await main.getByRole("radio", { name: "Renouvellement", exact: true }).click();
      await expect(main.getByText("Montant : 40,00 €")).toBeVisible();
      await main.getByRole("radio", { name: "Première adhésion", exact: true }).click();
      await expect(main.getByText("Montant : 60,00 €")).toBeVisible();

      // Sans consentement, le serveur refuse et l'erreur est reliée à la case.
      await main.getByRole("button", { name: "Envoyer ma demande" }).click();
      await expect(
        main.getByText("Vous devez accepter le traitement de vos données."),
      ).toBeVisible();
      await expectNoBlockingA11yViolations(page);

      await main.getByRole("checkbox", { name: /J'accepte que Les Sentiers du Val/ }).click();
      const since = new Date();
      await main.getByRole("button", { name: "Envoyer ma demande" }).click();
      await expect(
        page.getByRole("heading", { level: 1, name: "Consultez votre messagerie" }),
      ).toBeVisible();
      await expect(main.getByText(email)).toBeVisible();
      await expect(main.getByRole("link", { name: "Je n'ai rien reçu" })).toHaveAttribute(
        "href",
        "/suivi/lien",
      );

      const text = await waitForMail(email, since, "Confirmez votre demande");
      const link = text.match(/https?:\/\/\S+\/confirmer-demande#jeton=[A-Za-z0-9_-]+/)?.[0];
      expect(link).toBeTruthy();

      await test.step("confirmation par le lien de l'e-mail", async () => {
        const linkSince = new Date();
        await page.goto(link as string);
        await expect(
          page.getByRole("heading", { level: 1, name: "Demande enregistrée" }),
        ).toBeVisible();
        await expect(
          main.getByText("Un e-mail vous a été envoyé avec votre lien de suivi.", { exact: false }),
        ).toBeVisible();
        // Le jeton est effacé de la barre d'adresse.
        expect(new URL(page.url()).hash).toBe("");
        await expectNoBlockingA11yViolations(page);
        expect(await waitForMail(email, linkSince, "Votre lien de suivi")).toContain(
          "/suivi#jeton=",
        );

        // Un lien déjà utilisé n'est plus valable (nouvel onglet, comme un
        // second clic dans l'e-mail).
        const again = await page.context().newPage();
        await again.goto(link as string);
        await expect(
          again.getByRole("heading", { level: 1, name: "Ce lien n'est plus valable" }),
        ).toBeVisible();
        await expectNoBlockingA11yViolations(again);
        await again.close();
      });
    });

    const validatedSince = new Date();
    await test.step("l'administrateur voit la demande publique et la valide", async () => {
      await admin.goto("/admin/adhesions?season=all&status=pending&origin=public");
      const row = admin.getByRole("row").filter({ hasText: "Publique Claire" });
      await expect(row.getByText("Publique", { exact: true })).toBeVisible();
      await row.getByRole("link", { name: "Publique Claire" }).click();
      const detail = admin.locator("#contenu");
      await expect(detail.getByText("Demande publique", { exact: false })).toBeVisible();
      await detail.getByRole("button", { name: "Valider" }).click();
      const dialog = admin.getByRole("dialog");
      await dialog.getByRole("button", { name: "Valider l'adhésion" }).click();
      await expect(detail.getByText("Adhésion validée.").first()).toBeVisible();
    });

    // Suivi et paiement (US2).
    await test.step("le lien de l'e-mail de validation mène à la page de paiement", async () => {
      const text = await waitForMail(email, validatedSince, "Claire Publique validée");
      const link = text.match(/https?:\/\/\S+\/suivi\/adhesions\/\d+#jeton=[A-Za-z0-9_-]+/)?.[0];
      expect(link).toBeTruthy();
      await page.goto(link as string);
      await expect(page.getByRole("heading", { level: 2, name: "Paiement" })).toBeVisible();
      // Le jeton est effacé de la barre d'adresse.
      expect(page.url()).not.toContain("jeton");
      await expect(
        main.getByText("Pour corriger vos informations, contactez l'association."),
      ).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("paiement par carte sans compte", async () => {
      await main.getByRole("radio", { name: "Carte bancaire" }).click();
      await expect(main.getByText("Mode test — aucun débit réel.")).toBeVisible();
      await main.getByRole("button", { name: "Payer 60,00 € par carte" }).click();
      await expect(page.getByRole("heading", { name: "HelloAsso (test)" })).toBeVisible();
      await page.getByRole("button", { name: "Payer", exact: true }).click();
      await expect(main.getByText(/^Payée le .* \(carte bancaire\)/)).toBeVisible();
      await expect(page.getByRole("heading", { level: 2, name: "Paiement" })).toHaveCount(0);

      await page.goto("/suivi");
      await expect(page.getByRole("heading", { level: 1, name: "Mon suivi" })).toBeVisible();
      const card = main.getByRole("region", { name: "Claire Publique" });
      await expect(card.getByText(/^Payée/).first()).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("« Recevoir mon lien de suivi » répond toujours de même", async () => {
      for (const address of [email, `inconnue.${Date.now()}@exemple.fr`]) {
        await page.goto("/suivi/lien");
        await main.getByLabel(/^Adresse électronique/).fill(address);
        await main.getByRole("button", { name: "Recevoir mon lien" }).click();
        await expect(main.getByRole("status")).toContainText(
          "Si l'adresse est valide, un e-mail vient de vous être envoyé.",
        );
      }
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("fermer l'accès sur cet appareil", async () => {
      await page.goto("/suivi");
      await main.getByRole("button", { name: "Fermer l'accès sur cet appareil" }).click();
      await expect(page.getByRole("heading", { level: 1, name: "Accès fermé" })).toBeVisible();
      await page.goto("/suivi");
      await expect(
        page.getByRole("heading", { level: 1, name: "Ce lien n'est plus valable" }),
      ).toBeVisible();
      await expect(main.getByRole("link", { name: "Recevoir un nouveau lien" })).toHaveAttribute(
        "href",
        "/suivi/lien",
      );
    });
  });

  test("Suivi et paiement d'une fiche sans compte", async ({ page, browser }) => {
    const email = `nadia.importee.${Date.now()}@exemple.fr`;
    const main = page.locator("#contenu");
    await login(page, "admin@exemple.fr");

    await test.step("l'administrateur crée une fiche sans compte avec une adresse", async () => {
      await page.goto("/admin/membres/nouveau");
      await expect(
        main.getByText(
          "Cette adresse recevra les notifications et un accès au suivi et au paiement.",
        ),
      ).toBeVisible();
      await main.getByLabel(/^Prénom/).fill("Nadia");
      await main.getByLabel(/^Nom/).fill("Importée");
      await main.getByLabel(/^Date de naissance/).fill("1979-09-21");
      await main.getByLabel(/^Adresse électronique/).fill(email);
      await main.getByRole("button", { name: "Ajouter le membre" }).click();
      await expect(main.getByText("Membre ajouté.").first()).toBeVisible();
      await page.goto("/admin/membres?q=Importée");
      await main.getByRole("link", { name: "Importée Nadia" }).click();
      await main.getByRole("link", { name: "Créer une adhésion" }).click();
      await main.getByRole("radio", { name: /Jeune/ }).click();
      await main.getByRole("radio", { name: "Première adhésion" }).click();
      await main.getByRole("button", { name: "Enregistrer l'adhésion" }).click();
      await expect(main.getByText("Adhésion enregistrée.").first()).toBeVisible();
    });

    const since = new Date();
    await test.step("« Envoyer un lien de suivi »", async () => {
      await page.goto("/admin/membres?q=Importée");
      await main.getByRole("link", { name: "Importée Nadia" }).click();
      await expect(main.getByText("Interne")).toBeVisible();
      await main.getByRole("button", { name: "Envoyer un lien de suivi" }).click();
      await expect(main.getByText(`Lien de suivi envoyé à ${email}.`).first()).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("annonce d'un chèque par le lien", async () => {
      const text = await waitForMail(email, since, "Votre lien de suivi");
      const link = text.match(/https?:\/\/\S+\/suivi#jeton=[A-Za-z0-9_-]+/)?.[0];
      expect(link).toBeTruthy();
      const visitor = await (await browser.newContext({ locale: "fr-FR" })).newPage();
      const content = visitor.locator("#contenu");
      await visitor.goto(link as string);
      const card = content.getByRole("region", { name: "Nadia Importée" });
      await expect(card).toBeVisible();
      expect(visitor.url()).not.toContain("jeton");
      await card.getByRole("link", { name: /^Payer/ }).click();
      await expect(visitor.getByRole("heading", { level: 2, name: "Paiement" })).toBeVisible();
      await content.getByRole("radio", { name: "Chèque" }).click();
      await content.getByRole("button", { name: "Confirmer ce moyen de paiement" }).click();
      await expect(content.getByText("Moyen de paiement enregistré.").first()).toBeVisible();
      await expect(content.getByText("Paiement annoncé — chèque").first()).toBeVisible();
      await expectNoBlockingA11yViolations(visitor);
    });
  });
  test("Événement public", async ({ page, browser }) => {
    const stamp = Date.now();
    const title = `Stage découverte ${stamp}`;
    const email = `claire.evenement.${stamp}@exemple.fr`;
    const second = `bruno.evenement.${stamp}@exemple.fr`;
    const main = page.locator("#contenu");
    const admin = await (
      await browser.newContext({
        locale: "fr-FR",
        permissions: ["clipboard-read", "clipboard-write"],
        extraHTTPHeaders: { "X-Forwarded-For": "203.0.113.21, 198.51.100.121" },
      })
    ).newPage();
    const adminMain = admin.locator("#contenu");
    let publicLink = "";

    await test.step("l'administrateur crée un événement public et copie le lien", async () => {
      await login(admin, "admin@exemple.fr");
      await admin.goto("/admin/evenements/nouveau");
      await adminMain.getByLabel(/^Titre/).fill(title);
      await adminMain.getByLabel(/^Lieu/).fill("Base nautique");
      await adminMain.getByLabel(/^Début/).fill(localInput(20, 10));
      await adminMain.getByLabel(/^Fin/).fill(localInput(20, 16));
      await adminMain.getByRole("radio", { name: /Ouvert au public/ }).click();
      await expect(
        adminMain.getByText("Toute personne qui reçoit le lien peut s'inscrire", { exact: false }),
      ).toBeVisible();
      await adminMain.getByLabel(/^Capacité/).fill("2");
      await adminMain.getByLabel(/^Prix/).fill("5");
      await adminMain.getByRole("button", { name: "Enregistrer le brouillon" }).click();
      await admin.getByRole("button", { name: "Publier" }).click();
      await expect(
        adminMain.getByText("Événement publié : les membres le voient.").first(),
      ).toBeVisible();
      const section = adminMain.getByRole("region", { name: "Lien public" });
      await expect(section.getByLabel("Adresse de la page publique")).toHaveValue(
        /\/e\/[A-Za-z0-9_-]{22}$/,
      );
      await section.getByRole("button", { name: /Copier/ }).click();
      publicLink = await admin.evaluate(() => navigator.clipboard.readText());
      expect(publicLink).toMatch(/\/e\/[A-Za-z0-9_-]{22}$/);
      await expectNoBlockingA11yViolations(admin);
    });

    await test.step("inscription anonyme et confirmation par l'e-mail", async () => {
      await page.goto(publicLink);
      await expect(page.getByRole("heading", { level: 1, name: title })).toBeVisible();
      await expect(main.getByText("2 places restantes")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await main.getByLabel(/^Prénom/).fill("Claire");
      await main.getByLabel(/^Nom/).fill("Évènement");
      await main.getByLabel(/^Date de naissance/).fill("1990-02-11");
      await main.getByLabel(/^Adresse électronique/).fill(email);
      await main.getByRole("checkbox", { name: /J'accepte que/ }).click();
      const since = new Date();
      await main.getByRole("button", { name: "M'inscrire — 5 €" }).click();
      await expect(
        page.getByRole("heading", { level: 1, name: "Consultez votre messagerie" }),
      ).toBeVisible();

      const text = await waitForMail(email, since, "Confirmez votre demande");
      const link = text.match(/https?:\/\/\S+\/confirmer-demande#jeton=[A-Za-z0-9_-]+/)?.[0];
      expect(link).toBeTruthy();
      const confirmedSince = new Date();
      await page.goto(link as string);
      await expect(
        page.getByRole("heading", { level: 1, name: "Inscription confirmée" }),
      ).toBeVisible();

      const confirmation = await waitForMail(
        email,
        confirmedSince,
        `Inscription confirmée : ${title}`,
      );
      expect(confirmation).toContain("EVT-");
      const tracking = confirmation.match(
        /https?:\/\/\S+\/suivi\/inscriptions\/\d+#jeton=[A-Za-z0-9_-]+/,
      )?.[0];
      expect(tracking).toBeTruthy();
      await page.goto(tracking as string);
      await expect(page.getByRole("heading", { level: 2, name: "Paiement" })).toBeVisible();
      expect(page.url()).not.toContain("jeton");
      await expect(main.getByText(title).first()).toBeVisible();
      await expectNoBlockingA11yViolations(page);

      await page.goto(publicLink);
      await expect(main.getByText("1 place restante")).toBeVisible();
    });

    let secondLink = "";
    await test.step("une seconde personne dépose une inscription", async () => {
      const visitor = await (await browser.newContext({ locale: "fr-FR" })).newPage();
      const content = visitor.locator("#contenu");
      await visitor.goto(publicLink);
      await content.getByLabel(/^Prénom/).fill("Bruno");
      await content.getByLabel(/^Nom/).fill("Évènement");
      await content.getByLabel(/^Date de naissance/).fill("1992-04-03");
      await content.getByLabel(/^Adresse électronique/).fill(second);
      await content.getByRole("checkbox", { name: /J'accepte que/ }).click();
      const since = new Date();
      await content.getByRole("button", { name: "M'inscrire — 5 €" }).click();
      await expect(
        visitor.getByRole("heading", { level: 1, name: "Consultez votre messagerie" }),
      ).toBeVisible();
      const text = await waitForMail(second, since, "Confirmez votre demande");
      secondLink = text.match(/https?:\/\/\S+\/confirmer-demande#jeton=[A-Za-z0-9_-]+/)?.[0] ?? "";
      expect(secondLink).toBeTruthy();
      await visitor.context().close();
    });

    await test.step("l'administrateur prend la dernière place ; la confirmation suivante est refusée", async () => {
      await admin.reload();
      await adminMain.getByRole("button", { name: "Inscrire un membre" }).click();
      const dialog = admin.getByRole("dialog", { name: "Inscrire un membre" });
      await dialog.getByLabel(/^Membre/).fill("Bernard");
      await admin.getByRole("option", { name: /Bernard Hugo/ }).click();
      await dialog.getByRole("button", { name: "Inscrire" }).click();
      await expect(adminMain.getByText("Bernard Hugo est inscrit(e).").first()).toBeVisible();
      const row = adminMain.getByRole("row", { name: /Évènement Claire/ });
      await expect(row).toContainText("Inscription publique");

      const visitor = await (await browser.newContext({ locale: "fr-FR" })).newPage();
      await visitor.goto(secondLink);
      await expect(
        visitor.getByRole("heading", { level: 1, name: "Demande impossible" }),
      ).toBeVisible();
      await expect(visitor.getByText("Cet événement est complet.")).toBeVisible();
      await visitor.context().close();

      await page.goto(publicLink);
      await expect(
        main.getByText("Cet événement est complet : il n'y a plus de place."),
      ).toBeVisible();
      await expect(main.getByRole("button", { name: /M'inscrire/ })).toHaveCount(0);
    });

    await test.step("après régénération, l'ancien lien est introuvable", async () => {
      const section = adminMain.getByRole("region", { name: "Lien public" });
      await section.getByRole("button", { name: "Régénérer" }).click();
      await admin
        .getByRole("alertdialog")
        .getByRole("button", { name: "Régénérer le lien" })
        .click();
      await expect(
        adminMain.getByText("Lien public régénéré : l'ancien lien ne fonctionne plus.").first(),
      ).toBeVisible();
      await expect(section.getByLabel("Adresse de la page publique")).not.toHaveValue(publicLink);
      await page.goto(publicLink);
      await expect(
        page.getByRole("heading", { level: 1, name: "Événement introuvable" }),
      ).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });
  });

  // US4 : l'événement public du jeu « publics » (gratuit) sert de dépôt, car
  // l'accueil des demandes d'adhésion est un paramètre global que le test
  // « Adhésion publique » bascule en parallèle.
  test("Création de compte", async ({ page }) => {
    const stamp = Date.now();
    const email = `camille.compte.${stamp}@exemple.fr`;
    const main = page.locator("#contenu");
    let oldLink = "";

    await test.step("dépôt avec « Créer un compte » puis confirmation", async () => {
      await page.goto("/e/SortieDecouverteE2E000");
      await expect(
        page.getByRole("heading", { level: 1, name: "Sortie découverte" }),
      ).toBeVisible();
      await main.getByLabel(/^Prénom/).fill("Camille");
      await main.getByLabel(/^Nom/).fill(`Compte${stamp}`);
      await main.getByLabel(/^Date de naissance/).fill("1991-03-09");
      await main.getByLabel(/^Adresse électronique/).fill(email);
      await main.getByRole("checkbox", { name: /J'accepte que/ }).click();
      await main
        .getByRole("checkbox", { name: "Créer un compte pour suivre mes inscriptions" })
        .click();
      const since = new Date();
      await main.getByRole("button", { name: "M'inscrire" }).click();
      await expect(
        page.getByRole("heading", { level: 1, name: "Consultez votre messagerie" }),
      ).toBeVisible();
      const text = await waitForMail(email, since, "Confirmez votre demande");
      const link = text.match(/https?:\/\/\S+\/confirmer-demande#jeton=[A-Za-z0-9_-]+/)?.[0];
      expect(link).toBeTruthy();

      const confirmedSince = new Date();
      await page.goto(link as string);
      // La case cochée mène au choix du mot de passe.
      await expect(page).toHaveURL(/\/suivi\/compte$/);
      await expect(page.getByRole("heading", { level: 1, name: "Créer mon compte" })).toBeVisible();
      await expect(main.getByLabel("Adresse électronique")).toHaveValue(email);
      // Une seule fiche : la question de la fiche « Vous » n'est pas posée.
      await expect(main.getByText("Laquelle de ces fiches est la vôtre ?")).toHaveCount(0);
      await expectNoBlockingA11yViolations(page);

      const confirmation = await waitForMail(
        email,
        confirmedSince,
        "Inscription confirmée : Sortie découverte",
      );
      oldLink = confirmation.match(/https?:\/\/\S+\/suivi\S*#jeton=[A-Za-z0-9_-]+/)?.[0] ?? "";
      expect(oldLink).toBeTruthy();
    });

    await test.step("l'espace de suivi propose aussi la création", async () => {
      await page.goto("/suivi");
      const box = main.getByRole("region", { name: "Créer un compte" });
      await expect(box).toBeVisible();
      await box.getByRole("link", { name: "Créer mon compte" }).click();
      await expect(page).toHaveURL(/\/suivi\/compte$/);
    });

    await test.step("mot de passe puis connexion", async () => {
      await main.getByLabel(/^Mot de passe/).fill("court");
      await main.getByLabel(/^Confirmez le mot de passe/).fill("court");
      await main.getByRole("button", { name: "Créer mon compte" }).click();
      await expect(main.getByText(/au moins 8 caractères/i).first()).toBeVisible();
      await expectNoBlockingA11yViolations(page);

      await main.getByLabel(/^Mot de passe/).fill("Sentiers2026");
      await main.getByLabel(/^Confirmez le mot de passe/).fill("Sentiers2026");
      await main.getByRole("button", { name: "Créer mon compte" }).click();
      await expect(page).toHaveURL(/\/$/);
      await expect(page.getByRole("button", { name: /menu du compte/ }).first()).toBeVisible();

      await page.goto("/mes-membres");
      const row = page.locator("#contenu").getByText(`Camille Compte${stamp}`, { exact: false });
      await expect(row.first()).toBeVisible();
      await expect(page.locator("#contenu").getByText("Vous", { exact: true })).toBeVisible();
      await page.goto("/mes-inscriptions");
      await expect(page.locator("#contenu").getByText("Sortie découverte").first()).toBeVisible();
    });

    await test.step("un ancien lien n'est plus valable", async () => {
      const visitor = await (
        await page.context().browser()?.newContext({ locale: "fr-FR" })
      )?.newPage();
      if (!visitor) throw new Error("navigateur indisponible");
      await visitor.goto(oldLink);
      await expect(
        visitor.getByRole("heading", { level: 1, name: "Ce lien n'est plus valable" }),
      ).toBeVisible();
      await visitor.context().close();
    });
  });

  // US5 : origine « Publique » dans les listes, fiches publiques inactives.
  // Le jeu « publics » contient une fiche publique inactive (Odile Ancienne),
  // une fiche importée (Paul Lefèvre) et des adhésions et inscriptions
  // internes (Nina Leroy, Alix Perrin).
  test("Supervision", async ({ page }) => {
    const main = page.locator("#contenu");
    await login(page, "admin-carte-parametres@exemple.fr");

    await test.step("filtre d'origine des membres, reflété dans l'adresse", async () => {
      await page.goto("/admin/membres");
      await expect(main.getByRole("link", { name: "Lefèvre Paul" })).toBeVisible();
      await main.getByRole("search").getByRole("combobox", { name: "Origine" }).click();
      await page.getByRole("option", { name: "Publique", exact: true }).click();
      await expect(page).toHaveURL(/origine=publique/);
      const row = main.getByRole("row").filter({ hasText: "Ancienne Odile" });
      await expect(row.getByText("Publique", { exact: true })).toBeVisible();
      await expect(main.getByRole("link", { name: "Lefèvre Paul" })).toHaveCount(0);
      await page.reload();
      await expect(main.getByRole("link", { name: "Ancienne Odile" })).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("filtre d'origine des adhésions et des inscriptions", async () => {
      await page.goto("/admin/adhesions?season=all");
      await expect(main.getByRole("link", { name: "Leroy Nina" })).toBeVisible();
      await page.goto("/admin/adhesions?season=all&origin=public");
      await expect(main.getByRole("combobox", { name: "Origine" })).toHaveText(/Publique/);
      await expect(main.getByRole("link", { name: "Leroy Nina" })).toHaveCount(0);

      await page.goto("/admin/inscriptions");
      await expect(main.getByText("Perrin Alix", { exact: false }).first()).toBeVisible();
      await page.goto("/admin/inscriptions?origine=publique");
      await expect(main.getByRole("combobox", { name: "Origine" })).toHaveText(/Publique/);
      await expect(main.getByText("Perrin Alix", { exact: false })).toHaveCount(0);
    });

    await test.step("encart du tableau de bord puis suppression en lot", async () => {
      await page.goto("/admin");
      const alert = main.getByText("1 fiche publique inactive", { exact: false });
      await expect(alert).toBeVisible();
      await main.getByRole("link", { name: "Voir les fiches" }).click();
      await expect(page).toHaveURL(/\/admin\/membres\?origine=inactives/);
      const row = main.getByRole("row").filter({ hasText: "Ancienne Odile" });
      await expect(row.getByText("Inactive", { exact: true })).toBeVisible();
      await expect(main.getByRole("link", { name: "Lefèvre Paul" })).toHaveCount(0);
      await expectNoBlockingA11yViolations(page);

      await main.getByRole("button", { name: "Supprimer la fiche" }).click();
      const dialog = page.getByRole("alertdialog");
      await expect(dialog).toContainText("La fiche publique inactive");
      await dialog.getByRole("button", { name: "Supprimer la fiche" }).click();
      await expect(main.getByText("1 fiche supprimée.").first()).toBeVisible();
      await expect(main.getByRole("link", { name: "Ancienne Odile" })).toHaveCount(0);
      await page.goto("/admin");
      await expect(main.getByText(/fiches? publiques? inactives?/)).toHaveCount(0);
    });
  });
});
