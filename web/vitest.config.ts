import { defineConfig } from "vitest/config";

// Configuration distincte de vite.config.ts : le plugin React Router n'est pas
// prévu pour tourner sous Vitest.
export default defineConfig({
  resolve: {
    tsconfigPaths: true,
  },
  test: {
    environment: "jsdom",
    include: ["app/**/*.test.{ts,tsx}"],
    setupFiles: ["./vitest.setup.ts"],
  },
});
