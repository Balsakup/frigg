import type { Page } from "@playwright/test";
import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Champs personnalisés (011). Projet « champs » : serveur dédié dont la base
// contient une saison ouverte, les adhésions de 006 et les champs d'exemple
// (e2eseed, mode « champs ») :
// - membres : « Numéro de licence fédérale » (rempli par le membre ; Lucas
//   Moreau : 1234567) et « Remarque du bureau » (réservé) ;
// - adhésions : « A payé sa licence fédérale » (réservé ; cochée pour Hugo
//   Bernard) et « Taille de t-shirt » (S, M, L ; rempli par le membre,
//   obligatoire).
// Les tests tournent en parallèle : chacun crée et supprime ses propres
// champs, jamais obligatoires, pour ne pas gêner les autres.

const SECTION = "/admin/parametres/champs";

function region(page: Page, name: string) {
  return page.locator("#contenu").getByRole("region", { name });
}

async function addField(
  page: Page,
  regionName: string,
  label: string,
  type: string,
  visibility = "Réservé à l'administration",
) {
  await region(page, regionName).getByRole("button", { name: "Ajouter un champ" }).click();
  const dialog = page.getByRole("dialog", { name: "Ajouter un champ" });
  await dialog.getByLabel(/^Libellé/).fill(label);
  await dialog.getByRole("radio", { name: type }).click();
  await dialog.getByRole("radio", { name: visibility }).click();
  return dialog;
}

async function deleteField(page: Page, regionName: string, label: string) {
  await region(page, regionName)
    .getByRole("button", { name: `Plus d'actions pour ${label}` })
    .click();
  await page.getByRole("menuitem", { name: "Supprimer" }).click();
  const confirm = page.getByRole("alertdialog", { name: `Supprimer le champ « ${label} » ?` });
  await confirm.getByRole("button", { name: "Supprimer" }).click();
  await expect(page.getByText(`Champ ${label} supprimé.`).first()).toBeVisible();
}

test.describe("champs personnalisés (011)", () => {
  test("définir les champs", async ({ page, browser }) => {
    await login(page, "admin-champs-definition@exemple.fr");
    await page.goto(SECTION);
    await expect(page).toHaveTitle(/^Champs personnalisés · Paramètres/);
    const members = region(page, "Champs des membres");
    await expect(members.getByText("Numéro de licence fédérale")).toBeVisible();
    await expect(members.getByText("Rempli par le membre").first()).toBeVisible();

    // Ajout d'un champ de membre rempli par le membre.
    let dialog = await addField(
      page,
      "Champs des membres",
      "Club d'origine",
      "Texte court",
      "Rempli par le membre",
    );
    await dialog.getByLabel(/^Texte d'aide/).fill("Club fréquenté avant l'adhésion.");
    await dialog.getByRole("button", { name: "Ajouter", exact: true }).click();
    await expect(page.getByText("Champ ajouté.").first()).toBeVisible();
    await expect(members.getByRole("listitem").filter({ hasText: "Club d'origine" })).toBeVisible();

    // Doublon de libellé, sans casse ni accents.
    dialog = await addField(page, "Champs des membres", "numero de LICENCE federale", "Date");
    await dialog.getByRole("button", { name: "Ajouter", exact: true }).click();
    await expect(dialog.getByText("Un champ porte déjà ce libellé.")).toBeVisible();
    await dialog.getByRole("button", { name: "Annuler" }).click();

    // Liste de choix à un seul choix : refusée.
    dialog = await addField(page, "Champs des adhésions", "Niveau", "Liste de choix");
    await dialog.getByRole("textbox", { name: "Choix 1" }).fill("Débutant");
    await dialog.getByRole("button", { name: "Retirer le choix 2" }).click();
    await dialog.getByRole("button", { name: "Ajouter", exact: true }).click();
    await expect(dialog.getByText("Indiquez de 2 à 50 choix.")).toBeVisible();
    await dialog.getByRole("button", { name: "Annuler" }).click();

    // Oui/non : la case « Obligatoire » disparaît.
    dialog = await addField(page, "Champs des adhésions", "Certificat remis", "Oui/non");
    await expect(dialog.getByRole("checkbox", { name: "Obligatoire" })).toHaveCount(0);
    await dialog.getByRole("button", { name: "Ajouter", exact: true }).click();
    await expect(page.getByText("Champ ajouté.").first()).toBeVisible();

    // Déplacement au clavier, conservé après rechargement. Les autres tests
    // ajoutent des champs en parallèle : on vérifie l'ordre relatif au voisin.
    const labels = members.locator("ol > li p.font-bold");
    await expect(labels.filter({ hasText: "Club d'origine" })).toHaveCount(1);
    const before = await labels.allTextContents();
    const neighbour = before[before.indexOf("Club d'origine") - 1];
    await members.getByRole("button", { name: "Monter Club d'origine" }).focus();
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("status").getByText(/Club d'origine déplacé en position \d+ sur \d+/),
    ).toBeAttached();
    await page.reload();
    await expect(labels.filter({ hasText: "Club d'origine" })).toHaveCount(1);
    const after = await labels.allTextContents();
    expect(after.indexOf("Club d'origine")).toBeLessThan(after.indexOf(neighbour));

    // Archivage puis rétablissement.
    await members.getByRole("button", { name: "Plus d'actions pour Club d'origine" }).click();
    await page.getByRole("menuitem", { name: "Archiver" }).click();
    await expect(
      page.getByText("Champ Club d'origine archivé. Ses valeurs sont conservées.").first(),
    ).toBeVisible();
    await members.getByRole("button", { name: "Champs archivés (1)" }).click();
    await members.getByRole("button", { name: "Rétablir Club d'origine" }).click();
    await expect(page.getByText("Champ Club d'origine rétabli.").first()).toBeVisible();

    await expectNoBlockingA11yViolations(page);

    // Écran étroit : pas de défilement horizontal.
    await page.setViewportSize({ width: 320, height: 800 });
    await expect
      .poll(() =>
        page.evaluate(
          () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
        ),
      )
      .toBe(false);
    await page.setViewportSize({ width: 1280, height: 800 });

    await deleteField(page, "Champs des membres", "Club d'origine");
    await deleteField(page, "Champs des adhésions", "Certificat remis");

    // Membre : section refusée.
    const ctx = await browser.newContext();
    const member = await ctx.newPage();
    await login(member, "membre-champs@exemple.fr");
    await member.goto(SECTION);
    await expect(member.getByRole("heading", { level: 1, name: "Accès refusé" })).toBeVisible();
    await ctx.close();
  });

  test("champs de membre", async ({ page, browser }) => {
    await login(page, "admin-champs-membres@exemple.fr");
    const found = await page.request.get("/api/admin/members?q=Lucas%20Moreau");
    const id = (await found.json()).items[0].id as number;

    // Administration : toutes les valeurs, réservées comprises.
    await page.goto(`/admin/membres/${id}`);
    const extra = page.getByRole("group", { name: "Informations complémentaires" });
    await expect(extra.getByLabel(/^Numéro de licence fédérale/)).toHaveValue("1234567");
    await expect(extra.getByText("Figure sur votre licence, 7 chiffres.")).toBeVisible();
    await extra.getByLabel(/^Remarque du bureau/).fill("Certificat à renouveler en mars.");
    await page.getByRole("button", { name: "Enregistrer" }).click();
    await expect(page.getByText("Modifications enregistrées.").first()).toBeVisible();
    await page.goto(`/admin/membres/${id}`);
    await expect(page.getByLabel(/^Remarque du bureau/)).toHaveValue(
      "Certificat à renouveler en mars.",
    );

    // Espace membre : licence modifiable, remarque du bureau absente.
    const ctx = await browser.newContext();
    const member = await ctx.newPage();
    await login(member, "membre-champs@exemple.fr");
    await member.goto(`/mes-membres/${id}`);
    const mine = member.getByRole("group", { name: "Informations complémentaires" });
    await expect(mine.getByLabel(/^Numéro de licence fédérale/)).toHaveValue("1234567");
    await expect(member.getByText("Remarque du bureau")).toHaveCount(0);
    await expect(member.getByText("Certificat à renouveler en mars.")).toHaveCount(0);
    await expectNoBlockingA11yViolations(member);
    await mine.getByLabel(/^Numéro de licence fédérale/).fill("1234568");
    await member.getByRole("button", { name: "Enregistrer" }).click();
    await expect(member.getByText("Modifications enregistrées.").first()).toBeVisible();
    await member.goto(`/mes-membres/${id}`);
    await expect(member.getByLabel(/^Numéro de licence fédérale/)).toHaveValue("1234568");
    await ctx.close();

    // La valeur réservée n'a pas été effacée par l'enregistrement du membre.
    await page.goto(`/admin/membres/${id}`);
    await expect(page.getByLabel(/^Remarque du bureau/)).toHaveValue(
      "Certificat à renouveler en mars.",
    );
  });

  test("champs d'adhésion", async ({ page, browser }) => {
    // Demande par le membre : taille de t-shirt exigée, licence payée absente.
    const ctx = await browser.newContext();
    const member = await ctx.newPage();
    await login(member, "membre-champs-adhesion@exemple.fr");
    await member.goto("/adhesions/demande");
    const form = member.locator("#contenu");
    await expect(form.getByText("A payé sa licence fédérale")).toHaveCount(0);
    await form.getByRole("radio", { name: "Emma Lambert" }).click();
    await form.getByRole("radio", { name: /Jeune/ }).click();
    await form.getByRole("radio", { name: "Non, première adhésion" }).click();
    await form.getByRole("button", { name: "Envoyer la demande" }).click();
    await expect(form.getByText("Ce champ est obligatoire.")).toBeVisible();
    await form.getByRole("combobox", { name: /Taille de t-shirt/ }).click();
    await member.getByRole("option", { name: "M", exact: true }).click();
    await expectNoBlockingA11yViolations(member);
    await form.getByRole("button", { name: "Envoyer la demande" }).click();
    await expect(member).toHaveURL("/adhesions");
    await expect(form.getByText("Demande envoyée.").first()).toBeVisible();
    await ctx.close();

    // Validation par l'administration avec la licence payée.
    await login(page, "admin-champs-adhesions@exemple.fr");
    const found = await page.request.get("/api/admin/memberships?season=all&q=Emma");
    const id = (await found.json()).items[0].id as number;
    await page.goto(`/admin/adhesions/${id}`);
    const main = page.locator("#contenu");
    await main.getByRole("button", { name: "Valider" }).click();
    const dialog = page.getByRole("dialog", { name: "Valider l'adhésion" });
    await dialog.getByRole("checkbox", { name: "A payé sa licence fédérale" }).click();
    await dialog.getByRole("button", { name: "Valider l'adhésion" }).click();
    await expect(main.getByText("Adhésion validée.").first()).toBeVisible();
    const extra = main.getByRole("region", { name: "Informations complémentaires" });
    await expect(extra.getByRole("checkbox", { name: "A payé sa licence fédérale" })).toBeChecked();

    // Adhésion payée : le champ reste modifiable, sans toucher au paiement.
    const paid = await page.request.post(`/api/admin/memberships/${id}/payment`, {
      data: { method: "cash", receivedOn: new Date().toISOString().slice(0, 10) },
    });
    expect(paid.status()).toBe(200);
    await page.reload();
    await extra.getByRole("checkbox", { name: "A payé sa licence fédérale" }).click();
    await extra.getByRole("button", { name: "Enregistrer les informations" }).click();
    await expect(
      main.getByText("Informations complémentaires enregistrées.").first(),
    ).toBeVisible();
    await page.reload();
    await expect(
      extra.getByRole("checkbox", { name: "A payé sa licence fédérale" }),
    ).not.toBeChecked();
    await expect(main.getByText("Payée").first()).toBeVisible();
    await expectNoBlockingA11yViolations(page);

    // Adhésion annulée : valeurs en lecture seule.
    const cancelled = await page.request.post(`/api/admin/memberships/${id}/cancel`, {
      data: { reason: "Test" },
    });
    expect(cancelled.status()).toBe(200);
    await page.reload();
    await expect(extra.getByRole("checkbox")).toHaveCount(0);
    await expect(extra.getByText("Taille de t-shirt")).toBeVisible();
    await expect(extra.getByText("M", { exact: true })).toBeVisible();
  });

  test("filtres et recherche", async ({ page }) => {
    await login(page, "admin-champs-filtres@exemple.fr");
    const main = page.locator("#contenu");

    // Adhésions de la saison en cours dont la licence n'est pas payée.
    await page.goto("/admin/adhesions");
    await expect(main.getByRole("link", { name: /Bernard Hugo/ })).toBeVisible();
    await main.getByRole("combobox", { name: "A payé sa licence fédérale" }).click();
    await page.getByRole("option", { name: "Non", exact: true }).click();
    await expect(page).toHaveURL(/champ=\d+%3Afalse|champ=\d+:false/);
    await expect(main.getByRole("link", { name: /Durand Marc/ })).toBeVisible();
    await expect(main.getByRole("link", { name: /Bernard Hugo/ })).toHaveCount(0);
    await expect(main.getByRole("status").getByText(/adhésion/)).toBeVisible();
    await page.reload();
    await expect(main.getByRole("link", { name: /Bernard Hugo/ })).toHaveCount(0);
    await expect(main.getByRole("combobox", { name: "A payé sa licence fédérale" })).toHaveText(
      /Non/,
    );
    await expectNoBlockingA11yViolations(page);

    // Recherche d'un membre par un fragment de son numéro de licence.
    await page.goto("/admin/membres");
    await main.getByLabel("Rechercher un membre").fill("3456");
    await expect(main.getByRole("rowheader", { name: /Moreau Lucas/ })).toBeVisible();
    await expect(main.getByRole("status").getByText("1 membre")).toBeVisible();
  });

  test("suppression", async ({ page }) => {
    await login(page, "admin-champs-suppression@exemple.fr");
    await page.goto(SECTION);
    const dialog = await addField(page, "Champs des membres", "Champ à supprimer", "Texte court");
    await dialog.getByRole("button", { name: "Ajouter", exact: true }).click();
    await expect(page.getByText("Champ ajouté.").first()).toBeVisible();

    // Une valeur enregistrée sur une fiche créée pour le test.
    const list = await (await page.request.get("/api/admin/custom-fields")).json();
    const field = list.member.find((f: { label: string }) => f.label === "Champ à supprimer");
    const created = await page.request.post("/api/admin/members", {
      data: {
        firstName: "Sacha",
        lastName: "Suppression",
        birthDate: "2001-01-01",
        accountId: null,
        customFields: [{ fieldId: field.id, text: "valeur" }],
      },
    });
    expect(created.status()).toBe(201);
    await page.reload();

    // Le dialogue annonce la valeur perdue et propose l'archivage.
    const members = region(page, "Champs des membres");
    await members.getByRole("button", { name: "Plus d'actions pour Champ à supprimer" }).click();
    await page.getByRole("menuitem", { name: "Supprimer" }).click();
    const confirm = page.getByRole("alertdialog", {
      name: "Supprimer le champ « Champ à supprimer » ?",
    });
    await expect(
      confirm.getByText("La valeur enregistrée sera définitivement perdue."),
    ).toBeVisible();
    await expectNoBlockingA11yViolations(page);
    await confirm.getByRole("button", { name: "Archiver à la place" }).click();
    await expect(
      page.getByText("Champ Champ à supprimer archivé. Ses valeurs sont conservées.").first(),
    ).toBeVisible();

    // Suppression confirmée depuis les champs archivés.
    await members.getByRole("button", { name: /Champs archivés/ }).click();
    await members.getByRole("button", { name: "Supprimer Champ à supprimer" }).click();
    await page
      .getByRole("alertdialog", { name: "Supprimer le champ « Champ à supprimer » ?" })
      .getByRole("button", { name: "Supprimer", exact: true })
      .click();
    await expect(page.getByText("Champ Champ à supprimer supprimé.").first()).toBeVisible();
    await expect(members.getByText("Champ à supprimer")).toHaveCount(0);
  });
});
