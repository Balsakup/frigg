# Quickstart : valider les parcours publics

Ce guide valide la fonctionnalité de bout en bout. Contrats :
[api.md](./contracts/api.md) et [ui-routes.md](./contracts/ui-routes.md).
Modèle : [data-model.md](./data-model.md).

**Prérequis** : ceux de [012 › quickstart.md](../012-events/quickstart.md),
c'est-à-dire `create-admin`, `go tool task dev` et Mailpit pour lire les
e-mails. Il faut aussi :

- une saison en cours, avec un tarif « Plein » à 40 € ;
- le chèque, le virement et la carte activés, HelloAsso en mode test (008) ;
- un compte membre « Claire » (`claire@example.org`) ;
- une fiche importée « Paul Martin », sans compte, avec l'adresse
  `paul@example.org` et une adhésion validée « À payer ».

Les scénarios utilisent une **fenêtre de navigation privée** pour les parcours
publics.

## Contrôles automatisés (constitution v1.9.0)

```sh
go tool task check
```

Attendu, en plus de 014 :

- **Migration** : `00014_public_journeys.sql` s'applique sur une base vierge
  et sur une base `00013` peuplée. Les défauts sont `origin = 'internal'`,
  `via_public = 0`, `via_link = 0` et `public_membership_enabled = 0`, et
  `PRAGMA foreign_key_check` est vide.
- **Génération** : `sqlc generate`, `frigg openapi` et `bun run gen:api`
  sans diff.
- **Matrice d'autorisation** : 7 opérations publiques listées dans
  `allowedPublic` et 13 opérations `link`, refusées (`401`) à l'anonyme, au
  membre et à l'administrateur sans lien, ainsi qu'à un lien expiré ou
  révoqué. Les 4 opérations `admin` sont refusées au membre.
- **Isolation** : un lien ne voit que les fiches sans compte de son adresse.
- **Non-révélation** : les réponses de dépôt et de demande de lien sont
  identiques pour une adresse inconnue, une adresse de compte et une adresse
  de fiche.
- **Concurrence** : 50 confirmations simultanées sur 3 places donnent 3
  inscriptions.
- **Limite par adresse** : 6 dépôts en une heure produisent 5 e-mails.
- **Carte par lien** (`hatest`) : constatation unique, `account_id` à
  `NULL`, `via_link = 1`.
- **Playwright + axe** : `public-journeys.spec.ts` sans violation `serious`
  ni `critical`.

## Scénarios manuels

### 1. Accueil fermé par défaut (US5, SC-010)

1. Ouvrir `/adherer`. Attendu : « Les demandes d'adhésion ne sont pas
   ouvertes », sans formulaire.
2. `curl -X POST …/api/public/membership` avec un corps valide. Attendu :
   `409`.
3. En admin, ouvrir Paramètres › Général et activer « Accepter les demandes
   d'adhésion publiques ». Attendu : l'adresse `…/adherer` est affichée et
   copiable.

### 2. Demande d'adhésion sans compte (US1)

1. `/adherer` : la saison et le tarif « Plein — 40 € » sont affichés.
   Remplir pour « Léa Durand », `lea@example.org`, cocher le consentement et
   envoyer. Attendu : l'écran « Consultez votre messagerie ».
2. En admin, ouvrir Adhésions. Attendu : aucune demande de Léa.
3. Dans Mailpit, ouvrir l'e-mail de confirmation et suivre le lien. Attendu :
   « Demande enregistrée », et l'espace de suivi liste la demande « En
   attente ».
4. Suivre à nouveau le même lien. Attendu : « Ce lien n'est plus valable ».
5. En admin, la demande apparaît avec le badge « Publique » et l'auteur
   « Demande publique ». La valider.

### 3. Suivi et paiement par lien (US2)

1. Dans Mailpit, ouvrir l'e-mail de validation de Léa et suivre le lien.
   Attendu : la page de paiement s'ouvre sans connexion, avec 40 € et la
   référence `ADH-…`.
2. Vérifier que l'adresse de la page ne contient plus de jeton (le fragment a
   été effacé).
3. Payer par carte (carte de test HelloAsso). Attendu : retour sur
   `/suivi/adhesions/<id>`, statut « Payée », mention « Mode test », et un
   e-mail de confirmation dans Mailpit.
4. Ouvrir `/suivi/lien` et saisir `lea@example.org`, puis
   `inconnue@example.org`. Attendu : le même message à l'écran, et un seul
   e-mail dans Mailpit (pour Léa).

### 4. Fiche importée suivie par lien (US2, FR-040, SC-012)

1. En admin, ouvrir la fiche de Paul et cliquer « Envoyer un lien de suivi ».
2. Dans Mailpit, suivre le lien. Attendu : l'adhésion « À payer » de Paul.
   Annoncer « Chèque ». Attendu : « Paiement annoncé ».

### 5. Adresse d'un compte existant (US4, FR-033)

1. `/adherer` avec l'identité de Claire et `claire@example.org`. Attendu :
   le même écran neutre qu'au scénario 2.
2. Dans Mailpit, l'e-mail mentionne qu'un compte existe. Suivre le lien.
   Attendu : « Votre demande est dans votre espace membre » et le bouton « Se
   connecter ».
3. Se connecter en Claire. Attendu : la demande est listée dans « Adhésions »,
   portée par sa fiche existante, sans doublon.

### 6. Événement public (US3)

1. En admin, créer « Portes ouvertes », 2 places, 5 €, ouverture « Ouvert au
   public », puis publier. Copier le lien public.
2. Ouvrir le lien en navigation privée. Attendu : détails, « 2 places
   restantes », aucun inscrit visible.
3. S'inscrire en « Marc », puis en « Nina » sans confirmer. Confirmer Marc.
   Attendu : « Inscription confirmée », « 1 place restante ».
4. Inscrire en admin un membre sur la dernière place, puis confirmer Nina.
   Attendu : « L'événement est complet » et aucune inscription pour Nina.
5. Régénérer le lien public. Attendu : l'ancien lien affiche « Événement
   introuvable ».
6. Changer l'ouverture en « Toutes les fiches ». Attendu : le lien affiche
   « Événement introuvable », et l'inscription de Marc reste confirmée.

### 7. Création de compte (US4)

1. Depuis l'espace de suivi de Léa, cliquer « Créer un compte » et choisir un
   mot de passe. Attendu : connectée, sa fiche porte « Vous » et l'adhésion
   payée est listée.
2. Rouvrir un ancien lien personnel de Léa. Attendu : « Ce lien n'est plus
   valable ».

### 8. Rattachement à une fiche existante (US5, FR-038)

1. Déposer et confirmer une demande publique pour « Paul Martin », avec la
   même date de naissance et `paul.m@example.org`.
2. En admin, ouvrir la demande. Attendu : l'encart « Doublon probable »
   désigne la fiche importée de Paul.
3. Cliquer « Rattacher à cette fiche ». Attendu : comme Paul a déjà une
   adhésion validée sur la saison, le rattachement est refusé (`409`) avec le
   message de 006 FR-003.

### 9. Anti-abus (FR-028, FR-029)

1. Remplir le champ piège par la console (`website`) et envoyer. Attendu :
   l'écran neutre et aucun e-mail.
2. Envoyer 6 dépôts pour la même adresse. Attendu : 5 e-mails au plus.
3. Au clavier seul, parcourir le formulaire. Attendu : le piège n'est jamais
   atteint.

### 10. Fiches inactives (FR-041)

1. Avec l'horloge injectée des tests, ou une base de recette vieillie de
   13 mois, partir d'une fiche publique dont l'événement est terminé.
   Attendu : l'encart du tableau de bord « 1 fiche publique inactive ».
2. Filtrer « Inactives à supprimer » et cliquer « Supprimer les 1 fiches ».
   Attendu : le dialogue de confirmation indique le nombre, puis la fiche est
   supprimée.
