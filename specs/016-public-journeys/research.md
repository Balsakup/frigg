# Research: Parcours publics (adhésion et inscription sans compte)

**Feature**: `016-public-journeys` | **Date**: 2026-11-17 | **Spec**: [spec.md](./spec.md)

Les trois points ouverts de la spécification ont été tranchés le 2026-11-17 :
rattachement automatique à un compte existant (FR-033), lien personnel pour
toute fiche sans compte avec adresse (FR-040), signalement des fiches
publiques inactives sans suppression automatique (FR-041). Les choix
ci-dessous partent du code existant (001 jetons et sessions, 005 fiches, 006
adhésions, 008 carte, 011 champs, 012 événements, 014 montants) et de la
constitution **v1.9.0**, amendée pour cette fonctionnalité.

## R1. Accès par lien personnel : un niveau d'accès `Link` adossé à la session

**Decision** :

- `internal/auth` gagne un quatrième niveau d'accès, `Link`, hors de la
  hiérarchie des rôles : `Allows(role, Link)` est faux pour tout rôle. Seul
  `accessMiddleware` l'accorde, quand la session porte un accès par lien
  valide.
- Le jeton du lien n'apparaît jamais dans une URL envoyée au serveur. L'e-mail
  pointe vers `/suivi#jeton=…` (fragment). Le SPA lit le fragment, l'efface de
  l'historique (`history.replaceState`), puis appelle `POST
  /api/public/tracking/open` avec le jeton dans le corps.
- `open` vérifie l'empreinte, l'expiration et la révocation, renouvelle le
  jeton de session (`RenewToken`) et range dans la session scs existante
  `tracking_link_id`. Aucun compte n'est mis en session : une personne
  connectée garde sa session de compte à côté, sans effet sur le périmètre du
  lien.
- À chaque requête `Link`, le middleware relit la ligne `tracking_links`
  (non révoquée, non expirée) et place dans le contexte l'adresse du lien.
  Les requêtes `…OfTrackedEmail` filtrent sur
  `members.account_id IS NULL AND members.email = ? COLLATE NOCASE`. Le
  périmètre est donc réévalué à chaque accès (FR-014).
- Retours de HelloAsso : ils arrivent par navigation de premier niveau
  (`GET`). Le cookie `SameSite=Lax` est donc envoyé, et la page de suivi
  reprend sans nouveau jeton.

**Rationale** :

- Le fragment n'apparaît ni dans les journaux d'accès, ni dans `Referer`, ni
  dans les journaux d'un proxy.
- La session `HttpOnly` évite de garder un jeton en mémoire JavaScript, et
  `CrossOriginProtection` couvre déjà les `POST`.
- Un seul point de contrôle, le middleware, suit le principe III : pas de
  vérification ad hoc dans les handlers.

**Alternatives considered** :

- Jeton dans le chemin ou la requête de chaque appel d'API : il fuit dans les
  journaux et dans `Referer`, et oblige chaque handler à le relire.
- Session dédiée avec un second cookie : un second `SessionManager` pour un
  gain nul, puisque la clé `tracking_link_id` suffit.
- Compte « invité » créé à la volée : interdit par le principe III, qui ne
  connaît que deux rôles.

## R2. Jetons : confirmation à usage unique, lien personnel réutilisable

**Decision** : `auth.NewToken` (256 bits, empreinte SHA-256) sert pour les
deux jetons. Les valeurs sont fixes, dans un fichier `internal/public/limits.go`
(FR-009, FR-013) :

| Jeton | Durée | Usage | Révocation |
| --- | --- | --- | --- |
| Confirmation de dépôt | 48 h | unique : le dépôt est supprimé à la confirmation | — |
| Lien personnel | 30 j | réutilisable, plusieurs par adresse | à la création d'un compte pour l'adresse ; purge 24 h après l'expiration |

Chaque e-mail de notification qui vise une fiche suivie par lien crée son
propre lien, dans la transaction qui met l'e-mail en file (FR-035). Les
anciens liens restent valides : les liens des e-mails précédents continuent de
fonctionner (Assumptions de la spec).

**Rationale** : même génération et même stockage haché que les jetons de 001
(principe VII). La réutilisation du lien personnel est explicitement admise
par l'amendement 1.9.0.

**Alternatives considered** : lien à usage unique, renouvelé à chaque
ouverture. Il casse le bouton « retour » et les e-mails antérieurs, sans gain
réel puisque la session porte déjà l'accès après la première ouverture.

## R3. Dépôts : table `public_submissions`, purge par balayage

**Decision** :

- Un dépôt est une ligne de `public_submissions` : nature, cible, identité,
  coordonnées, choix « Créer un compte », date de consentement, empreinte du
  jeton de confirmation, expiration.
- Les valeurs des champs personnalisés sont validées au dépôt par
  `customfield.Apply` (audience `Owner`) et rangées en JSON canonique dans
  `custom_values`. À la confirmation, elles sont réappliquées avec les
  définitions du moment : un champ archivé est ignoré, un champ ajouté n'est
  pas exigé (Edge Cases).
- Un balayeur `PublicSweeper`, sur le modèle de `OnlinePaymentSweeper`, tourne
  toutes les 10 minutes. Il supprime les dépôts de plus de 7 jours et les
  liens personnels expirés depuis plus de 24 heures.

**Rationale** :

- Un dépôt n'a aucune existence métier : il n'apparaît dans aucune requête de
  fiches, d'adhésions ou d'inscriptions (FR-008).
- Le JSON n'est ni interrogé ni filtré. Il transporte des valeurs déjà
  normalisées par le domaine, sans dupliquer de règle (principe VI).

**Alternatives considered** : créer tout de suite une fiche et une adhésion
avec un état « non confirmé ». Cela oblige à les filtrer dans des dizaines de
requêtes existantes, avec un risque de fuite vers l'administration ou de
place réservée.

## R4. Origine « Publique » : colonnes ajoutées, sans reconstruction

**Decision** : migration `00014_public_journeys.sql`, uniquement en
`ALTER TABLE … ADD COLUMN`, `CREATE TABLE` et `CREATE INDEX`
([data-model.md](./data-model.md)) :

- `members.origin` (`'internal' | 'public'`, défaut `'internal'`),
  `members.email_confirmed_at`, `members.consented_at` ;
- `memberships.via_public` et `event_registrations.via_public` (0/1). Une
  ligne `via_public = 1` a `requested_by` (ou `registered_by`) et `created_by`
  à `NULL` ;
- `online_payments.via_link` (0/1) : tentative lancée par lien personnel,
  avec `account_id` à `NULL`.

Les lectures qui déduisent aujourd'hui l'auteur de `requested_by` et
`created_by` (« demandée par », « créée par ») traitent d'abord `via_public`.
Libellé affiché : « Demande publique (adresse confirmée) ».

**Rationale** : la colonne explicite évite qu'une demande publique
(`requested_by` à `NULL`) passe pour une adhésion créée par un administrateur
dont le compte a été supprimé (`ON DELETE SET NULL`).

**Alternatives considered** : une table `public_origins` en relation 1-1.
Elle impose une jointure dans toutes les listes pour un simple drapeau.

**À sonder en tâche 1** : `ADD COLUMN` avec un `CHECK` qui référence une autre
colonne. SQLite l'accepte et l'éprouve sur les lignes existantes ; à vérifier
avec le pilote modernc et sqlc, comme la sonde de 012 › R2.

## R5. Événements « Ouvert au public » : lien public sans reconstruire `events`

**Decision** :

- `events.openness` garde son `CHECK` (`season_members | all_members`).
  `events.public_slug` (texte, nullable, index unique partiel) porte le lien
  public.
- Le domaine `internal/event` expose `Openness` à trois valeurs. `Public`
  équivaut à `openness = 'all_members' AND public_slug IS NOT NULL`. L'API
  expose l'enum `season_members | all_members | public` (FR-023).
- Le slug compte 128 bits, en base64url. Il est stocké en clair : il ne
  protège aucune donnée personnelle et l'administrateur doit pouvoir le
  réafficher. « Régénérer » remplace le slug (FR-024). Quitter « Ouvert au
  public » le met à `NULL`.

**Rationale** : changer le `CHECK` d'`openness` exige de reconstruire
`events`, que référence `event_registrations` (`ON DELETE RESTRICT`), dans une
migration `NO TRANSACTION`. C'est un risque de migration pour une valeur
d'énumération. L'invariant reste garanti par
`CHECK (public_slug IS NULL OR openness = 'all_members')`.

**Alternatives considered** :

- Reconstruction de `events` (modèle de 012 › R2) : sûre mais lourde, sans
  bénéfice fonctionnel.
- Slug haché : impossible à réafficher, il faudrait le régénérer à chaque
  consultation de l'administration.

## R6. Paramètre d'accueil des demandes publiques

**Decision** : `association.public_membership_enabled` (0/1, défaut 0). Il
est exposé dans `get/update-general-settings` (section « Adhésions
publiques » de la page Paramètres › Général), avec l'adresse de la page
calculée à partir de `FRIGG_BASE_URL` : `<base>/adherer` (FR-004, SC-010).

**Rationale** : la table singleton `association` porte déjà les paramètres
généraux (010), et le défaut 0 est appliqué aux instances existantes par la
migration.

## R7. Confirmation : une transaction qui revérifie, déduplique et rattache

**Decision** : `POST /api/public/submissions/confirm` fait, en `WriteTx`
(`BEGIN IMMEDIATE`) :

1. Il lit le dépôt par empreinte. S'il est absent, utilisé ou expiré, il
   répond `410` avec un message unique.
2. Il revérifie les conditions (FR-010) : accueil activé, saison ouverte et
   non close, tarif existant ; ou événement public publié, dans sa période et
   non complet. La capacité est comptée dans la transaction, comme pour 012
   FR-015.
3. Il cherche le compte par adresse (`accounts.email`, même normalisation que
   la connexion). Si le compte est actif et gère moins de 20 fiches, la fiche
   lui est rattachée (FR-033). Sinon, elle reste « Sans compte ».
4. Il déduplique par `member.IdentityKey` et la date de naissance, parmi les
   fiches suivies par lien de l'adresse ou parmi les fiches du compte
   (FR-011).
5. Il crée la fiche, sauf réutilisation, puis l'adhésion en attente ou
   l'inscription confirmée (prix figé, référence `EVT-`). Il applique les
   champs personnalisés et supprime le dépôt.
6. Il met en file l'e-mail : lien personnel neuf (fiche sans compte) ou
   invitation à se connecter (fiche rattachée). Si la fiche reste sans
   compte, il ouvre aussi l'accès par lien dans la session, comme `open`.

La réponse `ConfirmOutcome` indique `tracking` (suivi par lien),
`account` (rattachée : se connecter), `duplicate` (déjà existante : renvoi
au suivi) ou `closed` (avec la raison).

**Rationale** : un seul passage atomique. Deux confirmations simultanées pour
la dernière place se sérialisent sur l'écrivain unique (SC-006).

**FR-033 et la non-révélation** : la différence de réponse n'apparaît qu'à
la confirmation, donc à la personne qui détient la messagerie. Le dépôt
répond, lui, toujours `202` avec le même corps (SC-005).

**E-mail de confirmation du dépôt** : son contenu dépend de l'état du compte
au moment du dépôt, puisque seule la personne qui détient la messagerie le
lit :

- compte actif qui gère moins de 20 fiches → `AccountExists` : « après
  confirmation, votre demande apparaîtra dans votre espace membre » ;
- compte actif qui gère déjà 20 fiches → `AccountFull` : « votre compte gère
  déjà 20 fiches : cette demande sera suivie par un lien personnel » ;
- compte désactivé ou aucun compte → texte standard, sans mention de compte.

L'état peut changer entre le dépôt et la confirmation. C'est l'écran de
confirmation (`ConfirmOutcome`) qui fait foi.

## R8. Notifications : destinataire unifié et lien personnel dans l'e-mail

**Decision** : `noticeRecipient` (adhésions) et `contactEmail`
(inscriptions) cèdent la place à une seule fonction pure,
`recipientOf(c noticeContact, disabledFallback bool) (to string, tracked bool)`
(`internal/api/recipients.go`), où `noticeContact` porte le compte
gestionnaire, son état et l'adresse de la fiche. L'émission du lien personnel
est à part, dans `noticePath`, appelée dans la transaction qui met l'e-mail
en file :

- compte actif qui gère la fiche → adresse du compte, lien vers l'espace
  membre (inchangé) ;
- fiche sans compte avec adresse → adresse de la fiche, avec un lien
  personnel neuf vers `/suivi…#jeton=…` (FR-035, FR-040) ;
- fiche dont le compte est désactivé → aucun e-mail pour une adhésion
  (006 FR-023 inchangé), adresse de la fiche sans lien personnel pour une
  inscription (012 inchangé). La fonction reçoit ce choix en paramètre
  (`disabledFallback`) : l'unification ne modifie aucun comportement existant
  hors FR-035 ;
- sinon, aucun e-mail.

La règle des fiches sans compte (FR-035, FR-040) est désormais écrite une
seule fois.

`enqueueNotice` accepte un chemin déjà muni de son fragment.
`mail.Data.Link` reste la seule variable de lien des modèles, et les modèles
existants ne changent pas de structure. Nouveaux modèles :
`public-confirm` (variantes « compte existant » et « compte complet »,
voir R7), `tracking-link`,
`public-registration-confirmed` (avec le lien personnel).

**Rationale** : 006 FR-023 et 012 FR-030 utilisent la même règle. La
consommation de liens est minime (une ligne par e-mail).

**Point de vigilance** : le corps des e-mails envoyés, qui contient le jeton,
reste dans `outbound_emails` jusqu'à la purge des e-mails envoyés. C'est déjà
le cas pour les invitations et les réinitialisations. La durée de 30 jours
du lien est bornée par cette purge si elle est plus courte, et le plan ne la
modifie pas.

## R9. Limitation de débit des parcours publics

**Decision** : neuf limiteurs `KeyedLimiter` ajoutés à `auth.Limits`, purgés
comme les autres. Chaque parcours public est limité **par IP et par adresse
visée**, comme l'exige le principe VII (FR-028, SC-008) :

| Parcours | Limiteur par IP | Limiteur par adresse |
| --- | --- | --- |
| Dépôt | `PublicSubmitIP` : 10 / heure | `PublicMailEmail` : 5 / heure (adresse saisie) |
| Demande de lien | `TrackingRequestIP` : 10 / heure | `PublicMailEmail` (adresse saisie) |
| Lien envoyé par un administrateur | — (opération `admin`) | `PublicMailEmail` (adresse de la fiche) |
| Confirmation d'un dépôt | `PublicConfirmIP` : 20 / heure | `PublicConfirmEmail` : 10 / heure (adresse du dépôt) |
| Ouverture d'un lien | `TrackingOpenIP` : 30 / heure | `TrackingOpenEmail` : 30 / heure (adresse du lien) |
| Opérations `link` | `TrackingUseIP` : 120 / minute | `TrackingUseEmail` : 120 / minute (adresse du lien) |

`PublicMailEmail` est partagé par tous les e-mails de confirmation et de lien
personnel, y compris ceux qu'envoie un administrateur : une adresse n'en
reçoit jamais plus de 5 par heure.

L'adresse d'une confirmation ou d'un lien n'est connue qu'après la lecture du
jeton. Le limiteur par adresse est donc consommé juste après cette lecture,
et avant toute écriture.

Réponses en cas de dépassement :

- `PublicMailEmail` au dépôt ou à la demande de lien : la réponse `202`
  neutre est conservée, et le dépôt ou la demande est ignoré, sans e-mail ;
- `PublicMailEmail` pour un envoi par un administrateur : `429` (« Trop
  d'envois vers cette adresse, réessayez plus tard. »), sans risque de
  révélation puisque l'administrateur connaît la fiche ;
- tout autre dépassement : `429` avec le message neutre de 001.

**Rationale** : `PublicMailEmail` empêche d'utiliser Frigg pour inonder une
messagerie (Edge Cases). Les limites par IP freinent l'énumération et les
robots. Les limites par adresse bornent l'usage d'un lien ou d'une
confirmation qui aurait fuité, même réparti sur plusieurs IP.

## R10. Piège à robots et formulaires publics

**Decision** : chaque formulaire de dépôt porte un champ `website`, rendu
dans un conteneur `aria-hidden="true"` hors écran, avec `tabindex="-1"` et
`autocomplete="off"`. Côté serveur, une valeur non vide renvoie la réponse
`202` d'un dépôt accepté, sans écriture ni e-mail. L'événement est journalisé
avec le motif `honeypot` (FR-029, FR-042).

**Rationale** : invisible au clavier et au lecteur d'écran (contrôlé par axe
et par un test au clavier), sans service tiers (FR-030, principe VII).

**Alternatives considered** : délai minimal de saisie. Il gêne les
gestionnaires de mots de passe et l'auto-complétion, pour un gain marginal.

## R11. Paiement par lien : circuit de carte paramétré par le payeur

**Decision** :

- `openCheckout(ctx, opID, base, req, payer)` reçoit un `helloasso.Payer`
  explicite au lieu de lire `CurrentAccount`.
  - Côté compte : adresse du compte, fiche « Vous » (inchangé).
  - Côté lien : adresse du lien, prénom et nom de la fiche concernée (FR-020).
- Les tentatives par lien ont `account_id` à `NULL` et `via_link = 1`. Les
  adresses de retour sont `/suivi/adhesions/{id}?tentative=…&carte=` et
  `/suivi/inscriptions/{id}?…`.
- `verifyOnlinePayment` reste l'unique chemin de constatation. Il n'est pas
  modifié (FR-021).
- L'annonce d'un moyen manuel réutilise `membership.CanAnnounce` et les mêmes
  requêtes, avec un filtre de périmètre `…OfTrackedEmail`.
- Les handlers « par lien » appellent les mêmes fonctions internes que ceux de
  l'espace membre (`announcePayment`, `startCard`, `confirmCard`). Seules la
  résolution de la cible et celle du payeur changent.

**Rationale** : aucune règle de paiement dupliquée. Les garanties de 008 sont
conservées par construction.

## R12. Création de compte depuis un lien

**Decision** : `POST /api/tracking/account` (niveau `Link`) fait, en
`WriteTx` :

- il refuse si un compte existe déjà pour l'adresse, avec le message neutre
  « Connectez-vous avec cette adresse » (l'appelant détient la messagerie) ;
- il refuse si le lien suit plus de 20 fiches (FR-031) ;
- il crée le compte : rôle `member`, `email_verified_at` à maintenant, mot de
  passe vérifié par `auth.ValidatePassword` (001 FR-008) ;
- il rattache les fiches suivies par lien et pose `is_self` selon le choix
  (FR-032) ;
- il révoque tous les liens de l'adresse et supprime l'accès par lien de la
  session ;
- enfin, `auth.Login`.

La case « Créer un compte » du dépôt mène simplement à cet écran après la
confirmation.

## R13. Rattacher une demande publique à une fiche existante

**Decision** : `POST /api/admin/memberships/{id}/attach` (admin), avec
`{memberId}` :

- il n'accepte qu'une demande `via_public` en attente ;
- il déplace `member_id` et laisse l'index `memberships_one_active` refuser
  un doublon (`409`, message de 006 FR-003) ;
- il copie l'adresse confirmée sur la fiche cible si celle-ci n'en a pas ;
- il copie les valeurs des champs de membre là où la cible n'en a pas ;
- il supprime la fiche publique si elle ne porte plus rien, dans la même
  transaction (FR-038).

Le signalement de doublon réutilise la recherche de 005 FR-005
(`members_birth_date` et `IdentityKey`).

## R14. Fiches publiques inactives

**Decision** : une requête sqlc statique, `ListInactivePublicMembers`, qui
renvoie les identifiants. La coupure (aujourd'hui moins 12 mois,
`public.InactiveCutoff`) est passée en deux paramètres : `cutoff_on`, le jour
(`AAAA-MM-JJ`, comparé à `seasons.ends_on`), et `cutoff_at`, ce jour à minuit
à Paris en UTC (comparé à `members.created_at` et `events.ends_at`).
`public.Inactive` reprend la même règle en Go pur. Les conditions
sont celles du Vocabulaire de la spec : `origin = 'public'`, sans compte,
créée avant la coupure, aucune demande en attente, aucune adhésion validée
d'une saison terminée après la coupure, aucune inscription confirmée à un
événement terminé après la coupure.

Cette requête est exposée par :

- le filtre `inactive=true` de `list-members` ;
- `inactivePublicCount` dans `get-membership-dashboard`, compté en Go à partir
  de la même liste (pas de requête de comptage distincte) ;
- `POST /api/admin/members/inactive/delete`, avec une liste d'identifiants
  revérifiée en transaction. Les fiches devenues actives entre-temps sont
  ignorées et comptées dans la réponse.

Aucun balayeur ne supprime ces fiches (FR-041, SC-013).

## R15. Écrans publics : un layout « public » commun

**Decision** : `routes/public/layout.tsx` reprend l'habillage des écrans
d'authentification : identité visuelle par `get-identity`, carte centrée,
pied de page. Il héberge `/adherer`, `/e/:slug`, `/confirmer-demande`,
`/suivi`, `/suivi/lien`, `/suivi/adhesions/:id`, `/suivi/inscriptions/:id` et
`/suivi/compte` ([ui-routes.md](./contracts/ui-routes.md)).

- Les composants de paiement de `payments/` (extraits en 012) sont réutilisés
  avec un client « par lien ».
- La détection d'une session de compte (bandeau « Vous êtes connecté »,
  FR-001 US1-9) passe par `get-session`. Un `401` y est attendu et
  silencieux.

**Rationale** : principe IV (un shell, un design system), et pas de nouveau
gabarit.

## R16. Journaux

**Decision** : `slog` avec `action` parmi `public_submit`,
`public_submit_rejected` (motif `rate_ip | rate_email | honeypot | closed |
invalid`, et pour une inscription `not_found` ou la raison de fermeture de
l'événement), `public_confirm`, `tracking_link_issued`, `tracking_link_opened`,
`tracking_account_created`, `membership_attached`, `event_public_link_regenerated`
et `inactive_members_deleted` (`actor_id`, `deleted`, `skipped`), plus
`tracking_link_request` (`outcome`), `tracking_link_sent` (envoi par un
administrateur) et `public_purge` (balayeur). `tracking_account_created` ne
porte que `account_id`. Les seuls attributs sont des identifiants
(dépôt, fiche, adhésion, inscription, lien, événement) et des compteurs :
jamais d'adresse, de nom ni de jeton (FR-042).

**Journal des requêtes** : le slug d'un lien public est un secret. Le
middleware `requestLogger` le masque (`redactPath` de
`internal/server/middleware.go`) pour `/api/public/events/{slug}…` et pour la
page `/e/{slug}`, comme la clé de l'adresse de notification HelloAsso. Le
jeton personnel ne parvient jamais au serveur dans l'adresse : il voyage dans
le fragment `#jeton=` puis dans le corps des requêtes. `public_logs_test.go`
vérifie ces deux points.

## R17. Tests

**Decision** :

- **Matrice d'accès** (`access_test.go`) :
  - `allowedPublic` gagne les 7 opérations publiques, avec un commentaire
    renvoyant à l'amendement 1.9.0 ;
  - nouvelle branche `Link` : anonyme → 401, membre et administrateur sans
    lien → 401, client muni d'un lien valide → jamais 401 ni 403 ;
  - lien expiré ou révoqué → 401.
- **Isolation** (`tracking_isolation_test.go`) : un lien de A ne voit ni les
  fiches de B, ni une fiche de A rattachée à un compte, ni une fiche dont
  l'adresse a changé. La réponse est `404` identique à une donnée inexistante.
- **Non-révélation** : pour une adresse inconnue, une adresse de compte et une
  adresse de fiche, les réponses de dépôt et de demande de lien sont
  identiques octet pour octet (statut, corps, en-têtes hors date).
- **Concurrence** : 50 confirmations simultanées sur 3 places → 3
  inscriptions.
- **Limites** : 6 dépôts pour la même adresse en une heure → 5 e-mails.
- **Migration** : `00014` sur une base `00013` peuplée, avec données
  conservées et défauts corrects. Test de la sonde de R4.
- **Carte par lien** avec `hatest` : constatation par retour et par
  notification, et `account_id` à `NULL` sur la tentative.
- **Playwright + axe** : `public-journeys.spec.ts`, de bout en bout avec
  Mailpit (dépôt, e-mail, confirmation, suivi, paiement de test, création
  de compte), sans violation `serious` ni `critical`. Le piège est
  inaccessible au clavier.

## R18. Dépendances

**Decision** : aucune nouvelle dépendance Go ni npm. Tout repose sur
`auth.NewToken`, `KeyedLimiter`, scs, huma, sqlc, goose, base-ui et
react-hook-form existants (principe V).
