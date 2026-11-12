# Contrat : site publié

## Adresses

Site : `https://balsakup.github.io/frigg/` (adresse fournie par GitHub Pages
pour `Balsakup/frigg`).

| Adresse | Contenu |
| --- | --- |
| `/frigg/` | Accueil de la version « release » ; sans release documentée, page de redirection vers `/frigg/main/` |
| `/frigg/<chemin>` | Page `<chemin>` de la version « release » |
| `/frigg/main/` | Accueil de la version « main » |
| `/frigg/main/<chemin>` | Page `<chemin>` de la version « main » |
| `/frigg/404.html` | Page introuvable par défaut, servie par GitHub Pages pour toute adresse inconnue |

Les adresses sont « propres » (`cleanUrls`) : `/frigg/installation/docker`,
sans `.html`. Aucune page ne s'appelle `main`.

## Arborescence des pages (`docs/`)

Noms de fichiers en anglais, titres en français :

```text
index.md                       Accueil (layout: home) : 3 portes d'entrée
installation/
  index.md                     Vue d'ensemble, prérequis
  docker.md                    Démarrer (docker run, Compose), volume, arrêt
  configuration.md             Toutes les variables d'environnement
  first-admin.md               Premier administrateur
  backup.md                    Sauvegarder et restaurer
  upgrade.md                   Mettre à jour
  reverse-proxy.md             Reverse proxy et point de santé
  hardening.md                 Durcissement
  helloasso.md                 Activer le paiement par carte
  email.md                     SMTP et file d'envoi
usage/
  index.md                     Rôles membre et administrateur, navigation
  account.md                   Connexion, mot de passe, adresse électronique
  admin-accounts.md            Comptes et invitations
  settings.md                  Paramètres de l'association
  seasons.md                   Saisons
  fees.md                      Tarifs de l'adhésion
  members.md                   Membres
  custom-fields.md             Champs personnalisés
  member-import.md             Importation des membres
  memberships.md               Adhésions, renouvellement déclaré
  card-payment.md              Paiement par carte
  adjustments.md               Remises et dons
  events.md                    Événements et inscriptions
  bulk-email.md                E-mails groupés
contributing/
  index.md                     Licence, CLA, proposer une modification
  setup.md                     Environnement de développement
  workflow.md                  Cycle Spec Kit, conventions de langue et de commit
  checks.md                    Contrôles avant fusion
  documentation.md             Rédiger et prévisualiser la documentation
  release.md                   Publier une release, image locale
```

Chaque page du guide d'utilisation commence, sous son titre, par
`**Rôle requis** : Membre` ou `**Rôle requis** : Administrateur`.

## Éléments de chaque page (thème par défaut)

| Élément | Comportement |
| --- | --- |
| Navigation | « Installation », « Utilisation », « Contribution », puis le sélecteur de version |
| Sélecteur de version | Texte : `v1.2.0-rc.1` ou `main (en développement)`. Menu : « Dernière release (vX.Y.Z) », absent sans release, et « main (en développement) ». Chaque entrée mène à l'accueil de la version |
| Barre latérale | Pages du guide courant, dans l'ordre ci-dessus |
| Recherche | Locale, limitée à la version consultée |

Aucun composant, style, police ni script propre au site.
