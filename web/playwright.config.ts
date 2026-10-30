import { defineConfig, devices } from "@playwright/test";

// Les tests tournent contre le binaire réel (SPA embarqué + API) sur une base
// jetable recréée à chaque lancement. Prérequis : `docker compose up -d`
// (Mailpit) et un binaire à jour (`go tool task e2e` s'en charge).
const port = 8090;
const db = "../data/e2e/frigg.db";

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
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: `go run ../internal/tools/e2eseed ${db} && ../bin/frigg serve`,
    url: `http://localhost:${port}`,
    reuseExistingServer: false,
    env: {
      FRIGG_DATABASE_PATH: db,
      FRIGG_DATA_DIR: "../data/e2e/files",
      FRIGG_LISTEN_ADDR: `:${port}`,
      FRIGG_BASE_URL: `http://localhost:${port}`,
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
    },
  },
});
