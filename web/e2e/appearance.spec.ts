import { expectNoBlockingA11yViolations } from "./axe";
import { waitForFullMail } from "./mailpit";
import { expect, login, test } from "./test";

// Paramètres › Apparence (010, US3). Projet « apparence » : instance dédiée,
// après les paramètres généraux, car le logo et les couleurs sont globaux.

const PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAGAAAAAwCAIAAABhdOiYAAAAYUlEQVR42u3QQQ0AIAwAsYGahQ/WkYEcJMzGSHoKLh3vZnRqn9XqZ4YAAQIECBAgQIAECBAgQIAAAQIkQIAAAQIECBAgQAgAAQIECBAgQIAECBAgQIAAAQIkQIAAAQL0RQWAlANJs0W6JAAAAABJRU5ErkJggg==",
  "base64",
);
const SVG = Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>');

test.describe.configure({ mode: "serial" });

test("logo, couleurs, écran de connexion et e-mails (US3)", async ({ page, browser }) => {
  await login(page, "admin@exemple.fr");
  await page.goto("/admin/parametres/apparence");
  const preview = page.getByRole("region", { name: "Aperçu" });
  const logoInput = page.getByLabel("Logo", { exact: true });

  // SVG refusé, aucun logo enregistré.
  await logoInput.setInputFiles({ name: "logo.svg", mimeType: "image/svg+xml", buffer: SVG });
  await page.getByRole("button", { name: "Enregistrer" }).click();
  await expect(
    page.getByText("Déposez une image PNG, JPEG ou WebP de 1 Mo au plus."),
  ).toBeVisible();

  // PNG : aperçu immédiat, puis barre latérale après l'enregistrement.
  await logoInput.setInputFiles({ name: "logo.png", mimeType: "image/png", buffer: PNG });
  await expect(preview.locator("img").first()).toBeVisible();
  await expect(page.locator("aside").first().locator("img")).toHaveCount(0);

  // Couleur claire : l'aperçu change, pas le reste de l'interface.
  const primary = page.getByLabel("Couleur principale", { exact: true });
  await primary.fill("#fff3b0");
  await expect(preview.getByText("Certains textes ont été assombris")).toBeVisible();
  const rootBrand = () =>
    page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue("--brand"));
  expect((await rootBrand()).trim()).toBe("31 82 120");

  await page.getByRole("button", { name: "Enregistrer" }).click();
  await expect(page.getByText("Apparence enregistrée.", { exact: true }).last()).toBeVisible();
  await expect(page.locator("aside").first().locator("img")).toHaveCount(1);
  expect((await rootBrand()).trim()).toBe("255 243 176");
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

  // Hors session : identité de l'association, sans clignotement de Frigg.
  // IP distincte, comme les autres tests (limitation de débit par IP).
  const anon = await (
    await browser.newContext({
      locale: "fr-FR",
      extraHTTPHeaders: { "X-Forwarded-For": "203.0.113.10, 198.51.100.77" },
    })
  ).newPage();
  await anon.goto("/connexion");
  await expect(
    anon.locator("#contenu").getByText("Les Sentiers du Val", { exact: false }),
  ).toBeVisible();
  expect(await anon.locator("#contenu > p").first().textContent()).not.toContain("Frigg");
  await expect(anon.locator("#contenu img")).toHaveCount(1);
  expect(
    (
      await anon.evaluate(() =>
        getComputedStyle(document.documentElement).getPropertyValue("--brand"),
      )
    ).trim(),
  ).toBe("255 243 176");
  const identity = await (await anon.request.get("/api/identity")).json();
  expect(Object.keys(identity).sort()).toEqual(["logoUrl", "name", "palette"]);
  await expectNoBlockingA11yViolations(anon);

  // E-mail de réinitialisation : bandeau aux couleurs, logo embarqué.
  const since = new Date();
  await anon.getByRole("link", { name: "Mot de passe oublié ?" }).click();
  await expect(anon.getByRole("heading", { level: 1, name: "Mot de passe oublié" })).toBeVisible();
  await anon.getByLabel(/^Adresse électronique/).fill("membre@exemple.fr");
  await anon.getByRole("button", { name: "Recevoir le lien" }).click();
  await expect(anon.getByText("Si un compte existe pour cette adresse")).toBeVisible();
  const mail = await waitForFullMail("membre@exemple.fr", since, "mot de passe");
  expect(mail.HTML).toContain("background:#fff3b0");
  expect(mail.HTML).toContain('src="cid:logo"');
  expect(mail.Inline.length).toBe(1);
  expect(mail.From.Name).toContain("Les Sentiers du Val");
  await anon.context().close();

  // Retrait du logo et couleurs par défaut, avec confirmation.
  await page.getByRole("button", { name: "Retirer le logo" }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Retirer le logo" }).click();
  await expect(page.getByText("Logo retiré.", { exact: true }).last()).toBeVisible();
  await page.getByRole("button", { name: "Rétablir les couleurs par défaut" }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Rétablir" }).click();
  await expect(
    page.getByText("Couleurs par défaut rétablies.", { exact: true }).last(),
  ).toBeVisible();
  await expect(page.locator("aside").first().locator("img")).toHaveCount(0);
  expect((await rootBrand()).trim()).toBe("31 82 120");
});
