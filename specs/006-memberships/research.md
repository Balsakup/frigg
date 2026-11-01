# Recherche : Adhésions

Décisions techniques de la fonctionnalité 006. Elles prolongent celles de
001 à 005, qui restent valables : `register` et niveau d'accès par opération,
écritures sérialisées par `Store.WriteTx` (une connexion, `BEGIN IMMEDIATE`),
horloge injectable `Deps.Now` (date du jour à Paris par `s.today()`), état
calculé des saisons (`season.StateOf`), recherche FTS5 trigram
(`members_fts`, `FTSTrigramQuery`), tri replié en Go (`textfold`), erreurs
`422` cumulées, actions autorisées calculées par le serveur, cloisonnement
par compte dans les requêtes SQL (`…OfAccount`), e-mails en file
(`outbound_emails`) dans la transaction de l'action, journalisation `INFO`
sans données personnelles.

## R1. Stockage d'une adhésion et de son paiement

- **Décision** : une table `memberships` (migration `00008_memberships.sql`)
  porte l'adhésion et son paiement, relation 1-1 :
  - rattachements : `member_id` (`ON DELETE CASCADE`), `season_id`
    (`ON DELETE RESTRICT`), `fee_id` (`ON DELETE RESTRICT`) ;
  - cycle de vie : `status` (`pending`, `validated`, `refused`,
    `cancelled`), `declared_kind` (`first`, `renewal`, `NULL` si créée par un
    administrateur), `kind` et `amount_cents` (renseignés à la validation),
    `reason` (refus ou annulation) ;
  - auteurs et dates : `requested_by` (compte demandeur), `created_by`
    (administrateur créateur), `decided_by`, `cancelled_by`, `paid_by`
    (`ON DELETE SET NULL`), `requested_at`, `decided_at`, `cancelled_at` ;
  - paiement : `payment_reference` (`UNIQUE`), `payment_status` (`due`,
    `announced`, `paid`, `NULL` = pas de paiement), `announced_method`,
    `paid_method`, `paid_on`, `payment_note`, `paid_at`.
  Index unique partiel `memberships_one_active` sur `(member_id, season_id)
  WHERE status IN ('pending', 'validated')`.
- **Rationale** : l'index partiel garantit FR-003 en base, y compris pour
  deux demandes simultanées (SC-003) ; la violation est traduite en `409`.
  Une table de paiement séparée n'apporterait rien : un seul paiement par
  adhésion, sans paiement partiel ni échéancier (hypothèses de la spec). Les
  `CHECK` croisés (voir data-model) rendent impossibles les combinaisons
  incohérentes (adhésion en attente avec montant, paiement sans référence…).
- **Suppressions** : `CASCADE` sur la fiche (FR-021 : l'administrateur
  supprime la fiche et ses adhésions) ; `RESTRICT` sur le tarif et la saison
  (filets de sécurité derrière les contrôles explicites de R9 ; une saison
  qui porte des adhésions a forcément été ouverte et n'est donc plus
  supprimable, 003 › FR-019) ; `SET NULL` sur les comptes auteurs (la
  suppression d'un compte, 005, ne doit pas emporter l'historique des
  adhésions conservées).
- **Alternatives écartées** : table `payments` séparée (jointure
  systématique pour un 1-1) ; historique des états dans une table
  d'événements (pas de journal consultable demandé, spec › Historique).

## R2. Domaine pur `internal/membership`

- **Décision** : nouveau package sans dépendance à la base, comme
  `internal/season` et `internal/member` :
  - `Status`, `Kind`, `PaymentStatus`, `Method` et leurs constantes ;
  - `AppliedAmount(fee, kind)` : montant de renouvellement si le tarif en a
    un et que le type est `renewal`, montant principal sinon ;
  - `ActionsFor(m, seasonState, space)` : actions autorisées (annuler la
    demande, payer, valider, refuser, corriger, annuler l'adhésion,
    constater le paiement, annuler la constatation) selon l'état, l'état du
    paiement et l'état de la saison (FR-009, FR-016, FR-017) ;
  - `PreviousSeasonHint(...)` (R6) ;
  - `NextReference(prefix, last)` et `ReferencePrefix(season)` (R4) ;
  - `ValidatePaymentSettings` : règles de FR-022g, IBAN et BIC (R7) ;
  - `Stats` : agrégats d'une saison (R10).
- **Rationale** : toutes les règles d'état sont testables en table sans
  base ; les handlers ne font que lire, appeler le domaine et écrire.

## R3. Transitions et concurrence

- **Décision** : chaque écriture relit l'adhésion dans la transaction et
  vérifie la transition avec le domaine ; une adhésion dont l'état ne permet
  plus l'action donne `409` « Cette adhésion a changé entre-temps. »
  (FR-024, US2 §6). Transitions :

  ```text
  (demande) ──► pending ──validate──► validated ──cancel(admin)──► cancelled
                  │  │
                  │  └──refuse──► refused
                  └──cancel(compte)──► cancelled
  (création admin) ─────────────────► validated
  ```

  Paiement d'une adhésion `validated` dont `amount_cents > 0` :
  `due ⇄ announced` (choix ou changement de moyen par le compte),
  `due|announced ──record──► paid ──cancel-payment──► announced` si un moyen
  avait été annoncé, `due` sinon (US4 §10).
- **Rationale** : les écritures sont sérialisées ; la relecture dans la
  transaction suffit, sans jeton de version. La correction (FR-016) et
  l'annulation administrateur ne sont admises que si la saison n'est pas
  close ; validation, refus et constatation du paiement restent possibles
  sur une saison close (FR-017).
- **Alternatives écartées** : `expectedStatus` envoyé par le client (le
  statut relu suffit : toute transition invalide est refusée de la même
  façon).

## R4. Référence de paiement

- **Décision** : `ADH-{AA}{AA}-{NNNN}` : années de début et de fin de la
  saison sur deux chiffres (`ADH-2627-0042`, `ADH-2727-0003` pour une saison
  civile), puis un numéro séquentiel par préfixe, sur 4 chiffres au moins
  (14 caractères au-delà de 9 999). Attribuée à la première validation (ou
  correction) donnant un montant non nul, jamais modifiée ensuite, même si
  le montant revient à 0 (le paiement disparaît, la référence reste). Le
  numéro suivant est lu dans la transaction (`MAX` sur les références du
  préfixe) ; l'index `UNIQUE` fait foi.
- **Rationale** : courte, lisible au téléphone, sans donnée personnelle,
  au plus 16 caractères (spec › cas limites) ; le préfixe de saison aide le
  trésorier à rapprocher un relevé bancaire. Le numéro par préfixe (et non
  par saison) évite la collision de deux saisons de même préfixe (deux
  saisons civiles la même année). La recherche par référence (FR-018) passe
  par une table FTS5 trigram `memberships_fts` (principe VI : une référence
  est un identifiant, champ d'identité court), voir R10.
- **Alternatives écartées** : identifiant de l'adhésion (révèle le volume et
  ne parle pas au trésorier) ; code aléatoire (illisible, risque de
  confusion de caractères).

## R5. Opérations d'API

- **Décision** : deux groupes de chemins, comme 005 › R5.
  - Espace membre (tag `Mes adhésions`, accès `member`), sous
    `/me/memberships` : `get-membership-offer`, `list-my-memberships`,
    `get-my-membership`, `create-my-membership`, `cancel-my-membership`,
    `announce-my-payment`.
  - Administration (tag `Adhésions`, accès `admin`), sous
    `/admin/memberships` : `list-memberships`, `get-membership-stats`,
    `get-membership-dashboard`, `get-membership`, `create-membership`,
    `validate-membership`, `refuse-membership`, `update-membership`,
    `cancel-membership`, `record-membership-payment`,
    `cancel-membership-payment`.
  - Paramètres (tag `Paramètres`, accès `admin`) :
    `get-payment-settings`, `update-payment-settings` sous
    `/admin/settings/payment`.
  - Les transitions sont des `POST` sur des sous-ressources nommées
    (`/validate`, `/refuse`, `/cancel`, `/payment`) ; la correction est un
    `PUT` complet (tarif et type).
- **Rationale** : un verbe par transition rend chaque opération et son
  journal explicites ; le niveau d'accès est déclaré par opération
  (principe III). Les opérations membre ne reçoivent jamais d'identifiant de
  compte : toutes leurs requêtes sont conditionnées par le compte de session
  via la fiche (`members.account_id = ?`), et une adhésion d'une fiche gérée
  par un autre compte donne le `404` d'une adhésion inexistante (FR-008,
  US3 §4, US4 §12).
- **Offre** : `get-membership-offer` renvoie les saisons ouvertes, non
  closes, portant au moins un tarif, avec leurs tarifs, et pour chaque fiche
  du compte les saisons où elle peut encore adhérer. Le formulaire n'a ainsi
  aucune règle à recalculer (US1 §5, §7, §8, §9).

## R6. Aide « Adhérent la saison précédente dans Frigg »

- **Décision** : la saison précédente est celle dont `ends_on` est la
  veille du `starts_on` de la saison de l'adhésion (suite continue, 003 ›
  FR-006). `yes` si le membre y a une adhésion `validated` ; `no` si elle
  existe et compte au moins une adhésion `validated` ; `unknown` sinon.
  Calculée à la lecture de `get-membership` (deux requêtes indexées), jamais
  stockée.
- **Rationale** : « au moins une adhésion validée » distingue une saison
  gérée dans Frigg d'une saison antérieure à son usage (spec › cas limites) ;
  l'aide reste indicative (le type est choisi par l'administrateur).

## R7. Paramètres des moyens de paiement

- **Décision** : table singleton `payment_settings` (`id = 1`, ligne créée
  par la migration), une colonne par information : `cash_enabled`,
  `cash_instructions`, `cheque_enabled`, `cheque_payee`, `cheque_address`,
  `cheque_instructions`, `transfer_enabled`, `transfer_holder`,
  `transfer_iban`, `transfer_bic`, `transfer_instructions`, `updated_at`.
  `update-payment-settings` est un `PUT` complet ; les champs d'un moyen
  désactivé sont conservés (FR-022h) mais ne sont validés que s'il est
  activé (seule leur longueur est toujours contrôlée).
  - IBAN : espaces retirés, majuscules ; 15 à 34 caractères, deux lettres de
    pays, clé ISO 13616 vérifiée (modulo 97) ; affiché par groupes de 4.
  - BIC : facultatif ; 8 ou 11 caractères `AAAABBCC(DDD)`.
  - Textes d'instructions et adresse : texte brut multiligne (1 000 et
    300 caractères au plus), affiché avec retours à la ligne conservés,
    jamais interprété comme HTML.
- **Rationale** : données singleton de l'instance (principe I) ; une ligne
  à colonnes typées permet des `CHECK` en base et des types huma explicites,
  contrairement à une table clé-valeur. Le modulo 97 tient en quelques
  lignes, sans dépendance.
- **Exposition** : seules les informations des moyens activés sont renvoyées
  aux comptes, et seulement dans `get-my-membership` d'une adhésion à payer ;
  la lecture complète des paramètres est réservée aux administrateurs.
- **Alternatives écartées** : colonnes sur `association` (mélange identité
  et paiement) ; Markdown pour les instructions (nouvelle dépendance de rendu
  côté frontend, risque XSS).

## R8. E-mails

- **Décision** : quatre gabarits (`membership-validated`,
  `membership-refused`, `membership-cancelled`, `membership-paid`), texte et
  HTML, mis en file dans la transaction de l'action. `mail.Data` gagne des
  champs facultatifs (`Member`, `Season`, `Amount`, `Reference`, `Reason`).
  Le destinataire est le compte qui gère la fiche au moment de l'action,
  s'il existe et n'est pas désactivé (FR-023, spec › cas limites). Le lien de
  validation mène à `{BaseURL}/adhesions/{id}` (sans jeton : la page exige
  une session, et la connexion ramène sur l'URL demandée via le paramètre
  `suite` existant, FR-022c). Aucun lien pour une adhésion gratuite.
- **Rationale** : constitution (e-mails différés) ; un nouvel assistant
  `enqueueNotice` à côté de `enqueueMail` (qui reste dédié aux liens à
  jeton). Le motif de refus figure dans l'e-mail (US2 §7) mais jamais dans
  les journaux.

## R9. Règles ajoutées aux fonctionnalités 004 et 005

- **Décision** :
  - `delete-membership-fee` : si une adhésion utilise le tarif → `409`
    « Ce tarif est utilisé par au moins une adhésion : il ne peut pas être
    supprimé. » ; `MembershipFee` gagne `inUse` pour que la fiche de la
    saison masque « Supprimer » (FR-005).
  - `delete-my-member` : si la fiche porte une adhésion `pending` ou
    `validated` → `409` « Ce membre a une adhésion en cours. Annulez la
    demande ou contactez le bureau. » (FR-010a).
  - `delete-member` (administration) : supprime les adhésions avec la fiche
    (`CASCADE`) ; `get-member` expose `memberships` (FR-010) et le dialogue
    les nomme (FR-021). Le journal `member.delete` gagne
    `memberships_deleted`.
  - `delete-account` avec `members=delete` : supprime aussi les adhésions de
    ces fiches (par la même cascade) ; le dialogue de 005 mentionne « et leurs
    adhésions ».
- **Rationale** : contrôles explicites dans la transaction pour un message
  clair ; les clés étrangères restent des filets.

## R10. Liste, recherche et chiffres (administration)

- **Décision** :
  - `list-memberships` filtre en SQL par saison (`season` = identifiant, ou
    `all` ; absent = saison courante, à défaut la plus récente ouverte, à
    défaut la plus récente), `status`, `payment` et fiche (`member`) ; `q`
    passe par `FTSTrigramQuery` puis cherche dans deux tables FTS5 trigram :
    `memberships_fts` (référence de paiement, table à contenu externe sur
    `memberships`, triggers d'insertion, de suppression et de mise à jour de
    `payment_reference`) et `members_fts` (existante : prénom, nom, adresse,
    téléphone de la fiche). Une adhésion correspond si la requête trouve sa
    référence **ou** sa fiche ; « 0042 », « adh-2627 » ou « mart » sont
    acceptés, aucun fragment de 3 caractères → `422`. Tri en Go : demandes
    en attente d'abord par date de demande croissante, puis nom et prénom
    repliés, puis identifiant ; pagination 25 (100 au plus). La réponse
    rappelle la saison retenue (`seasonId`).
  - `get-membership-stats?season=` calcule en Go, sur les adhésions de la
    saison : nombre de validées, dont `first` et `renewal`, nombre en
    attente, répartition par tarif (nombre et somme), somme des montants
    appliqués, encaissé (`paid`), restant à percevoir (`due` + `announced`),
    détaillé par moyen annoncé (`none`, `cash`, `cheque`, `transfer`).
  - `get-membership-dashboard` : nombre total de demandes en attente (toutes
    saisons) et `paymentSetupNeeded` (aucun moyen activé et au moins une
    saison ouverte, non close, avec tarif) (FR-019, FR-022i).
- **Rationale** : même stratégie que 005 › R7 (volume associatif, tri sans
  accents imposé par la constitution) ; des agrégats calculés par le
  domaine sont testables et cohérents par construction avec la liste
  (SC-005, SC-007).

## R11. Écrans

- **Décision** :
  - Espace membre : `/adhesions` (adhésions par saison, cartes ; bouton
    « Demander une adhésion » ; messages d'indisponibilité),
    `/adhesions/demande` (formulaire : saison, membre, tarif en
    `RadioGroup` avec montants et description, type déclaré, montant
    calculé affiché en direct à partir des montants de l'offre),
    `/adhesions/:id` (détail ; si à payer : choix du moyen en `RadioGroup`,
    informations du moyen sélectionné, boutons « Copier », confirmation).
    La fiche `/mes-membres/:id` gagne une section « Adhésions ».
  - Administration : `/admin/adhesions` (chiffres de la saison en tête,
    filtres saison / état / paiement / recherche dans l'URL, tableau
    paginé), `/admin/adhesions/nouvelle` (membre par Combobox sur
    `list-members`, saison, tarif, type, règlement reçu facultatif),
    `/admin/adhesions/:id` (détail, aide saison précédente, dialogues
    Valider / Refuser / Corriger / Annuler / Constater le paiement),
    `/admin/parametres/paiements`, première sous-page des paramètres (formulaire des moyens de paiement, un `fieldset`
    par moyen avec interrupteur `Switch` base-ui). La fiche
    `/admin/membres/:id` gagne la section « Adhésions » et le lien « Créer une
    adhésion » (`?membre=`).
  - Tableau de bord : encarts « {n} demandes d'adhésion en attente » et
    « Paramétrez les moyens de paiement ».
  - Navigation : « Adhésions » (deux espaces) quitte `comingSoon` ;
    nouvelle entrée « Paramètres » (icône `Settings`) dans la section
    Configuration, entre « Comptes » et « Aide » (FR-022f).
- **Rationale** : principe IV (shell, composants et base-ui existants) ; le
  montant affiché avant envoi est lu dans `amounts.first` / `amounts.renewal`
  de chaque tarif, calculés par le serveur avec `AppliedAmount` (offre,
  `seasonFees` et tarifs de 004) : le frontend choisit une valeur selon le
  type sélectionné, sans connaître la règle (principe VI). Le serveur
  recalcule et fige le montant à la validation. « Copier » utilise
  `navigator.clipboard` et annonce « Copié » dans une région `aria-live`.
- **Alternatives écartées** : page de paiement séparée de la fiche
  d'adhésion (deux URL pour la même information ; le lien de l'e-mail mène à
  la fiche, qui montre le paiement en premier lorsqu'il est attendu).

## R12. Journalisation

- **Décision** : une ligne `INFO` par écriture réussie : `membership.request`,
  `membership.cancel`, `membership.validate`, `membership.refuse`,
  `membership.update`, `membership.create`, `membership.payment_announce`,
  `membership.payment_record`, `membership.payment_cancel`,
  `settings.payment_update`. Attributs : `actor_id`, `membership_id`,
  `member_id`, `season_id`, `space`, et selon l'action `status`, `kind`,
  `method`, `amount_cents`. Jamais de motif, de commentaire, de nom ni de
  coordonnées bancaires (`settings.payment_update` ne journalise que la liste
  des moyens activés).
- **Rationale** : FR-026, principe VII.

## R13. Données de test de bout en bout

- **Décision** : `internal/tools/e2eseed` crée une saison ouverte avec trois
  tarifs (dont un avec montant de renouvellement), un tarif gratuit, des
  fiches pour le compte membre, des paramètres de paiement avec les trois
  moyens activés et un IBAN de test valide.
- **Rationale** : les tests Playwright partent d'un état prêt, comme pour
  005.
