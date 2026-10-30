import { reactRouter } from "@react-router/dev/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// En développement, /api est relayé vers le serveur Go : le navigateur reste sur
// une seule origine. L'en-tête Host n'est pas réécrit (changeOrigin: false) pour
// que http.CrossOriginProtection et les cookies se comportent comme en production.
const apiTarget = process.env.FRIGG_API_URL ?? "http://localhost:8080";

export default defineConfig({
  plugins: [tailwindcss(), reactRouter()],
  define: {
    __FRIGG_VERSION__: JSON.stringify(process.env.FRIGG_VERSION ?? "dev"),
  },
  resolve: {
    tsconfigPaths: true,
  },
  server: {
    proxy: {
      "/api": { target: apiTarget, changeOrigin: false },
    },
  },
});
