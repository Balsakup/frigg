import { index, layout, prefix, type RouteConfig, route } from "@react-router/dev/routes";

// Rubriques livrées par des fonctionnalités ultérieures (app/shell/navigation.ts).
const memberComingSoon = ["mes-inscriptions", "a-regler", "evenements", "mes-documents", "aide"];
const adminComingSoon = ["evenements", "inscriptions", "finances", "communication", "aide"];

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
    route("mes-membres", "routes/app/member/my-members-layout.tsx", [
      index("routes/app/member/my-members.tsx"),
      route("nouveau", "routes/app/member/my-member-new.tsx"),
      route(":id", "routes/app/member/my-member-detail.tsx"),
    ]),
    route("adhesions", "routes/app/member/memberships-layout.tsx", [
      index("routes/app/member/memberships.tsx"),
      route("demande", "routes/app/member/membership-request.tsx"),
      route(":id", "routes/app/member/membership-detail.tsx"),
    ]),
    ...memberComingSoon.map((p) => route(p, "routes/app/coming-soon.tsx", { id: `bientot-${p}` })),
    ...prefix("admin", [
      layout("routes/app/admin/layout.tsx", [
        index("routes/app/admin/home.tsx"),
        route("comptes", "routes/app/admin/accounts-layout.tsx", [
          index("routes/app/admin/accounts.tsx"),
          route(":id", "routes/app/admin/account-detail.tsx"),
        ]),
        route("saisons", "routes/app/admin/seasons-layout.tsx", [
          index("routes/app/admin/seasons.tsx"),
          route("nouvelle", "routes/app/admin/season-new.tsx"),
          route(":id", "routes/app/admin/season-detail.tsx"),
        ]),
        route("membres", "routes/app/admin/members-layout.tsx", [
          index("routes/app/admin/members.tsx"),
          route("nouveau", "routes/app/admin/member-new.tsx"),
          route(":id", "routes/app/admin/member-detail.tsx"),
        ]),
        route("adhesions", "routes/app/admin/memberships-layout.tsx", [
          index("routes/app/admin/memberships.tsx"),
          route("nouvelle", "routes/app/admin/membership-new.tsx"),
          route(":id", "routes/app/admin/membership-detail.tsx"),
        ]),
        route("parametres", "routes/app/admin/settings-layout.tsx", [
          index("routes/app/admin/settings-index.tsx"),
          route("paiements", "routes/app/admin/settings-payments.tsx"),
        ]),
        ...adminComingSoon.map((p) =>
          route(p, "routes/app/coming-soon.tsx", { id: `bientot-admin-${p}` }),
        ),
      ]),
    ]),
    route("*", "routes/app/not-found.tsx"),
  ]),
] satisfies RouteConfig;
