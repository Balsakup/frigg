# Contrat d'interface : rubrique « Comptes »

Complète [001 › ui-routes.md](../../001-auth-app-shell/contracts/ui-routes.md).
Aucune route publique ajoutée ; la navigation est inchangée (« Comptes » reste
dans les ressources de l'espace administration).

## Routes

| Chemin | Fichier de route | Accès | Contenu |
| --- | --- | --- | --- |
| `/admin/comptes?q=&role=&statut=&page=` | `routes/app/admin/accounts.tsx` | admin | Section « Comptes » (recherche, filtres, tableau, pagination) puis section « Invitations » (existante) |
| `/admin/comptes/:id` | `routes/app/admin/account-detail.tsx` | admin | Fiche d'un compte et ses actions ; compte inexistant → « Page introuvable » |

Valeurs d'URL (françaises, visibles) → API : `role=membre|admin` →
`member|admin` ; `statut=actif|desactive` → `active|disabled`. Valeur inconnue
→ ignorée (filtre « Tous »).

## Liste des comptes

- **Barre de critères** : champ « Rechercher une adresse » (`type="search"`,
  mise à jour de l'URL après 300 ms, `replace` ; indication « Saisissez au
  moins 3 caractères. » liée par `aria-describedby` tant que la saisie ne
  compte aucun mot de 3 caractères, liste non filtrée), liste « Rôle » (Tous,
  Membres, Administrateurs), liste « État » (Tous, Actifs, Désactivés), bouton
  « Effacer les critères » visible dès qu'un critère est actif.
- **Tableau** (`ui/Table`, `<caption>` « Comptes de l'association ») :
  Adresse (lien vers la fiche ; mention « (vous) » pour son propre compte) ·
  Rôle (badge) · État (badge `ok` Actif / `muted` Désactivé) · Adresse
  confirmée (Oui / Non) · Créé le · Dernière connexion (« Jamais » si nulle).
- **Pagination** (`ui/Pagination`) : « Page n sur m », Précédente / Suivante ;
  masquée s'il n'y a qu'une page.
- **Région `role=status`** : « 42 comptes » / « Aucun compte ne correspond à
  ces critères. » (+ bouton « Effacer les critères »).
- À 320 px, le tableau défile horizontalement dans son `ScrollRegion`
  (focusable, étiqueté), jamais la page.

## Fiche d'un compte

- Fil d'Ariane : Tableau de bord › Comptes › {adresse}. Titre `h1` = adresse ;
  introduction = rôle et état (badges).
- **Informations** (liste de définitions) : rôle, état (et « depuis le … » si
  désactivé), adresse confirmée, créé le, dernière connexion.
- **Actions de l'en-tête de page** (masquées sur sa propre fiche, remplacées
  par une note « Vous ne pouvez pas modifier votre propre compte ici. Utilisez
  “Mon compte”. ») :

| Action | Affichée si | Dialogue de confirmation |
| --- | --- | --- |
| Changer le rôle | toujours | Titre « Changer le rôle de {adresse} » ; liste Membre / Administrateur ; bouton « Enregistrer » |
| Envoyer un lien de réinitialisation | compte actif | « Un e-mail de réinitialisation sera envoyé à {adresse}. » ; bouton « Envoyer » |
| Désactiver | compte actif | « {adresse} sera déconnecté de tous ses appareils et ne pourra plus se connecter. Vous pourrez réactiver le compte. » ; bouton `danger` « Désactiver » |
| Réactiver | compte désactivé | pas de dialogue (action non destructive) |
| Supprimer | toujours | « Supprimer définitivement {adresse} ? Cette action est irréversible. » + lien « Désactiver plutôt » (si actif) ; bouton `danger` « Supprimer » |

- Après chaque action : message `Alert` `success` + annonce `role=status`,
  invalidation de `get-account` et `list-accounts`. Après suppression :
  navigation vers `/admin/comptes` avec le message « Le compte {adresse} a été
  supprimé. ».
- Erreur `409` / `404` : `Alert` `danger` avec le `detail` du serveur ; `404`
  rafraîchit la fiche (qui devient « Page introuvable »).

## Retrait des droits en cours de navigation

Un `403` reçu sur `/api/admin/*` invalide la session. Si le rôle passe de
`admin` à `member`, l'espace mémorisé repasse à « Membre », la personne est
envoyée sur `/` et voit « Vos droits d'administration ont été retirés. »
(voir [research.md › R4](../research.md#r4-changement-de-rôle)).

## Composants ajoutés

| Composant | Rôle |
| --- | --- |
| `ui/Table` | Tableau accessible (caption, `scope="col"`, `ScrollRegion`) ; repris par la liste des invitations |
| `ui/Pagination` | Navigation entre pages par liens (`aria-label="Pagination"`, `aria-current="page"`) |
| `ui/Select` | Liste déroulante base-ui habillée des tokens (filtres, choix du rôle) |
