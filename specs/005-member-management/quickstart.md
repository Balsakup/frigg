# Quickstart : valider la gestion des membres

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Prérequis et mise en place : ceux de
[001 › quickstart.md](../001-auth-app-shell/quickstart.md) (Mailpit,
`create-admin`, `go tool task dev`).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 004 :

- la migration `00007_members.sql` s'applique sur une base vierge et sur une
  base de développement existante ;
- la matrice d'autorisation couvre les 10 nouvelles opérations (anonyme →
  401 ; membre → 403 sur les 5 opérations `admin` ; administrateur accepté
  partout) ;
- `members_isolation_test.go` : pour `get`, `update`, `delete-my-member`, un
  compte B reçoit `404` sur une fiche de A, une fiche sans compte et un
  identifiant inexistant, et la fiche de A est inchangée ; `list-my-members`
  ne renvoie jamais une fiche d'un autre compte ;
- tests en table de `internal/member` : nettoyage des noms, date (aujourd'hui,
  demain, 120 ans pile, 120 ans et un jour), adresse électronique, téléphone
  (`06 12 34 56 78`, `06.12.34.56.78`, `+32 470 12 34 56`, `0612`, `+1`,
  `abc`), adresse tout ou rien, code postal, calcul de l'âge (veille et jour
  d'anniversaire, 29 février) ;
- tests d'API : création, modification, suppression dans les deux espaces ;
  `422` du contrat ; doublon puis `confirmDuplicate` ; 21e fiche → `409` ;
  seconde fiche « Vous » → `422` ; changement et retrait du rattachement
  (`isSelf` remis à faux) ; recherche par fragment de nom accentué
  (« elod » trouve « Élodie ») et de téléphone ; tri sans accents ; filtre
  `account=none` ;
- `delete-account` : sans fiche (inchangé), `members` absent → `422`,
  `memberIds` périmé → `409`, `delete` (fiches effacées), `keep` (fiches sans
  compte, `isSelf` faux) ;
- Vitest : `formatPhone` ; Playwright + axe : `my-members.spec.ts`,
  `admin-members.spec.ts`, et `admin-accounts-delete.spec.ts` étendu.

## Parcours manuels

Avec un administrateur `admin@exemple.fr` et un membre `claire@exemple.fr`
(invité puis inscrit) :

1. **Espace membre (US1)** — connecté en Claire, « Mes membres » : état vide.
   Ajouter sa fiche avec « Il s'agit de moi » (adresse préremplie), puis
   « Léo Martin » né le 2015-04-12 : la liste montre « Vous » en premier et
   « 11 ans » pour Léo. Modifier le téléphone de Léo en `06.12.34.56.78` :
   affiché `06 12 34 56 78`. Supprimer une fiche après confirmation.
2. **Doublon** — ajouter à nouveau « léo MARTIN », même date : message de
   doublon sans mention d'un compte ; cocher « Enregistrer quand même » :
   la fiche est créée.
3. **Cloisonnement (US2)** — noter l'identifiant de la fiche de Léo ;
   connecté avec un autre membre, ouvrir `/mes-membres/{id}` : « Page
   introuvable » ; `curl` d'un `PUT` sur `/api/me/members/{id}` avec sa
   session : `404`.
4. **Administration (US3, US4)** — en administrateur, « Membres » :
   rechercher `mart` : Léo et Claire apparaissent avec le compte de Claire.
   Créer « Jeanne Petit » sans compte, filtrer « Sans compte » : elle
   apparaît. Lui rattacher le compte de Claire : elle apparaît dans « Mes
   membres » de Claire.
5. **Suppression d'un compte (US5)** — supprimer le compte de Claire :
   le dialogue liste ses fiches et impose un choix. Choisir « Conserver ces
   fiches sans compte » : les fiches apparaissent sous « Sans compte », sans
   badge « Vous ». Recommencer avec un autre compte et « Supprimer aussi ces
   fiches » : elles disparaissent.
6. **Concurrence** — ouvrir le dialogue de suppression d'un compte, ajouter
   une fiche à ce compte depuis un autre onglet, confirmer : alerte « La liste
   des fiches de ce compte a changé. », liste rechargée.
7. **Journaux** — les lignes `member.*` et `account.delete` contiennent les
   identifiants, jamais de nom, de date ni de coordonnées.
8. **Accessibilité** — parcours complet au clavier, à 320 px de large.
