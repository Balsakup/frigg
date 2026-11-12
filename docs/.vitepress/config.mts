import { defineConfig } from "vitepress";

// Variables fournies par le workflow docs.yml. Contrat rétrocompatible :
// specs/015-vitepress-docs/contracts/build-env.md (ne jamais renommer ni
// supprimer une variable).
const base = process.env.DOCS_BASE ?? "/";
const version = process.env.DOCS_VERSION ?? "main";
const release = process.env.DOCS_RELEASE_VERSION ?? "";
const siteUrl = process.env.DOCS_SITE_URL ?? "https://balsakup.github.io/frigg/";

const mainLabel = "main (en développement)";

export default defineConfig({
  base,
  lang: "fr-FR",
  title: "Frigg",
  description: "Documentation de Frigg, gestion d'association auto-hébergée",
  cleanUrls: true,

  themeConfig: {
    nav: [
      { text: "Installation", link: "/installation/" },
      { text: "Utilisation", link: "/usage/" },
      { text: "Contribution", link: "/contributing/" },
      // Sélecteur de version : URL absolues, car un chemin absolu serait
      // préfixé par la base de la version construite.
      {
        text: version === "main" ? mainLabel : version,
        items: [
          ...(release
            ? [{ text: `Dernière release (${release})`, link: siteUrl, target: "_self" }]
            : []),
          { text: mainLabel, link: `${siteUrl}main/`, target: "_self" },
        ],
      },
    ],

    sidebar: {
      "/installation/": [
        {
          text: "Installation et exploitation",
          items: [
            { text: "Vue d'ensemble", link: "/installation/" },
            { text: "Démarrer avec Docker", link: "/installation/docker" },
            { text: "Configuration", link: "/installation/configuration" },
            { text: "Premier administrateur", link: "/installation/first-admin" },
            { text: "Sauvegarder et restaurer", link: "/installation/backup" },
            { text: "Mettre à jour", link: "/installation/upgrade" },
            { text: "Reverse proxy et santé", link: "/installation/reverse-proxy" },
            { text: "Durcissement", link: "/installation/hardening" },
            { text: "Paiement par carte (HelloAsso)", link: "/installation/helloasso" },
            { text: "E-mails", link: "/installation/email" },
          ],
        },
      ],
      "/usage/": [
        {
          text: "Utilisation",
          items: [
            { text: "Vue d'ensemble", link: "/usage/" },
            { text: "Votre compte", link: "/usage/account" },
            { text: "Comptes et invitations", link: "/usage/admin-accounts" },
            { text: "Paramètres de l'association", link: "/usage/settings" },
            { text: "Saisons", link: "/usage/seasons" },
            { text: "Tarifs de l'adhésion", link: "/usage/fees" },
            { text: "Membres", link: "/usage/members" },
            { text: "Champs personnalisés", link: "/usage/custom-fields" },
            { text: "Importation des membres", link: "/usage/member-import" },
            { text: "Adhésions", link: "/usage/memberships" },
            { text: "Paiement par carte", link: "/usage/card-payment" },
            { text: "Remises et dons", link: "/usage/adjustments" },
            { text: "Événements et inscriptions", link: "/usage/events" },
            { text: "E-mails groupés", link: "/usage/bulk-email" },
          ],
        },
      ],
      "/contributing/": [
        {
          text: "Contribution",
          items: [
            { text: "Contribuer à Frigg", link: "/contributing/" },
            { text: "Environnement de développement", link: "/contributing/setup" },
            { text: "Cycle de développement", link: "/contributing/workflow" },
            { text: "Contrôles avant fusion", link: "/contributing/checks" },
            { text: "Documentation", link: "/contributing/documentation" },
            { text: "Releases et image locale", link: "/contributing/release" },
          ],
        },
      ],
    },

    search: {
      provider: "local",
      options: {
        translations: {
          button: { buttonText: "Rechercher", buttonAriaLabel: "Rechercher" },
          modal: {
            noResultsText: "Aucun résultat pour",
            resetButtonTitle: "Effacer la recherche",
            footer: {
              selectText: "pour sélectionner",
              navigateText: "pour naviguer",
              closeText: "pour fermer",
            },
          },
        },
      },
    },

    outline: { label: "Sur cette page" },
    docFooter: { prev: "Page précédente", next: "Page suivante" },
    darkModeSwitchLabel: "Apparence",
    sidebarMenuLabel: "Menu",
    returnToTopLabel: "Retour en haut",
    skipToContentLabel: "Aller au contenu",
    notFound: {
      title: "Page introuvable",
      quote: "Cette page n'existe pas dans cette version de la documentation.",
      linkLabel: "Revenir à l'accueil",
      linkText: "Revenir à l'accueil",
    },

    socialLinks: [{ icon: "github", link: "https://github.com/Balsakup/frigg" }],
  },
});
