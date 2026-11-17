# Contrat des écrans : parcours publics

Les écrans publics utilisent un nouveau layout, `routes/public/layout.tsx`
([R15](../research.md)). Il reprend l'habillage des écrans hors session :
identité visuelle par `get-identity`, carte centrée, pied de page, sans
barre latérale. Ils sont construits avec les composants `ui/*`, les tokens et
base-ui (principe IV). Les écrans sont utilisables dès 320 px et respectent
le RGAA : `fieldset`, erreurs reliées, `aria-live` (FR-044).

Aucune route publique ne figure dans la navigation : on y accède seulement
par une adresse diffusée ou un lien d'e-mail.

## Écrans publics

| Route | Fichier | Contenu |
| --- | --- | --- |
| `/adherer` | `routes/public/membership.tsx` | Page d'adhésion publique (US1) |
| `/e/:slug` | `routes/public/event.tsx` | Page publique d'un événement (US3) |
| `/confirmer-demande` | `routes/public/confirm.tsx` | Confirmation d'un dépôt, jeton dans le fragment (US1, US3) |
| `/suivi` | `routes/public/tracking.tsx` | Espace de suivi ; ouverture par fragment `#jeton=` (US2) |
| `/suivi/lien` | `routes/public/tracking-request.tsx` | « Recevoir mon lien de suivi » (US2) |
| `/suivi/adhesions/:id` | `routes/public/tracking-membership.tsx` | Demande, annulation, page de paiement, retour de HelloAsso (US2) |
| `/suivi/inscriptions/:id` | `routes/public/tracking-registration.tsx` | Inscription, annulation, paiement (US3) |
| `/suivi/compte` | `routes/public/tracking-account.tsx` | Création d'un compte : mot de passe et fiche « Vous » (US4) |

### `/adherer`

- **Si l'accueil est fermé** : un message « Les demandes d'adhésion ne sont
  pas ouvertes pour le moment. », sans formulaire.
- **Si une session de compte est détectée** : un bandeau « Vous êtes
  connecté : faites votre demande depuis votre espace » avec un lien vers
  `/adhesions/demande`.

Le formulaire est découpé en `fieldset` :

1. **« Vous »** : prénom, nom, date de naissance.
2. **« Vous joindre »** : adresse électronique (obligatoire), téléphone, et
   adresse postale facultative et repliable.
3. **« Votre adhésion »** :
   - la saison, présentée seulement s'il y en a plusieurs ;
   - le tarif, en `RadioGroup`, avec son libellé, son ou ses montants et sa
     description ;
   - le type (« Première adhésion » ou « Renouvellement ») ;
   - le montant correspondant, annoncé en `aria-live`.
4. **Champs personnalisés** : membre puis adhésion, avec les composants de
   011.
5. **Consentement** : une case obligatoire, avec le texte de FR-007.
6. **« Créer un compte pour suivre mes demandes »** : une case facultative,
   avec une aide.
7. **Piège** : invisible ([R10](../research.md)).

Le bouton « Envoyer ma demande » mène, une fois la demande envoyée, à un
écran « Consultez votre messagerie », qui reprend l'adresse saisie et donne
un lien « Je n'ai rien reçu » vers `/suivi/lien`.

### `/e/:slug`

- **En-tête** : titre, dates (formats de 012), lieu.
- **Corps** : description, puis le prix ou « Gratuit », la période
  d'inscription et les places restantes.
- **Si l'événement est annulé, terminé, complet ou hors période** : un encart
  d'information avec la raison, sans formulaire.
- **Sinon** : un formulaire avec l'identité, l'adresse, le consentement,
  « Créer un compte » et le piège, sans champ personnalisé (FR-026). Le bouton est
  « M'inscrire » (« M'inscrire — 35 € » si l'événement est payant).
- **Si le lien est inconnu** : la page « Événement introuvable », identique
  pour tous les cas de FR-025.

### `/confirmer-demande`

L'écran lit `#jeton=`, efface le fragment, appelle
`confirm-public-submission`, puis affiche :

| Issue | Écran |
| --- | --- |
| `tracking` | « Demande enregistrée » ou « Inscription confirmée », avec la mention « Un e-mail vous a été envoyé avec votre lien de suivi ». À partir de US2, s'y ajoute un bouton « Voir mon suivi » vers `/suivi`. À partir de US4, si `createAccount`, l'écran redirige vers `/suivi/compte`. |
| `account` | « Votre demande est dans votre espace membre », avec un bouton « Se connecter » vers `/connexion?suite=/adhesions` (ou `/mes-inscriptions`). |
| `duplicate` | « Cette demande existe déjà », avec un lien vers `/suivi`. |
| `closed` | Message de la raison, avec un lien vers `/adherer` ou vers la page de l'événement. |
| `410` | « Ce lien n'est plus valable », avec les liens « Recommencer ma demande » et « Recevoir mon lien de suivi ». |

### `/suivi`

- **À l'ouverture** : si le fragment `#jeton=` est présent, l'écran appelle
  `open-tracking-link` puis efface le fragment. Il appelle ensuite
  `get-tracking`.
- **Contenu** : une liste par fiche (titre de niveau 2 au nom de la fiche), où
  chaque élément est une carte avec son libellé, un badge d'état (tokens de
  006 et 012), le montant à payer et l'état du paiement, et une action
  « Payer » ou « Voir ».
- **Encart « Créer un compte »** : affiché si `canCreateAccount`.
- **Bouton « Fermer l'accès sur cet appareil »** : appelle `close-tracking`.
- **Si l'accès est absent ou expiré (`401`)** : le message « Ce lien n'est
  plus valable » et un lien vers `/suivi/lien`.

### `/suivi/adhesions/:id` et `/suivi/inscriptions/:id`

Ces écrans réutilisent les composants extraits en 012 : `PaymentPanel` et les
tentatives de carte. Ils les alimentent par un client « par lien ». Ils
réutilisent aussi le traitement du retour de HelloAsso (`?tentative=…&carte=`)
de `membership-detail.tsx`.

- **Annulation** : action « Annuler ma demande » ou « Annuler mon
  inscription », derrière un `ConfirmDialog`.
- **Aucun champ modifiable.** Une note explique : « Pour corriger vos
  informations, contactez l'association. »

### `/suivi/lien`

Un champ « Adresse électronique » et un bouton « Recevoir mon lien ». Le
message de réponse est toujours le même, quelle que soit l'adresse.

### `/suivi/compte`

- **Champ mot de passe** : avec la règle affichée, comme `accept-invitation`.
- **Fiche « Vous »** : avec plusieurs fiches, un `RadioGroup` « Laquelle de
  ces fiches est la vôtre ? », avec une option « Aucune ». Avec une seule
  fiche, la question n'est pas posée.
- **Une fois le compte créé** : la personne est redirigée vers `/`, connectée.

## Administration (modifications)

| Route | Modification |
| --- | --- |
| `/admin/parametres/general` | Section « Adhésions publiques » : interrupteur, adresse copiable (bouton « Copier »), rappel que les tarifs deviennent visibles de tous (FR-004) |
| `/admin/evenements/nouveau`, `/:id/modifier` | Option d'ouverture « Ouvert au public », avec l'aide « Toute personne qui reçoit le lien peut s'inscrire » |
| `/admin/evenements/:id` | Encart « Lien public » : adresse, « Copier », « Régénérer » (`ConfirmDialog`) |
| `/admin/membres` | Filtres « Origine : publique » et « Inactives à supprimer » ; badge « Publique » ; avec le filtre « Inactives », action « Supprimer les N fiches » (`ConfirmDialog` avec le nombre) |
| `/admin/membres/:id` | Origine, adresse confirmée le…, consentement le… ; « Envoyer un lien de suivi » si la fiche est sans compte et a une adresse ; avertissement sous le champ d'adresse d'une fiche sans compte (FR-040) |
| `/admin/membres/nouveau` | Même avertissement sous le champ d'adresse, tant qu'aucun compte n'est choisi (FR-040) |
| `/admin/adhesions`, `/admin/inscriptions` | Filtre « Origine : publique » ; badge « Publique » |
| `/admin/adhesions/:id` | Pour une demande publique en attente : auteur « Demande publique », encart « Doublon probable » avec « Rattacher à cette fiche » (`ConfirmDialog` qui nomme les deux fiches, FR-039) |
| `/admin` (tableau de bord) | Encart « N fiches publiques inactives » menant au filtre (FR-041) |

## Tests de bout en bout

`web/e2e/public-journeys.spec.ts` couvre :

- le dépôt, l'e-mail lu dans Mailpit, la confirmation, le suivi, le paiement
  de test et la création de compte ;
- l'événement public et la dernière place ;
- l'administration (paramètre, rattachement, inactives).

Il vérifie axe sur chaque écran public et l'inaccessibilité du piège au
clavier.
