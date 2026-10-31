import { index, layout, prefix, type RouteConfig, route } from "@react-router/dev/routes";

// Rubriques livrées par des fonctionnalités ultérieures (app/shell/navigation.ts).
const memberComingSoon = [
  "mes-membres",
  "mes-inscriptions",
  "a-regler",
  "evenements",
  "adhesions",
  "mes-documents",
  "aide",
];
const adminComingSoon = [
  "membres",
  "adhesions",
  "evenements",
  "inscriptions",
  "finances",
  "communication",
  "aide",
];

// Chaque user story ajoute ses routes ici en même temps que leurs fichiers
// (contracts/ui-routes.md).
export default [
  layout("routes/auth/layout.tsx", [
    route("connexion", "routes/auth/login.tsx"),
    route("invitation", "routes/auth/accept-invitation.tsx"),
    route("mot-de-passe-oublie", "routes/auth/forgot-password.tsx"),
    route("reinitialiser-mot-de-passe", "routes/auth/reset-password.tsx"),
    route("confirmer-adresse", "routes/auth/confirm-email.tsx"),
  ]),
  layout("routes/app/layout.tsx", [
    index("routes/app/member/home.tsx"),
    route("mon-compte", "routes/app/account.tsx"),
    ...memberComingSoon.map((p) => route(p, "routes/app/coming-soon.tsx", { id: `bientot-${p}` })),
    ...prefix("admin", [
      layout("routes/app/admin/layout.tsx", [
        index("routes/app/admin/home.tsx"),
        route("comptes", "routes/app/admin/accounts-layout.tsx", [
          index("routes/app/admin/accounts.tsx"),
          route(":id", "routes/app/admin/account-detail.tsx"),
        ]),
        ...adminComingSoon.map((p) =>
          route(p, "routes/app/coming-soon.tsx", { id: `bientot-admin-${p}` }),
        ),
      ]),
    ]),
    route("*", "routes/app/not-found.tsx"),
  ]),
] satisfies RouteConfig;
