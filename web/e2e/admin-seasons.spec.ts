import { expectNoBlockingA11yViolations } from "./axe";
import { expect, login, test } from "./test";

// Les saisons sont un état global de l'instance : un seul parcours, en série,
// qui part d'une base sans saison (quickstart.md 003 › Jeu de données).
test.describe.configure({ mode: "serial" });

const pad = (n: number) => String(n).padStart(2, "0");
const iso = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const long = (d: Date) => new Intl.DateTimeFormat("fr-FR", { dateStyle: "long" }).format(d);
const label = (s: Date, e: Date) =>
  s.getFullYear() === e.getFullYear()
    ? `${s.getFullYear()}`
    : `${s.getFullYear()}-${e.getFullYear()}`;
const addDays = (d: Date, n: number) => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);

// Saison de référence : du 1er du mois courant au dernier jour du 12e mois, en
// mois entiers et contenant aujourd'hui ; les suivantes sont ses duplications.
const now = new Date();
const [y, m] = [now.getFullYear(), now.getMonth()];
const s1 = new Date(y, m, 1);
const e1 = new Date(y, m + 12, 0);
const s2 = new Date(y, m + 12, 1);
const e2 = new Date(y, m + 24, 0);
const s3 = new Date(y, m + 24, 1);
const e3 = new Date(y, m + 36, 0);
const [l1, l2, l3] = [label(s1, e1), label(s2, e2), label(s3, e3)];

test.describe("saisons (003)", () => {
  test("mise en place, ouverture, duplication, modification et suppression", async ({ page }) => {
    await login(page, "admin-saisons@exemple.fr");
    const main = page.locator("#contenu");
    const heading = (name: string) => page.getByRole("heading", { level: 1, name });
    const dialog = page.getByRole("alertdialog");

    await test.step("US5 — tableau de bord sans saison", async () => {
      await page.goto("/admin");
      await expect(
        main.getByText("Créez la première saison pour mettre en place Frigg."),
      ).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      await main.getByRole("link", { name: "Créer la saison" }).click();
      await expect(page).toHaveURL("/admin/saisons/nouvelle");
    });

    await test.step("US1 — navigation et état vide", async () => {
      const settings = page.getByRole("navigation", { name: "Configuration" });
      await expect(settings.getByRole("link")).toHaveText(["Saisons", "Comptes", "Aide"]);
      await page.goto("/admin/saisons");
      await expect(heading("Saisons")).toBeVisible();
      await expect(
        main.getByRole("heading", { name: "Aucune saison pour le moment" }),
      ).toBeVisible();
      await expect(main.getByRole("link", { name: "Nouvelle saison" })).toHaveCount(0);
      await expectNoBlockingA11yViolations(page);
    });

    await test.step("US1 — création de la première saison", async () => {
      await main.getByRole("link", { name: "Créer la première saison" }).click();
      await expect(heading("Nouvelle saison")).toBeVisible();
      await page.getByLabel(/^Date de début/).fill(iso(addDays(now, -40)));
      await page.getByLabel(/^Date de fin/).fill(iso(addDays(now, -1)));
      await page.getByRole("button", { name: "Créer la saison" }).click();
      await expect(main.getByText("La date de fin ne peut pas être passée.")).toBeVisible();
      await expectNoBlockingA11yViolations(page);

      await page.getByLabel(/^Date de début/).fill(iso(s1));
      await page.getByLabel(/^Date de fin/).fill(iso(e1));
      await page.getByRole("button", { name: "Créer la saison" }).click();
      await expect(heading(`Saison ${l1}`)).toBeVisible();
      await expect(main.getByText(`Saison ${l1} créée.`).first()).toBeVisible();
      const infos = main.locator("dl");
      await expect(infos.getByText("Brouillon")).toBeVisible();
      await expect(infos.getByText("Saison courante")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });
    const firstURL = page.url();

    await test.step("US5 — saison courante en brouillon", async () => {
      await page.goto("/admin");
      await expect(
        main.getByText(`La saison ${l1} a commencé mais n'est pas encore ouverte.`),
      ).toBeVisible();
      await main.getByRole("link", { name: "Voir la saison" }).click();
      await expect(page).toHaveURL(firstURL);
    });

    await test.step("US1 — liste", async () => {
      await page.goto("/admin/saisons");
      const table = main.getByRole("table", { name: /Saisons de l'association/ });
      await expect(table.getByRole("row")).toHaveCount(2);
      await expect(table.getByText("Saison courante")).toBeVisible();
      await expectNoBlockingA11yViolations(page);
    });

    let secondURL = "";
    await test.step("US2 — ouverture et duplication", async () => {
      await page.goto(firstURL);
      await main.getByRole("button", { name: "Ouvrir la saison" }).click();
      await dialog.getByRole("button", { name: "Ouvrir la saison" }).click();
      await expect(main.getByText(`Saison ${l1} ouverte.`).first()).toBeVisible();
      await expect(main.locator("dl").getByText("Ouverte", { exact: true })).toBeVisible();
      await expect(main.getByRole("button", { name: "Supprimer" })).toHaveCount(0);

      await page.goto("/admin");
      await expect(heading("Tableau de bord")).toBeVisible();
      await expect(main.getByRole("link", { name: /Créer la saison|Voir la saison/ })).toHaveCount(
        0,
      );
      await expect(
        main.getByText(/n'est pas encore ouverte|Aucune saison n'est en cours/),
      ).toHaveCount(0);

      await page.goto(firstURL);
      await main.getByRole("button", { name: "Dupliquer" }).click();
      await expect(heading(`Saison ${l2}`)).toBeVisible();
      await expect(main.getByText(`Saison ${l2} créée en brouillon.`).first()).toBeVisible();
      await expect(main.locator("dl")).toContainText(long(s2));
      await expect(main.locator("dl")).toContainText(long(e2));
      secondURL = page.url();

      await main.getByRole("button", { name: "Dupliquer" }).click();
      await expect(heading(`Saison ${l3}`)).toBeVisible();
      await expect(main.getByText(`Ouvrez d'abord la saison ${l2}.`)).toBeVisible();
      await expect(main.getByRole("button", { name: "Ouvrir la saison" })).toHaveCount(0);
      await expectNoBlockingA11yViolations(page);

      await page.goto(secondURL);
      await main.getByRole("button", { name: "Ouvrir la saison" }).click();
      await expectNoBlockingA11yViolations(page);
      await dialog.getByRole("button", { name: "Ouvrir la saison" }).click();
      await expect(main.getByText(`Saison ${l2} ouverte.`).first()).toBeVisible();

      await page.goto("/admin/saisons");
      const table = main.getByRole("table", { name: /Saisons de l'association/ });
      await expect(table.getByText("Ouverte", { exact: true })).toHaveCount(2);
      await expect(table.getByText("Saison courante")).toHaveCount(1);

      await main.getByRole("link", { name: "Nouvelle saison" }).click();
      const start = page.getByLabel(/^Date de début/);
      await expect(start).toHaveAttribute("readonly", "");
      await expect(start).toHaveValue(iso(addDays(e3, 1)));
    });

    await test.step("US3 — modification des dates et suppression", async () => {
      await page.goto(firstURL);
      await expect(heading(`Saison ${l1}`)).toBeVisible();
      await expect(main.getByRole("button", { name: "Modifier les dates" })).toHaveCount(0);

      await page.goto(secondURL);
      await main.getByRole("button", { name: "Modifier les dates" }).click();
      const edit = page.getByRole("dialog");
      await expect(edit.getByLabel(/^Date de début/)).toHaveCount(0);
      const end = edit.getByLabel(/^Date de fin/);
      await expect(end).toHaveAttribute("min", iso(addDays(s2, 1)));
      await expect(edit.getByText(`La saison ${l3} commencera le lendemain.`)).toBeVisible();
      await expectNoBlockingA11yViolations(page);
      const newEnd = new Date(y, m + 25, 0);
      await end.fill(iso(newEnd));
      await edit.getByRole("button", { name: "Enregistrer" }).click();
      await expect(main.getByText(`Dates de la saison ${l2} enregistrées.`).first()).toBeVisible();
      await expect(main.locator("dl")).toContainText(long(newEnd));

      await main.getByRole("link", { name: "Saisons" }).first().click();
      await main.getByRole("link", { name: `Saison ${label(addDays(newEnd, 1), e3)}` }).click();
      await expect(main.locator("dl")).toContainText(long(addDays(newEnd, 1)));
      await main.getByRole("button", { name: "Supprimer" }).click();
      await dialog.getByRole("button", { name: "Supprimer" }).click();
      await expect(page).toHaveURL("/admin/saisons");
      await expect(main.getByText(/^Saison .+ supprimée\.$/).first()).toBeVisible();
      await expect(main.getByRole("table").getByRole("row")).toHaveCount(3);

      await page.goto(secondURL);
      await expect(heading(`Saison ${l2}`)).toBeVisible();
      await expect(main.getByRole("button", { name: "Supprimer" })).toHaveCount(0);
    });

    await test.step("320 px : aucun défilement horizontal de la page", async () => {
      await page.setViewportSize({ width: 320, height: 800 });
      for (const path of ["/admin/saisons", firstURL, "/admin/saisons/nouvelle", "/admin"]) {
        await page.goto(path);
        await expect(page.locator("#contenu h1")).toBeVisible();
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
          ),
          path,
        ).toBeLessThanOrEqual(0);
        await expectNoBlockingA11yViolations(page);
      }
    });
  });
});
