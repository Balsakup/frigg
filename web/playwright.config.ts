import { defineConfig, devices } from "@playwright/test";

// Les tests tournent contre le binaire réel (SPA embarqué + API) sur une base
// jetable recréée à chaque lancement. Prérequis : `docker compose up -d`
// (Mailpit) et un binaire à jour (`go tool task e2e` s'en charge).
const port = 8090;
const db = "../data/e2e/frigg.db";
// Second serveur pour les adhésions (006) : sa base contient une saison ouverte,
// des tarifs et les moyens de paiement, alors que le serveur principal reste
// sans saison (les saisons sont un état global de l'instance, test 003).
const membershipsPort = 8091;
const membershipsDb = "../data/e2e/frigg-adhesions.db";
const membershipSpecs = /(memberships|admin-memberships)\.spec\.ts/;
const settingsSpec = /admin-settings\.spec\.ts/;
// Troisième serveur pour le paiement par carte (008) : jeu des adhésions, carte
// activée en sandbox et relié à un faux HelloAsso local (e2e/fake-helloasso.ts).
const cardPort = 8092;
const cardDb = "../data/e2e/frigg-carte.db";
const fakeHelloAssoPort = 8093;
const fakeHelloAssoSecret = "secret-e2e-ne-pas-utiliser";
const cardSpecs = /card-payment\.spec\.ts/;
const cardSettingsSpec = /card-settings\.spec\.ts/;
// Quatrième serveur pour les communications (009) : saison ouverte et adhérents
// dans chaque situation de paiement (e2eseed, mode « communication »).
const communicationPort = 8094;
const communicationDb = "../data/e2e/frigg-communication.db";
const communicationSpecs = /communications\.spec\.ts/;
// Cinquième serveur pour les paramètres généraux (010) : le nom, les couleurs et
// le logo sont globaux et changeraient l'affichage des autres tests.
const generalPort = 8095;
const generalDb = "../data/e2e/frigg-parametres.db";
const generalSpec = /general-settings\.spec\.ts/;
const appearanceSpec = /appearance\.spec\.ts/;

const serverEnv = (dbPath: string, listen: number, files: string) => ({
  FRIGG_DATABASE_PATH: dbPath,
  FRIGG_DATA_DIR: files,
  FRIGG_LISTEN_ADDR: `:${listen}`,
  FRIGG_BASE_URL: `http://localhost:${listen}`,
  FRIGG_COOKIE_SECURE: "true",
  // Chaque test se présente avec sa propre IP (voir e2e/test.ts) pour ne pas
  // épuiser la limite de connexions par IP.
  FRIGG_TRUSTED_PROXIES: "127.0.0.1,::1",
  FRIGG_SMTP_HOST: "localhost",
  FRIGG_SMTP_PORT: "1025",
  FRIGG_SMTP_TLS: "none",
  FRIGG_MAIL_FROM: "frigg@localhost",
  FRIGG_MAIL_POLL_INTERVAL: "1s",
  FRIGG_LOG_LEVEL: "warn",
});

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: `http://localhost:${port}`,
    locale: "fr-FR",
    trace: "on-first-retry",
    actionTimeout: 10_000,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      testIgnore: [
        membershipSpecs,
        settingsSpec,
        cardSpecs,
        cardSettingsSpec,
        communicationSpecs,
        generalSpec,
        appearanceSpec,
      ],
    },
    {
      name: "adhesions",
      use: { ...devices["Desktop Chrome"], baseURL: `http://localhost:${membershipsPort}` },
      testMatch: membershipSpecs,
    },
    // Les paramètres de paiement sont globaux : leur test passe après ceux des
    // adhésions, qui en dépendent.
    {
      name: "adhesions-parametres",
      use: { ...devices["Desktop Chrome"], baseURL: `http://localhost:${membershipsPort}` },
      testMatch: settingsSpec,
      dependencies: ["adhesions"],
    },
    {
      name: "carte",
      use: { ...devices["Desktop Chrome"], baseURL: `http://localhost:${cardPort}` },
      testMatch: cardSpecs,
    },
    // Le paramétrage de la carte est global : il passe après les paiements.
    {
      name: "carte-parametres",
      use: { ...devices["Desktop Chrome"], baseURL: `http://localhost:${cardPort}` },
      testMatch: cardSettingsSpec,
      dependencies: ["carte"],
    },
    {
      name: "communication",
      use: { ...devices["Desktop Chrome"], baseURL: `http://localhost:${communicationPort}` },
      testMatch: communicationSpecs,
    },
    {
      name: "parametres-generaux",
      use: { ...devices["Desktop Chrome"], baseURL: `http://localhost:${generalPort}` },
      testMatch: generalSpec,
    },
    // L'apparence change les couleurs et le logo de toute l'instance : elle passe
    // après les paramètres généraux.
    {
      name: "apparence",
      use: { ...devices["Desktop Chrome"], baseURL: `http://localhost:${generalPort}` },
      testMatch: appearanceSpec,
      dependencies: ["parametres-generaux"],
    },
  ],
  webServer: [
    {
      command: `go run ../internal/tools/e2eseed ${db} && ../bin/frigg serve`,
      url: `http://localhost:${port}`,
      reuseExistingServer: false,
      env: serverEnv(db, port, "../data/e2e/files"),
    },
    {
      command: `go run ../internal/tools/e2eseed ${membershipsDb} adhesions && ../bin/frigg serve`,
      url: `http://localhost:${membershipsPort}`,
      reuseExistingServer: false,
      env: serverEnv(membershipsDb, membershipsPort, "../data/e2e/files-adhesions"),
    },
    {
      command: `bun e2e/fake-helloasso.ts ${fakeHelloAssoPort} http://localhost:${cardPort} ${fakeHelloAssoSecret}`,
      url: `http://localhost:${fakeHelloAssoPort}/v5/organizations/sentiers-du-val`,
      reuseExistingServer: false,
    },
    {
      command: `go run ../internal/tools/e2eseed ${cardDb} carte && ../bin/frigg serve`,
      url: `http://localhost:${cardPort}`,
      reuseExistingServer: false,
      env: {
        ...serverEnv(cardDb, cardPort, "../data/e2e/files-carte"),
        FRIGG_HELLOASSO_CLIENT_ID: "client-e2e",
        FRIGG_HELLOASSO_CLIENT_SECRET: fakeHelloAssoSecret,
        FRIGG_HELLOASSO_ORGANIZATION: "sentiers-du-val",
        FRIGG_HELLOASSO_ENVIRONMENT: "sandbox",
        FRIGG_HELLOASSO_API_URL: `http://localhost:${fakeHelloAssoPort}`,
      },
    },
    {
      command: `go run ../internal/tools/e2eseed ${communicationDb} communication && ../bin/frigg serve`,
      url: `http://localhost:${communicationPort}`,
      reuseExistingServer: false,
      env: serverEnv(communicationDb, communicationPort, "../data/e2e/files-communication"),
    },
    {
      command: `go run ../internal/tools/e2eseed ${generalDb} parametres && ../bin/frigg serve`,
      url: `http://localhost:${generalPort}`,
      reuseExistingServer: false,
      env: serverEnv(generalDb, generalPort, "../data/e2e/files-parametres"),
    },
  ],
});
