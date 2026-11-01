# Contrat d'interface : Adhésions

Complète [005 › ui-routes.md](../../005-member-management/contracts/ui-routes.md).
Les rubriques « Adhésions » (espace membre et administration) quittent
`comingSoon` (`app/shell/navigation.ts`, `routes.ts`). Nouvelle entrée
« Paramètres » (icône `Settings`, section `settings`, entre « Comptes » et
« Aide »).

## Routes

| Route | Écran | Données |
| --- | --- | --- |
| `/adhesions` | mes adhésions | `list-my-memberships`, `get-membership-offer` |
| `/adhesions/demande` | demande | `get-membership-offer` (`?membre=` présélectionne la fiche) |
| `/adhesions/:id` | détail et paiement | `get-my-membership` ; `404` → « Page introuvable » |
| `/admin/adhesions` | liste et chiffres | `list-memberships`, `get-membership-stats`, `list-seasons` (`season`, `status`, `payment`, `q`, `page` dans l'URL) |
| `/admin/adhesions/nouvelle` | création directe | `list-seasons`, `list-membership-fees`, `list-members` (`?membre=`) |
| `/admin/adhesions/:id` | détail et décisions | `get-membership` |
| `/admin/parametres` | redirection vers la première section | — |
| `/admin/parametres/paiements` | paramètres › paiements | `get-payment-settings` |

Fil d'Ariane : « Adhésions › Demander une adhésion » / « Adhésions ›
{prénom} {nom} — {saison} » ; « Adhésions › … » et « Paramètres » côté
administration.

Libellés communs (`app/lib/format.ts`) : états « En attente de validation »,
« Validée », « Refusée », « Annulée » ; types « Première adhésion »,
« Renouvellement » ; paiement « À payer », « Paiement annoncé — {moyen} »,
« Payée le {date} », « Gratuite » ; moyens « Espèces », « Chèque »,
« Virement ». Badges : `warn` (en attente, à payer), `info` (paiement
annoncé), `ok` (validée payée, gratuite), `danger` (refusée), neutre
(annulée).

## « Adhésions » (`/adhesions`)

- En-tête : titre « Adhésions », introduction « Les adhésions de vos membres
  à l'association. », bouton « Demander une adhésion » (icône `CreditCard`)
  si `offer.canRequest`.
- Messages (`Alert` `info`) : aucun membre → « Ajoutez d'abord un membre
  pour demander son adhésion. » avec lien vers « Mes membres » ; aucune
  saison dans l'offre → « Les adhésions ne sont pas ouvertes pour le
  moment. ».
- Liste : une section par saison (titre `h2` « Saison {libellé} »), cartes
  par adhésion : prénom et nom (lien vers `/adhesions/:id`), tarif, type,
  montant, badge d'état et de paiement ; motif d'un refus ou d'une
  annulation ; bouton « Payer » si `actions.pay`.

## Demande (`/adhesions/demande`)

- « Saison » : `Select` si plusieurs saisons, sinon texte fixe.
- « Membre » : `RadioGroup` des fiches disponibles pour la saison ; une fiche
  déjà adhérente ou en attente est absente, avec la note « {prénom} a déjà
  une adhésion en cours pour cette saison. ».
- « Tarif » : `RadioGroup`, chaque option : libellé, montant(s)
  (« 60,00 € · Renouvellement 40,00 € »), description.
- « Était-il adhérent la saison passée ? » : `RadioGroup` « Non, première
  adhésion » / « Oui, renouvellement ».
- Récapitulatif `aria-live="polite"` : « Montant : 40,00 € » (`amounts.first`
  ou `amounts.renewal` du tarif choisi, selon le type) et rappel « Votre demande sera validée par le
  bureau ; vous recevrez ensuite les informations de paiement. ».
- Boutons « Envoyer la demande » / « Annuler ». Succès → `/adhesions`,
  notification « Demande envoyée. ». `409` → `Alert` `danger` et offre
  rechargée.

## Détail et paiement (`/adhesions/:id`)

- Liste de définitions : membre, saison, tarif, type (déclaré, puis retenu
  s'il diffère : « Renouvellement (retenu par le bureau) »), montant, état,
  date de la demande, motif éventuel.
- `actions.cancel` : bouton « Annuler la demande » → `ConfirmDialog`
  « Annuler la demande d'adhésion de {prénom} ? ».
- Section « Paiement » (`h2`), si `payment` non nul :
  - à payer ou annoncé : montant et référence en évidence ; `RadioGroup`
    « Comment souhaitez-vous payer ? » des `paymentMethods` (présélection du
    moyen annoncé) ; sous l'option sélectionnée, les informations :
    - Espèces : instructions ;
    - Chèque : « À l'ordre de {ordre} », adresse d'envoi, instructions,
      « Inscrivez la référence {réf.} au dos du chèque. » ;
    - Virement : titulaire, IBAN (groupé par 4), BIC, instructions,
      « Indiquez la référence {réf.} dans le libellé du virement. » ;
    - boutons « Copier » à côté de l'IBAN, du BIC et de la référence ;
      région `aria-live` « Copié. » ;
  - bouton « Confirmer ce moyen de paiement » (« Changer de moyen » si déjà
    annoncé) ; notification « Moyen de paiement enregistré. » ;
  - moyen annoncé désactivé : `Alert` `warn` « Ce moyen n'est plus proposé.
    Choisissez-en un autre. » ;
  - aucun moyen : `Alert` `info` « Contactez le bureau pour régler cette
    adhésion. » ;
  - payée : « Payée le {date} ({moyen}). ».
- Adhésion gratuite : « Adhésion gratuite : aucun paiement n'est
  demandé. ».

## Fiches membre (005, modifiées)

- `/mes-membres/:id` : section « Adhésions » (`list-my-memberships?member=`)
  avec, pour chaque saison de l'offre où la fiche n'a rien, « Non adhérent »
  et le lien « Demander l'adhésion » (`/adhesions/demande?membre={id}`).
  La suppression affiche le `409` en `Alert`.
- `/admin/membres/:id` : section « Adhésions » (`memberships` de
  `get-member`) avec liens vers `/admin/adhesions/:id` et bouton « Créer
  une adhésion » (`/admin/adhesions/nouvelle?membre={id}`). Le dialogue de
  suppression ajoute « Ses {n} adhésion(s) seront aussi supprimées : … »
  (saisons et états).
- Dialogue de suppression d'un compte (005) : « Supprimer aussi ces fiches
  et leurs adhésions ».

## « Adhésions » (`/admin/adhesions`)

- En-tête : titre « Adhésions », bouton « Créer une adhésion ».
- Si des demandes sont en attente (`get-membership-dashboard`) : texte
  « {n} demande(s) en attente » avec le lien « Afficher les demandes en
  attente » (`?season=all&status=pending`), quels que soient les filtres.
- Filtres (dans l'URL) : `Select` « Saison » (saisons + « Toutes ») ; `Select`
  « État » ; `Select` « Paiement » ; recherche « Nom ou référence ».
- Chiffres de la saison (si une saison est filtrée), liste de définitions en
  grille : adhésions validées (dont premières adhésions, renouvellements),
  demandes en attente, total attendu, encaissé, restant à percevoir (détail
  par moyen annoncé : « non choisi », espèces, chèque, virement) ;
  répartition par tarif dans un petit tableau (`<caption>` « Répartition
  par tarif »).
- Tableau (`<caption>` « Adhésions ») : Membre (lien, `th scope="row"`) ·
  Saison (si « Toutes ») · Tarif · Type · Montant · État · Paiement
  (badge + référence). Demandes en attente en tête. `ScrollRegion` à
  320 px. Pagination et total.

## Création directe (`/admin/adhesions/nouvelle`)

- « Membre » : Combobox base-ui sur `list-members?q=` (3 caractères), affiche
  « {nom} {prénom} — {âge} ans — {compte ou Sans compte} » ; présélection
  par `?membre=`.
- « Saison » (saisons ouvertes non closes avec tarif), « Tarif »
  (`RadioGroup`), « Type » (`RadioGroup`), montant calculé `aria-live`.
- Case « Règlement déjà reçu » : affiche « Moyen » (`RadioGroup`), « Date de
  réception » (`type="date"`, défaut aujourd'hui), « Commentaire »
  (facultatif).
- Succès → `/admin/adhesions/:id`, notification « Adhésion enregistrée. ».

## Détail (`/admin/adhesions/:id`)

- Informations : membre (lien vers la fiche), compte de rattachement (lien
  ou « Sans compte »), saison, tarif, type déclaré, type retenu, montant
  (« calculé avec le tarif actuel » si non figé), dates et auteurs, motif,
  aide « Adhérent la saison précédente dans Frigg : Oui / Non / Inconnu »
  (avec infobulle « Inconnu : la saison précédente n'est pas gérée dans
  Frigg. »).
- Actions selon `actions` :
  - « Valider » → dialogue : tarif (`RadioGroup`, présélection du tarif
    choisi), type (présélection du type déclaré), montant calculé ;
  - « Refuser » → dialogue avec « Motif » obligatoire (`textarea`,
    500 caractères) ;
  - « Corriger » → même dialogue que « Valider » ;
  - « Annuler l'adhésion » → `ConfirmDialog` `danger` avec « Motif » ; si
    payée : « Un éventuel remboursement se fait en dehors de Frigg. » ;
  - « Marquer comme payée » → dialogue : moyen (présélection du moyen
    annoncé), date (défaut aujourd'hui), commentaire ;
  - « Annuler la constatation » → `ConfirmDialog`.
- `409` : `Alert` `danger` avec le message, adhésion rechargée.

## Paramètres (`/admin/parametres/…`)

- Une sous-page par section, avec sa propre adresse ; `/admin/parametres`
  redirige vers la première. Fil d'Ariane « Paramètres › {section} », titre
  du document « {section} · Paramètres · {association} ». L'entrée
  « Paramètres » de la barre latérale reste active sur toutes les sections.
- Gabarit commun (`SettingsPage`) : en-tête « Paramètres », introduction
  « Réglages de l'association. », navigation « Sections des paramètres » :
  liste de liens (et non onglets ARIA : chaque section est une page), la
  section affichée portant `aria-current="page"` ; colonne à gauche sur grand
  écran, rangée défilante horizontalement à 320 px.
- Section « Paiements » (`/admin/parametres/paiements`) :
- Section « Moyens de paiement » (`h2`), introduction « Les informations
  des moyens activés sont présentées aux adhérents au moment de payer. » ;
  un `fieldset` par moyen, `legend` « Espèces » / « Chèque » / « Virement »,
  interrupteur `Switch` base-ui « Proposer ce moyen » ; champs toujours
  modifiables, marqués requis seulement si le moyen est activé :
  - Espèces : « Instructions » (`textarea`, 1 000) ;
  - Chèque : « Ordre » (120), « Adresse d'envoi » (`textarea`, 300),
    « Instructions » ;
  - Virement : « Titulaire du compte » (120), « IBAN » (`autocomplete="off"`,
    `spellcheck="false"`), « BIC », « Instructions ».
- Bouton « Enregistrer » ; notification « Paramètres enregistrés. » ; erreurs
  `422` sur les champs (`body.transfer.iban`…).

## Tableau de bord (`/admin`)

- `get-membership-dashboard` : `Alert` `info` « {n} demande(s) d'adhésion en
  attente. » + lien « Traiter les demandes »
  (`/admin/adhesions?season=all&status=pending`) ; `Alert` `warn` « Aucun
  moyen de paiement n'est proposé aux adhérents. » + lien « Paramétrer »
  (`/admin/parametres/paiements`).

## Accessibilité

- Chaque groupe de choix est un `fieldset` / `RadioGroup` avec légende ;
  erreurs reliées par `aria-describedby`.
- Montants calculés et « Copié. » annoncés en `aria-live="polite"`.
- Instructions affichées en texte brut avec `white-space: pre-line`.
- Parcours clavier complet sur les sept écrans et les dialogues ;
  Playwright + axe sans violation `serious` ou `critical`.
