import { QueryClientProvider } from "@tanstack/react-query";
import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "react-router";

import type { Route } from "./+types/root";
import "@fontsource/atkinson-hyperlegible/400.css";
import "@fontsource/atkinson-hyperlegible/400-italic.css";
import "@fontsource/atkinson-hyperlegible/700.css";
import "@fontsource-variable/bricolage-grotesque/opsz.css";
import "./app.css";
import { identityQuery } from "./api/identity";
import { queryClient } from "./api/query-client";
import { BrandStyle } from "./shell/BrandStyle";

export function meta(_: Route.MetaArgs) {
  return [{ title: "Frigg" }];
}

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="fr">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
        <meta name="robots" content="noindex, nofollow" />
        <Meta />
        <Links />
      </head>
      <body className="font-sans">
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

// L'identité visuelle (nom, logo, couleurs) est chargée avant le premier
// rendu : aucun clignotement des couleurs de Frigg (010 › R5). Pendant ce
// temps, HydrateFallback, neutre, s'affiche.
export async function clientLoader() {
  await queryClient.ensureQueryData(identityQuery);
  return null;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrandStyle />
      <Outlet />
    </QueryClientProvider>
  );
}

export function HydrateFallback() {
  return (
    <p role="status" className="p-6 text-muted">
      Chargement…
    </p>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let title = "Une erreur inattendue s'est produite";
  let details = "Rechargez la page. Si le problème persiste, contactez l'association.";
  let stack: string | undefined;

  if (isRouteErrorResponse(error)) {
    if (error.status === 404) {
      title = "Page introuvable";
      details = "La page demandée n'existe pas.";
    }
  } else if (import.meta.env.DEV && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="mx-auto max-w-prose p-6">
      <h1 className="font-display text-2xl font-bold">{title}</h1>
      <p className="mt-2 text-muted">{details}</p>
      <a
        href="/"
        className="mt-4 inline-block font-bold text-brand-ink underline underline-offset-2"
      >
        Revenir à l'accueil
      </a>
      {stack ? (
        <pre className="mt-4 overflow-x-auto rounded-md bg-surface p-4 text-sm">
          <code>{stack}</code>
        </pre>
      ) : null}
    </main>
  );
}
