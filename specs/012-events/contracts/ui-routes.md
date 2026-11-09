# Contrat des écrans : événements et inscriptions

Les rubriques existent déjà dans `web/app/shell/navigation.ts` avec
`comingSoon: true` ; ce drapeau est retiré et les entrées correspondantes
quittent `memberComingSoon` et `adminComingSoon` dans `web/app/routes.ts`
(FR-034). Shell, composants `ui/*`, tokens et base-ui de l'artefact de
référence (principe IV).

## Espace membre

| Route | Fichier | Contenu |
| --- | --- | --- |
| `/evenements` | `routes/app/member/events.tsx` | « Événements à venir » (US2) |
| `/evenements/:id` | `routes/app/member/event-detail.tsx` | Page d'un événement et inscription (US2, US3) |
| `/mes-inscriptions` | `routes/app/member/registrations.tsx` | « Mes inscriptions » (US3) |
| `/mes-inscriptions/:id` | `routes/app/member/registration-detail.tsx` | Inscription et page de paiement, retour HelloAsso (US4) |
| `/a-regler` | `routes/app/member/dues.tsx` | « À régler » (US4) |

### `/evenements`

Liste (`ul`) de cartes, par début croissant : date en bloc (jour, mois
abrégé), titre (`h2` lien vers le détail), heures (« sam. 12 déc. · 14 h –
17 h » ; sur plusieurs jours « du lun. 13 au ven. 17 avr. »), lieu, badge
prix (« Gratuit » ou « 35 € »), badge « Annulé » (`--danger`) ou « Complet »
(`--warn`), « 12 places restantes » si capacité. État vide : « Aucun
événement à venir pour le moment. »

### `/evenements/:id`

En-tête de page : titre, date, lieu. Corps : description (texte, retours à la
ligne conservés, jamais interprété), encart « Inscriptions » :

- période (« Inscriptions ouvertes jusqu'au 1er avril à 23 h 59 ») ou motif
  de fermeture ;
- ouverture (« Réservé aux adhérents de la saison 2026-2027 » ou « Ouvert à
  tous les membres ») ;
- `fieldset` « Qui inscrire ? » : une case par fiche gérée ; fiche inscrite
  → badge « Inscrit » et lien « Voir l'inscription » ; fiche non éligible →
  case désactivée et raison reliée par `aria-describedby` ;
- bouton « Inscrire » (« Inscrire et payer 35 € » si payant, montant × nombre
  de fiches) ; après succès : notification et, si payant, redirection vers
  `/mes-inscriptions/{createdRegistrationIds[0]}` (réponse `RegisterResult`) ;
- « Complet » : encart `--warn`, bouton absent.

Événement annulé : alerte `--danger` en tête avec le message d'annulation.

### `/mes-inscriptions`

Deux régions titrées « À venir » et « Passées » ; tableau responsive (cartes
sous 640 px) : membre, événement (lien), date, état (« Confirmée »,
« Annulée »), paiement (`MembershipBadges` réutilisé : « À payer »,
« Paiement annoncé — chèque », « Payée »), action « Payer » si `pay`. État
vide avec lien vers « Événements à venir ».

### `/mes-inscriptions/:id`

Récapitulatif (événement, membre, montant, référence), puis la **page de
paiement de 006/008** factorisée : `PaymentMethodInfo`, choix du moyen,
`CardPaymentPanel` (retour `?tentative=&carte=` traité au chargement comme
pour `/adhesions/:id`). Bouton « Annuler mon inscription » si
`actions.cancel` (`ConfirmDialog`). Inscription réglée : encart « Pour
annuler, contactez l'association. »

### `/a-regler`

Liste de `Due` : membre, intitulé, montant, référence, état, bouton « Payer »
vers `/adhesions/:id` ou `/mes-inscriptions/:id`. Total à régler en tête.
État vide : « Rien à régler. »

## Espace administration

| Route | Fichier | Contenu |
| --- | --- | --- |
| `/admin/evenements` | `routes/app/admin/events.tsx` | Liste (US1) |
| `/admin/evenements/nouveau` | `routes/app/admin/event-new.tsx` | Création (US1) |
| `/admin/evenements/:id` | `routes/app/admin/event-detail.tsx` | Détail, inscrits, totaux, actions (US1, US5, US6) |
| `/admin/evenements/:id/modifier` | `routes/app/admin/event-edit.tsx` | Modification (US1) |
| `/admin/inscriptions` | `routes/app/admin/registrations.tsx` | Toutes les inscriptions (US5) |
| `/admin/inscriptions/:id` | `routes/app/admin/registration-detail.tsx` | Une inscription : paiement, tentatives, signalements (US5) |

### `/admin/evenements`

Action d'en-tête « Créer un événement ». Filtres (paramètres d'URL) : saison
(`Select`, défaut saison courante), état (`Select`). Deux régions « À
venir » et « Passés ». Tableau : date, titre (lien), lieu, état (badges
« Brouillon » `--muted`, « Publié » `--ok`, « Annulé » `--danger`,
« Terminé » `--info`), prix, inscrits « 12 / 20 ». État vide avec
explication et bouton.

### Formulaire d'événement (`events/EventForm.tsx`)

`react-hook-form`, erreurs `422` par l'adaptateur commun. Champs : titre,
lieu, description (`textarea`), début et fin (`input type="datetime-local"`,
valeurs locales sans fuseau), `RadioGroup` « Qui peut s'inscrire ? »
(adhérents de la saison / toutes les fiches), capacité (facultative), prix
en euros (facultatif, texte libre « 15,50 » analysé par le serveur comme le
montant d'un tarif dans `FeeDialog`, aide « Laisser vide pour un événement
gratuit »), `Collapsible` « Période d'inscription » (ouverture, clôture).
Aucune déduction de la saison dans le navigateur (principe VI) : aide
statique « L'événement est rattaché à la saison qui contient sa date de
début. » ; en modification, la saison actuelle vient de `AdminEvent.season` ;
une date hors saison est signalée par le `422` sur `startsAt`. Modification d'un événement avec inscrits : avertissement
« Les inscrits seront prévenus » si début, fin ou lieu changent ; note
« Le nouveau prix s'appliquera aux prochaines inscriptions ».

### `/admin/evenements/:id`

En-tête : titre, badges d'état, actions selon `actions` : « Publier »,
« Modifier », menu « Plus » (« Annuler l'événement », « Rétablir »,
« Supprimer »). Sections :

- **Informations** (dates, lieu, ouverture, capacité, prix, période,
  description) ;
- **Chiffres** : inscrits / capacité, total dû, total encaissé (« dont mode
  test » le cas échéant) ;
- **Inscrits** : bouton « Inscrire un membre » (`Dialog` avec
  `MemberPicker` de 006 et, si payant, « Déjà payé » + moyen et date),
  bouton « Exporter (CSV) », bascule « Afficher les annulées » ; tableau :
  nom, prénom, adhérent (oui/non), badge `--warn` « Non éligible », date
  d'inscription, inscrit par, paiement, actions (« Constater le paiement »,
  « Retirer ») ; « non joignable » signalé.

Dialogues (`ConfirmDialog`) :

- annuler : `textarea` « Message aux inscrits (facultatif) », nombre de
  personnes prévenues, encart « n inscriptions payées (total) : le
  remboursement se fait hors de Frigg ou dans l'espace HelloAsso » ;
- supprimer : seulement si aucune inscription ;
- retirer une inscription payée : rappel de remboursement.

### `/admin/inscriptions`

Filtres : événement (`Combobox`), état, paiement, case « Signalements à
traiter » (`alerts=open`) ; recherche (nom, prénom, référence ; message
« 3 caractères minimum » sous 3). Tableau paginé : membre, événement, date
d'inscription, montant, référence, paiement, badge `--danger`
« n signalement(s) » si `openAlerts > 0`.

### Tableau de bord (`/admin`, modifié)

L'encart des signalements de paiement affiche un lien par cible non nulle :
« n signalement(s) sur les adhésions » → `/admin/adhesions?season=all&alerts=open`
(inchangé) et « n signalement(s) sur les inscriptions » →
`/admin/inscriptions?alerts=open`.

### `/admin/inscriptions/:id`

Calqué sur `/admin/adhesions/:id` : récapitulatif, paiement (constater,
annuler la constatation), tentatives par carte et signalements (composants
de 008 réutilisés), « Vérifier auprès de HelloAsso », « Retirer
l'inscription ».

## Composants partagés

- `payments/PaymentPanel.tsx` : page de paiement extraite de
  `routes/app/member/membership-detail.tsx`, paramétrée par la cible
  (adhésion ou inscription) et ses mutations ; utilisée par les deux pages.
- `payments/RecordPaymentDialog.tsx`, `payments/OnlinePayments.tsx`,
  `payments/PaymentAlerts.tsx` : extraits de `admin/membership-detail.tsx`.
- `events/EventDate.tsx` : formatage `Intl.DateTimeFormat("fr-FR",
  { timeZone: "Europe/Paris" })` d'une plage de dates.
- `api/events.ts`, `api/registrations.ts` : requêtes et mutations
  (invalidation explicite de `my-events`, `my-registrations`, `my-dues`,
  `events`, `event`, `registrations`, `membership-dashboard`).

## Accessibilité

Titres hiérarchisés, tableaux avec `caption` masquée, badges doublés de
texte, focus ramené sur le titre après une inscription réussie, annonce
`aria-live` du nombre de places restantes après échec « Complet », 320 px
(cartes empilées), `prefers-reduced-motion` respecté.
