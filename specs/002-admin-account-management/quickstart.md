# Quickstart : valider la gestion des comptes

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Prérequis et mise en place : ceux de
[001 › quickstart.md](../001-auth-app-shell/quickstart.md) (Mailpit,
`create-admin`, `go tool task dev`).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 001 :

- les migrations modifiées s'appliquent sur une base vierge (les bases de
  développement antérieures sont recréées : `00001` a changé) ;
- la matrice d'autorisation couvre les 7 nouvelles opérations (anonyme → 401,
  membre → 403, administrateur accepté) ;
- tests Go des garde-fous : action sur soi-même → 409 ; dernier administrateur
  actif → 409 ; deux administrateurs qui se désactivent mutuellement en
  parallèle → un seul succès, il reste un administrateur actif ;
- tests de `FTSTrigramQuery` (accents, saisie vide, mots de moins de
  3 caractères, guillemets et syntaxe FTS5 neutralisés) et de la recherche
  infixe (`pont` → `dupont`) ;
- Playwright + axe sur `/admin/comptes` et `/admin/comptes/:id`, y compris à
  320 px : aucune violation `serious` ou `critical`.

## Jeu de données

`go run ./internal/tools/e2eseed <base>` ajoute pour cette fonctionnalité des
comptes dédiés aux tests (désactivé, administrateur à rétrograder, compte à
supprimer ayant envoyé une invitation) et une trentaine de comptes
`membre-liste-NN@exemple.fr` pour la pagination.

## Scénarios manuels

| # | Parcours (spec) | Étapes | Résultat attendu |
| --- | --- | --- | --- |
| 1 | US1 — liste | En administrateur, ouvrir `/admin/comptes` | Tableau trié par adresse, total affiché, pagination si plus de 25 comptes ; son propre compte marqué « (vous) » |
| 2 | US1 — recherche | Saisir `du`, puis `pont`, puis `LEFÈ`, puis `exemple dup` | `du` : indication « Saisissez au moins 3 caractères. », liste entière ; ensuite, comptes dont l'adresse contient tous les fragments, où qu'ils soient, sans tenir compte des accents ni de la casse ; l'URL contient `q=` ; recharger conserve les critères |
| 3 | US1 — filtres | Rôle « Administrateurs » + État « Désactivés », puis « Effacer les critères » | Filtres combinés ; « Aucun compte ne correspond » si vide ; retour à la liste complète |
| 4 | US2 — désactivation | Navigateur A connecté en `membre@…` ; navigateur B (admin) désactive ce compte | Dans A, l'action suivante renvoie vers la connexion ; reconnexion refusée avec le message d'identifiants incorrects ; « Mot de passe oublié » → message neutre, aucun e-mail dans Mailpit |
| 5 | US2 — réactivation | Réactiver le compte | Connexion possible avec l'ancien mot de passe |
| 6 | US3 — promotion | Passer un membre administrateur ; se connecter avec | Sélecteur d'espace et rubriques d'administration présents |
| 7 | US3 — rétrogradation en cours de session | Navigateur A admin sur `/admin/comptes` ; B le rétrograde ; A change de page | A est renvoyé sur `/` avec « Vos droits d'administration ont été retirés. », sélecteur absent |
| 8 | US4 — suppression | Supprimer un compte ayant envoyé une invitation en attente | Retour à la liste avec confirmation ; l'invitation affiche « Compte supprimé » ; son adresse peut être réinvitée ; connexion avec l'adresse → message d'identifiants incorrects |
| 9 | US5 — lien de réinitialisation | Depuis la fiche d'un compte actif, « Envoyer un lien » | E-mail dans Mailpit ; lien valable 1 h, à usage unique ; action absente sur un compte désactivé |
| 10 | Garde-fous | Ouvrir sa propre fiche ; `curl` `POST /api/admin/accounts/{son id}/disable` avec sa session | Aucune action proposée ; API → 409 |
| 11 | Journaux | Effectuer chaque action, lire la sortie du serveur | Une ligne `INFO` par action avec `action`, `actor_id`, `target_id`, sans adresse électronique |
| 12 | Accessibilité | `/admin/comptes` et une fiche au clavier seul, à 320 px, avec un lecteur d'écran | Pas de défilement horizontal de la page ; nombre de résultats et confirmations annoncés ; focus rendu après fermeture d'un dialogue |
