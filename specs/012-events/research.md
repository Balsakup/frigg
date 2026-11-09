# Research: Événements de l'association

**Feature**: `012-events` | **Date**: 2026-11-09 | **Spec**: [spec.md](./spec.md)

Les trois points ouverts de la spécification ont été tranchés le 2026-11-09
(inscriptions payantes, ouverture par événement, refus quand complet). Les
choix ci-dessous partent du code existant (003 saisons, 005 fiches, 006
adhésions et paiement, 008 HelloAsso, 009 e-mails groupés) et de la
constitution **v1.8.0**.

## R1. Stockage : `events` et `event_registrations`

**Decision** : migration `00012_events.sql` avec deux tables nouvelles
([data-model.md](./data-model.md)) :

- `events` : une ligne par événement, `season_id` figé à l'enregistrement
  (saison qui contient la date de début, en heure de Paris) ;
- `event_registrations` : une ligne par inscription, **colonnes de paiement
  calquées sur `memberships`** (`amount_cents`, `payment_reference`,
  `payment_status`, `announced_method`, `paid_method`, `paid_on`, `paid_by`,
  `paid_at`, `payment_note`, `paid_online_payment_id`,
  `paid_amount_cents`) avec les mêmes `CHECK`.

Unicité « une inscription confirmée par fiche et par événement » par un index
unique partiel `WHERE status = 'confirmed'` : une fiche désinscrite peut être
réinscrite (FR-014), l'historique est conservé.

**Rationale** : mêmes colonnes et mêmes contraintes que les adhésions, donc
mêmes règles du domaine de paiement (R3) et mêmes requêtes de constatation.
Une table par objet payable garde de vraies clés étrangères en `ON DELETE
CASCADE` vers `members` (droit à l'effacement, Edge Cases).

**Alternatives considered** :

- table `payments` commune aux adhésions et aux inscriptions : refonte des
  colonnes de `memberships` déjà publiées, migration de données risquée,
  pour un gain faible ;
- prix et paiement portés par une table d'« articles » générique : sur-conçu
  pour deux objets payables.

## R2. Rattacher les paiements par carte aux inscriptions : reconstruction de `online_payments` et `payment_alerts`

**Decision** : la migration `00012` reconstruit `online_payments` et
`payment_alerts` pour rendre `membership_id` facultatif et ajouter
`registration_id` (clé étrangère `ON DELETE CASCADE` vers
`event_registrations`), avec `CHECK ((membership_id IS NULL) <>
(registration_id IS NULL))`. Procédure SQLite documentée (« 12 étapes ») :

```text
-- +goose NO TRANSACTION
PRAGMA foreign_keys = OFF;
BEGIN;  …créer *_new, copier, DROP, RENAME, recréer index…  COMMIT;
PRAGMA foreign_keys = ON;
```

`foreign_keys = OFF` est indispensable : avec les clés actives, `DROP TABLE
online_payments` déclencherait le `ON DELETE SET NULL` de
`memberships.paid_online_payment_id`, ce que refuse le `CHECK` des
paiements par carte. Le pool d'écriture n'a qu'une connexion
(`SetMaxOpenConns(1)`) : le `PRAGMA` porte bien sur la connexion qui exécute
la migration, et le dernier statement le rétablit. Un test de migration
appelle `PRAGMA foreign_key_check` après coup.

**Vérifié pendant la recherche** (sonde jetable, supprimée) :

- `sqlc generate` accepte `PRAGMA`, `BEGIN`/`COMMIT` et `ALTER TABLE … RENAME
  TO` dans une migration et produit `MembershipID sql.NullInt64` ;
- goose v3.29 applique la migration `NO TRANSACTION` sur une base `00011`
  peuplée (adhésion payée par carte) : données conservées, `PRAGMA
  foreign_keys` revenu à 1, aucune violation, et la clé étrangère de
  `memberships.paid_online_payment_id` vers la table reconstruite agit
  toujours.

**Rationale** : un seul circuit de paiement par carte (création de
l'intention, notification, balayage, vérification, remboursement,
signalements) au lieu de deux copies. Migration append-only respectée : on
ne modifie pas `00008`, on reconstruit dans `00012`.

**Alternatives considered** :

- tables `event_online_payments` et `event_payment_alerts` parallèles :
  aucune migration délicate, mais deux recherches dans le webhook et le
  balayage, deux jeux de requêtes et le risque qu'une correction ne soit
  appliquée qu'à l'un des deux circuits ;
- `ALTER TABLE ADD COLUMN registration_id` sans reconstruire :
  `membership_id` resterait `NOT NULL` ; impossible ;
- modifier `00008` : interdit, migrations publiées.

## R3. Domaine pur `internal/event`, paiement réutilisé de `internal/membership`

**Decision** : nouveau paquet sans base, comme `internal/membership` :

- `Event` (titre, lieu, description, début, fin, ouverture, capacité, prix,
  période) et `ValidateEvent(in, seasons)` : bornes de FR-002, fin après
  début, clôture des inscriptions au plus tard à la fin, saison contenant le
  début (FR-003), heures en `season.Paris` ;
- `Status` (`draft`, `published`, `cancelled`) et `Phase(ev, now)`
  (`draft`, `upcoming`, `finished`, `cancelled`) : « Terminé » est calculé,
  jamais stocké (FR-005) ;
- `Openness` (`season_members`, `all_members`) et `Eligible(openness,
  hasValidatedMembership)` avec la raison d'inéligibilité ;
- `RegistrationOpen(ev, now)`, `CanRegister`, `CanUnregister` (FR-013),
  `CanAdminRegister`, `CanRemove`, `CanCancelEvent`, `CanRestore`,
  `CanDelete`, `CapacityLeft` ;
- `PaymentSnapshot(reg, ev) membership.Snapshot` : **adaptateur** qui
  présente une inscription au domaine de paiement existant (`Status =
  Validated` si l'inscription est confirmée et l'événement non annulé,
  `Cancelled` sinon ; `SeasonClosed = false` ; montant dû ; moyen payé).

Ainsi `membership.CanPay`, `CanPayByCard`, `CanCancelPayment`,
`SettleOnline` et `RefundCancels` s'appliquent tels quels aux inscriptions :
un paiement par carte reçu sur une inscription annulée ou sur un événement
annulé produit le signalement `refund_due` (FR-028), un double paiement
`duplicate`, un montant différent `amount_mismatch`.

**Rationale** : aucune règle de paiement dupliquée ; le domaine de paiement
est déjà testé en table. L'adaptateur tient en quelques lignes et se teste
seul.

**Alternatives considered** : extraire `Method`, `PaymentStatus`,
`Settings`, `SettleOnline` dans un paquet `internal/payment` : plus propre
sur le nom, mais déplace du code stable dans une fonctionnalité qui n'en a
pas besoin. À reconsidérer si un troisième objet payable apparaît.

## R4. Paiement par carte : généraliser le circuit de 008 à une « cible »

**Decision** : dans `internal/api/card_payments.go`, la tentative porte une
**cible** (`membership_id` ou `registration_id`). Le cœur
(`verifyOnlinePayment`, `settleOnlinePayment`, `settleRefund`, balayage
`OnlinePaymentSweeper`, webhook) lit la tentative, puis charge la cible et
son `membership.Snapshot` via une petite interface interne :

```text
payTarget { snapshot(); recordCard(); cancelRefunded(); notifyPaid(); notifyCancelled(); logAttrs() }
```

implémentée pour l'adhésion (code actuel déplacé) et pour l'inscription.

- **Métadonnées HelloAsso** : `helloasso.Metadata` reçoit `Registration
  int64 \`json:"registration,omitempty"\``, et `Membership` devient
  `omitempty`. La vérification exige que la cible des métadonnées soit celle
  de la tentative (même contrôle qu'aujourd'hui). Les intentions déjà créées
  restent valides.
- **Intitulé** (FR-024) : `Événement <titre> — <référence>`, tronqué par
  `helloasso.TruncateItemName`, sans nom du membre.
- **Adresses de retour** : `/mes-inscriptions/{id}?tentative=…&carte=…`.
- **Opérations** : `start-my-registration-card-payment`,
  `confirm-my-registration-card-payment`,
  `verify-registration-card-payment` (admin), sur le modèle de 008.
- **Aucune nouvelle route publique** : la notification arrive sur l'adresse
  existante ; le webhook retrouve la tentative par `n.Payment` ou par le
  paiement HelloAsso, sans connaître la cible (principe II inchangé).
- `MinCardAmount` (1 €) s'applique : un prix inférieur n'est pas payable par
  carte, comme une adhésion.

**Rationale** : un seul chemin constate un paiement par carte (008 › R5) ;
la cible est le seul point de variation.

## R5. Références de paiement `EVT-AAAA-NNNN`

**Decision** : `event.ReferencePrefix(season)` = `EVT-` + années de début et
de fin de la saison de l'événement sur deux chiffres + `-` ; numérotation
par `membership.NextReference` sur les références existantes du préfixe
(table `event_registrations`), dans la transaction d'inscription. Unicité
garantie par `UNIQUE` sur la colonne ; préfixes `ADH-` et `EVT-` disjoints,
donc aucune collision entre tables.

**Rationale** : même forme que les adhésions (006 › R4), lisible au dos d'un
chèque, distinguable au premier coup d'œil.

## R6. Capacité et concurrence

**Decision** : l'inscription s'exécute dans `WriteTx` (`BEGIN IMMEDIATE`) :
relecture de l'événement, `CountConfirmedRegistrations`, contrôle de la
capacité, insertion. Le pool d'écriture à une connexion et le verrou
immédiat sérialisent les écritures : deux inscriptions simultanées pour la
dernière place ne peuvent pas aboutir toutes deux (FR-015, SC-004). Une
inscription de plusieurs fiches en une demande est **tout ou rien** : si
les places restantes ne suffisent pas, rien n'est inscrit et le message
indique le nombre de places restantes.

Abaisser la capacité sous le nombre d'inscrits est refusé dans la même
transaction (FR-007).

**Alternatives considered** : trigger SQL qui refuse l'insertion au-delà de
la capacité : logique métier dispersée dans la base, message d'erreur peu
exploitable ; inutile avec un seul écrivain.

## R7. Éligibilité

**Decision** : `season_members` : la fiche a une adhésion `validated` à la
saison de l'événement (même définition que l'« adhérent d'une saison » de
009) ; `all_members` : toute fiche. Contrôlée par le serveur à chaque
inscription, par le membre comme par l'administrateur (FR-011). Les
inscriptions existantes ne sont jamais retirées automatiquement : la liste
des inscrits calcule `eligible` à la lecture et signale les inscrits devenus
inéligibles (Edge Cases).

## R8. Dates, heures et saison

**Decision** :

- l'API reçoit et renvoie des **dates-heures locales sans fuseau**
  (`2026-12-12T14:00`), interprétées en `season.Paris` (Europe/Paris) ; la
  base stocke l'instant UTC (`2026-12-12T13:00:00Z`) pour trier et comparer ;
- une heure locale inexistante (passage à l'heure d'été) est refusée avec un
  message sur le champ ; une heure ambiguë (passage à l'heure d'hiver) prend
  la première occurrence ;
- la saison est celle qui contient la **date locale** du début ; elle est
  recalculée à chaque modification des dates ;
- les saisons se suivent sans trou (003) : seul un début avant la première
  ou après la dernière saison est refusé ;
- **modification des dates d'une saison** (003) : refusée (`409`) si un
  événement de cette saison sortirait de ses nouvelles bornes, avec le
  nombre d'événements concernés ; **suppression d'une saison** : refusée si
  elle a des événements (`ON DELETE RESTRICT` + message explicite).

**Rationale** : des bénévoles saisissent « 14 h » ; ils ne doivent pas voir
de fuseau. Stocker l'UTC rend les comparaisons et le tri triviaux en SQL.

## R9. Annulation, rétablissement, suppression

**Decision** :

- annuler (`published` → `cancelled`) conserve les inscriptions et leur état
  de paiement ; `cancel_message` facultatif (≤ 1 000 caractères) ;
- tant qu'un événement est annulé, ses inscriptions ne sont pas payables
  (snapshot `Cancelled`, R3) et sortent de « À régler » ;
- rétablir (`cancelled` → `published`) avant la fin rend les inscriptions de
  nouveau payables, sans contrôle de capacité (les places étaient déjà
  prises) ;
- supprimer : seulement sans aucune ligne dans `event_registrations` ; le
  `DELETE` est conditionné en SQL (`NOT EXISTS`) pour éviter une course ;
- un brouillon n'a jamais d'inscription (inscription impossible avant
  publication) ; un brouillon peut donc toujours être supprimé.

## R10. Notifications et regroupement par adresse

**Decision** : six modèles d'e-mail (texte et HTML) dans
`internal/mail/templates` : `registration-confirmed` (avec montant,
référence et lien de paiement si payante), `registration-paid`,
`registration-removed`, `event-changed`, `event-cancelled`,
`event-restored`.

- **Adresse de contact** : celle de 009 (compte gestionnaire actif, sinon
  adresse de la fiche, sinon aucune). Elle diffère volontairement des e-mails
  d'adhésion de 006, envoyés au seul compte : un inscrit sans compte doit
  être prévenu de l'annulation d'un événement.
- **Regroupement** (FR-031) : pour `event-changed`, `event-cancelled` et
  `event-restored`, les inscriptions confirmées passent par
  `communication.BuildAudience` (009), qui choisit l'adresse de contact et
  regroupe par adresse sans casse ; un e-mail par adresse nomme chaque
  inscrit.
- **Changement notifié** : seulement si le début, la fin ou le lieu d'un
  événement publié et non terminé change ; titre, description et capacité ne
  déclenchent rien.
- Tous les e-mails sont insérés dans `outbound_emails` dans la transaction de
  l'action (constitution, « E-mails »). Au-delà de 500 destinataires, la
  file absorbe le volume comme pour 009.

## R11. « À régler » et « Mes inscriptions »

**Decision** :

- `list-my-dues` (`GET /me/dues`) : union, côté Go, des adhésions et des
  inscriptions des fiches du compte en « À payer » ou « Paiement annoncé »
  et payables (`CanPay`), triées par date de création ; chaque ligne porte
  `kind` (`membership` | `registration`), l'id, le membre, l'intitulé, le
  montant, l'état et le moyen annoncé. La page renvoie vers la page de
  paiement existante (`/adhesions/{id}`) ou nouvelle
  (`/mes-inscriptions/{id}`).
- `list-my-registrations` : inscriptions confirmées et annulées des fiches
  du compte, à venir puis passées.

## R12. Recherche des inscriptions (administration)

**Decision** : « Inscriptions » cherche par nom ou prénom via `members_fts`
existante (trigram) et par fragment de référence via une table
`event_registrations_fts` (`payment_reference`, trigram, contenu externe,
triggers), sur le modèle de `memberships_fts`. Aucune colonne repliée. Les
listes sont chargées entières puis paginées côté client comme les adhésions
(volume : quelques milliers d'inscriptions par saison au plus).

## R13. Export CSV

**Decision** : opération `export-event-registrations`
(`GET /admin/events/{id}/registrations.csv`) qui renvoie `text/csv;
charset=utf-8` avec BOM UTF-8 et séparateur `;` (ouverture directe dans un
tableur réglé en français), nom de fichier `inscrits-<id>.csv`. Colonnes de
la spec (US5 §6). Les cellules commençant par `=`, `+`, `-` ou `@` sont
préfixées d'une apostrophe (injection de formule). Le téléchargement passe
par le client `openapi-fetch` (`parseAs: "blob"`), sans `fetch` direct.

## R14. Fiche supprimée, compte détaché

**Decision** : `event_registrations.member_id … ON DELETE CASCADE` (et en
cascade les tentatives et signalements). Le dialogue de suppression d'une
fiche (005, admin et espace membre) reçoit `paidUpcomingRegistrations` dans
`get-member`/`get-my-member` et l'annonce ; la règle est la même que pour une
adhésion payée. Le rattachement à un compte n'est pas copié sur
l'inscription : le compte gestionnaire est toujours lu sur la fiche.

## R15. Journaux

**Decision** : `slog` en JSON, messages en français, attributs anglais :
`event_id`, `registration_id`, `online_payment_id`, `payment_reference`,
`actor_id`, `outcome`. Ni titre, ni lieu, ni description, ni nom de membre,
ni adresse électronique (FR-036).

## R16. Tests

**Decision** :

- `internal/event` : tests en table (validation, phase, période, éligibilité,
  capacité, actions, adaptateur de paiement, références) ;
- migration : base `00011` peuplée (adhésion payée par carte, signalement)
  → `00012` : données conservées, `foreign_key_check` vide, `foreign_keys`
  rétabli ; puis retour arrière ;
- API (`httptest`) : CRUD des événements, cycle de vie, inscriptions par le
  membre et par l'administrateur, capacité sous concurrence (50 goroutines
  sur 3 places), éligibilité, paiement manuel, carte via `hatest`
  (constatation, notification, remboursement, signalements sur inscription
  annulée), « À régler », export CSV, regroupement des e-mails, isolation
  entre comptes, brouillons invisibles ;
- matrice d'autorisation automatique étendue aux nouvelles opérations ;
- Vitest : formatage des dates locales, formulaire d'événement ;
- Playwright + axe : `events.spec.ts` (création, publication, inscription,
  paiement annoncé, annulation) sans violation `serious` ni `critical`.
