# Research : paiement des adhésions par carte bancaire (HelloAsso)

Décisions techniques de la fonctionnalité 008. Chaque section donne la
décision, sa justification et les alternatives écartées. Sources HelloAsso :
documentation développeur (<https://dev.helloasso.com>), spécification OpenAPI
publique (`helloasso/helloasso-open-api`).

## R1. Ce que fournit HelloAsso (faits vérifiés)

- **Authentification** : OAuth2 `client_credentials` sur
  `POST {hôte}/oauth2/token` (formulaire `client_id`, `client_secret`,
  `grant_type`). Jeton d'accès valable 1 799 s ; jeton de rafraîchissement
  valable 30 jours, renouvelé à chaque rafraîchissement
  (`grant_type=refresh_token`). HelloAsso demande de ne pas regénérer un jeton
  à chaque appel.
- **Hôtes** : production `https://api.helloasso.com` (API sous `/v5`),
  environnement de test `https://api.helloasso-sandbox.com` (même chemins),
  avec un compte d'association créé sur `helloasso-sandbox.com`. Les
  identifiants, le jeton et l'hôte doivent appartenir au même environnement
  (`unauthorized_client` sinon).
- **Checkout** : `POST /v5/organizations/{slug}/checkout-intents` avec
  `totalAmount` et `initialAmount` (centimes), `itemName` (≤ 250),
  `backUrl`, `errorUrl`, `returnUrl` (HTTPS obligatoire),
  `containsDonation`, `payer` (prénom, nom, e-mail…) et `metadata` (JSON,
  ≤ 20 000 caractères). Réponse : `id` (intention) et `redirectUrl`, **valable
  15 minutes**.
- **Retours navigateur** : `returnUrl` reçoit `checkoutIntentId`,
  `code=succeeded`, `orderId` ; `errorUrl` reçoit `checkoutIntentId` et
  parfois `error` ; `backUrl` ne reçoit rien. HelloAsso précise que ces
  paramètres sont falsifiables.
- **Vérification** : `GET /v5/organizations/{slug}/checkout-intents/{id}`
  renvoie l'intention, ses `metadata`, et `order` **seulement si le paiement
  est autorisé**. Le paiement (`order.payments[]`) porte `amount`,
  `amountTip` (contribution volontaire), `state` (`Authorized`, …), `date`.
- **Notifications** : `POST` JSON vers une URL déclarée par l'association
  dans *Mon compte › Intégrations et API* (5 URL au plus), types `Order`,
  `Payment`, `Form`, `Organization`. Les `metadata` du checkout y sont
  renvoyées à son émetteur. Toute réponse autre que `200` déclenche une
  nouvelle tentative (`min(48 h, 3 × 2^n)` s, 16 essais, ≈ 27 h). **Aucune
  signature** n'est documentée, ni de plage d'adresses IP sources.

**Conséquence** : une notification n'est qu'un signal. Frigg constate un
paiement uniquement après `GET checkout-intents/{id}` authentifié (principe II
amendé, FR-007).

**À confirmer en environnement de test pendant l'implémentation** (tâche
dédiée) : unité de `payments[].amount` et `amountTip` (centimes attendus en
v5), valeurs de `state` d'un paiement refusé, forme exacte du corps d'une
notification `Order` et `Payment` (emplacement de `metadata`), URL d'une
commande dans l'espace HelloAsso.

## R2. Configuration de l'instance

**Décision** : quatre variables, toutes facultatives mais **tout ou rien** :

| Variable | Rôle |
| --- | --- |
| `FRIGG_HELLOASSO_CLIENT_ID` | identifiant du client API |
| `FRIGG_HELLOASSO_CLIENT_SECRET` | secret du client API |
| `FRIGG_HELLOASSO_ORGANIZATION` | identifiant (slug) de l'association sur HelloAsso |
| `FRIGG_HELLOASSO_ENVIRONMENT` | `sandbox` (défaut) ou `production` |

- Aucune des trois premières : HelloAsso n'est pas configuré, Frigg démarre
  normalement (principe V amendé).
- Une partie seulement, ou un environnement inconnu : échec du démarrage
  avec un message explicite (« FRIGG_HELLOASSO_CLIENT_SECRET est obligatoire
  lorsque FRIGG_HELLOASSO_CLIENT_ID est renseigné »).
- HelloAsso configuré et `FRIGG_BASE_URL` non HTTPS : échec du démarrage
  (HelloAsso refuse des URL de retour non HTTPS), sauf si
  `FRIGG_HELLOASSO_API_URL` est renseignée.
- `FRIGG_HELLOASSO_API_URL`, réservée aux tests, remplace l'hôte HelloAsso :
  elle permet aux tests de bout en bout de viser un faux HelloAsso local.
  Comme toute variable, elle est documentée dans `.env.example` (commentée,
  avec la mention « Ne pas renseigner en production »).
- Le défaut `sandbox` évite qu'une instance mal configurée débite de vraies
  cartes ; passer en réel est un geste explicite de l'hébergeur.
- `.env.example` et la documentation d'installation décrivent les variables
  et la marche à suivre (compte HelloAsso, client API, déclaration de
  l'adresse de notification).

**Alternatives écartées** : saisie dans « Paramètres » (refusée en
clarification) ; mode test dans « Paramètres » (les identifiants de test et de
production sont distincts, le mode leur est donc lié).

## R3. Client HelloAsso : paquet `internal/helloasso`

**Décision** : un client écrit avec `net/http` et `encoding/json`, sans SDK
(liste fermée des dépendances, aucun SDK Go officiel). Il ne connaît ni la
base ni les adhésions.

- `New(cfg, httpClient, now)` ; `Configured()` ; méthodes
  `Organization(ctx)` (test de connexion : `GET /v5/organizations/{slug}`),
  `CreateCheckout(ctx, CheckoutRequest)`, `GetCheckout(ctx, id)`.
- **Jeton** gardé en mémoire uniquement (jamais en base ni dans les
  journaux), protégé par un `sync.Mutex`, renouvelé 60 s avant expiration par
  le jeton de rafraîchissement, puis par `client_credentials` en cas d'échec.
  Un `401` sur un appel déclenche un seul renouvellement puis une seule
  nouvelle tentative.
- **Délais** : `http.Client` dédié, 10 s par appel (SC test de connexion
  < 10 s) ; aucune nouvelle tentative automatique sur `POST checkout-intents`
  (pas de double intention).
- **Erreurs typées** : `ErrUnauthorized` (identifiants refusés),
  `ErrForbidden` (droit « Checkout » manquant), `ErrNotFound` (association
  introuvable), `ErrUnavailable` (réseau, délai, 5xx). Les messages
  d'erreur ne contiennent jamais le secret ni le jeton.
- **Garde-fou** : `redirectUrl` n'est renvoyée au navigateur que si son hôte
  appartient à `helloasso.com` ou `helloasso-sandbox.com` (ou à l'hôte de
  `FRIGG_HELLOASSO_API_URL`).
- Tests : `httptest.Server` simulant les points de HelloAsso (jeton,
  expiration, `401` puis succès, intention avec et sans `order`, délai).

## R4. Tentatives de paiement en ligne

**Décision** : table `online_payments`, une ligne par tentative
([data-model.md](./data-model.md)). Démarrer un paiement :

1. Lecture de l'adhésion (cloisonnée au compte), `membership.ActionsFor`
   doit autoriser `PayByCard` : validée, non gratuite, à payer ou au paiement
   annoncé, carte disponible (R7).
2. **Reprise** : s'il existe une tentative `pending` de cette adhésion, du
   même environnement, du même montant, avec un `redirect_url` créé il y a
   moins de 14 minutes (validité de 15 min chez HelloAsso, marge d'une
   minute), elle est renvoyée telle quelle (double clic, deux onglets).
3. Sinon, insertion d'une tentative `pending` (transaction courte), puis
   **hors transaction** `CreateCheckout` avec :
   - `totalAmount = initialAmount = amount_cents`, `containsDonation=false` ;
   - `itemName` = « Adhésion {saison} — {tarif} — {référence} »
     (tronqué à 250), sans nom du membre (FR-004, clarification du
     2026-11-02) ;
   - `payer` : e-mail du compte, prénom et nom de la fiche « Vous » du
     compte si elle existe (FR-004 ; aucune autre donnée) ;
   - `metadata` = `{"frigg": {"payment": <id tentative>, "membership": <id>,
     "reference": "<référence de paiement>"}}` ;
   - `returnUrl` = `{BASE_URL}/adhesions/{id}?carte=retour&tentative={n}`,
     `errorUrl` = `…?carte=erreur&tentative={n}`,
     `backUrl` = `…?carte=annule&tentative={n}`.
4. Succès : la tentative reçoit `checkout_intent_id` et `redirect_url`.
   Échec : elle passe à `failed` (`error_code` = `unavailable`,
   `unauthorized`…) et l'API répond `503` avec un message invitant à choisir
   un autre moyen.

Le montant envoyé est toujours le montant appliqué lu en base, jamais une
valeur du navigateur.

**Alternatives écartées** : stocker l'intention sur la ligne `memberships`
(perd l'historique, impossible de détecter un double paiement) ; créer
l'intention avant la ligne (pas d'identifiant à mettre dans `metadata`).

## R5. Vérification et constatation (cœur de la fonctionnalité)

**Décision** : une seule fonction `verifyOnlinePayment(ctx, paymentID,
source)`, appelée par les trois déclencheurs : retour du navigateur,
notification, vérification par un administrateur ou par le balayage (R8).

1. Lecture de la tentative (pool de lecture). Si elle est déjà `succeeded`,
   rien à faire (idempotence, FR-008a).
2. **Limitation** : au plus une vérification par tentative toutes les 5 s
   (`last_verified_at`) ; au-delà, on renvoie l'état connu.
3. Appel `GetCheckout(checkout_intent_id)` **hors transaction d'écriture**.
4. Contrôles de rattachement : `metadata.frigg.payment` et
   `metadata.frigg.membership` égaux à ceux de la tentative, environnement
   identique. Sinon : journalisé et ignoré.
5. `BEGIN IMMEDIATE`, relecture de la tentative et de l'adhésion, puis
   décision par le domaine pur `membership.SettleOnline` :

| Situation relue | Effet |
| --- | --- |
| pas de `order` ou aucun paiement `Authorized` | inchangé (encore en cours) ; si le retour navigateur dit `erreur` → `failed`, s'il dit `annule` → `abandoned` |
| paiement autorisé, adhésion validée à payer ou annoncée, montant reçu = montant appliqué | tentative `succeeded` ; adhésion `paid`, `paid_method='card'`, `paid_on` = date du paiement à Paris, `paid_at`, `paid_by` NULL, `paid_online_payment_id` ; e-mail `membership-paid` |
| idem, montant reçu ≠ montant appliqué | idem + signalement `amount_mismatch` |
| adhésion déjà payée (autre tentative ou autre moyen) | tentative `succeeded`, adhésion inchangée, signalement `duplicate` |
| adhésion annulée | tentative `succeeded`, adhésion inchangée, signalement `refund_due` |

6. Le montant reçu est `payments[].amount − amountTip` du paiement autorisé
   (contribution volontaire exclue) ; la date du paiement est
   `payments[].date`.
7. Une tentative `failed`, `abandoned` ou `expired` reste vérifiable : un
   paiement autorisé découvert ensuite l'emporte toujours (un retour
   « erreur » falsifié ne peut pas faire perdre un paiement).

L'écriture est sérialisée (pool d'écriture à une connexion, `BEGIN
IMMEDIATE`) et conditionnée par l'état relu : deux vérifications
simultanées (notification et retour) produisent une seule constatation et un
seul e-mail. Un index unique sur `(environment, helloasso_payment_id)`
garantit qu'un même paiement HelloAsso n'est rattaché qu'une fois.

**Alternatives écartées** : se fier au contenu de la notification (non
signé) ; se fier au `code=succeeded` du retour (falsifiable).

## R6. Adresse de notification

**Décision** : opération huma `POST /api/webhooks/helloasso/{key}`
(`receive-helloasso-notification`), déclarée `auth.Public` et ajoutée à
`allowedPublic` dans `access_test.go` (exception du principe II, v1.7.0).
La réponse suit le protocole de HelloAsso (`200` attendu, toute autre
réponse renvoyée pendant environ 27 h, R1) pour qui détient la clé, et ne
révèle rien aux autres.

- `{key}` = 32 premiers caractères hexadécimaux de
  `HMAC-SHA256(client_secret, "frigg-helloasso-notifications")` : l'adresse
  complète, affichée dans « Paramètres » pour être copiée dans HelloAsso, ne
  révèle pas le secret et n'est connue que de HelloAsso et de l'instance.
  Changer de secret change l'adresse ; « Paramètres » l'affiche toujours à
  jour.
- Entrées **non contraintes par huma** : `key` est une chaîne libre (pas de
  `pattern`, pour qu'une clé mal formée ne produise pas de `422`) comparée en
  temps constant (`hmac.Equal`) ; le corps est lu brut (`RawBody`), jusqu'à
  1 Mio, le surplus étant ignoré (pas de `413`) ; seuls `eventType` et
  `metadata.frigg.payment` sont décodés, de façon tolérante.
- **Réponses** :

| Situation | Réponse | Traitement |
| --- | --- | --- |
| clé absente, fausse ou mal formée | `200`, corps vide | aucun, pas d'appel à HelloAsso |
| clé valide, corps illisible, autre vente, type `Form` ou `Organization`, tentative inconnue ou d'un autre environnement | `200`, corps vide | ignorée, journalisée au niveau `debug` |
| clé valide, tentative connue, vérification aboutie (constatée, toujours en cours, déjà constatée, limitation de 5 s atteinte) | `200`, corps vide | `verifyOnlinePayment(…, "notification")` |
| clé valide, tentative connue, HelloAsso injoignable ou erreur interne (base, transaction) | `503`, corps vide | HelloAsso renverra la notification |
| débit dépassé | `429`, corps vide | aucun |

  Aucune réponse ne porte de détail d'erreur, de version ni de
  configuration (corps toujours vide, pas de `problem+json`). Seul qui
  détient la clé peut distinguer `200` de `503`.
- **Limitation de débit** par IP (`golang.org/x/time/rate`, nouveau
  `KeyedLimiter` `NotificationIP`, 60 requêtes/min), appliquée avant la
  comparaison de la clé.
- **CSRF** : `http.CrossOriginProtection` laisse passer les requêtes sans
  `Origin` ni `Sec-Fetch-Site` (appel de serveur à serveur) : aucune
  exemption à coder. Un test vérifie qu'une notification passe la chaîne
  HTTP complète et qu'une requête portant `Sec-Fetch-Site: cross-site` est
  refusée.
- Aucune session n'est créée ni lue ; aucun cookie n'est posé.
- Le balayage (R8) reste le filet de sécurité si HelloAsso épuise ses
  nouvelles tentatives ou si l'adresse est mal déclarée.

**Alternatives écartées** : gestionnaire `net/http` hors huma (sortirait de
la liste centralisée vérifiée par la matrice d'accès) ; réponse toujours
`200` (prive Frigg des nouvelles tentatives de HelloAsso lors d'une erreur
passagère) ; `422`/`413` sur une entrée mal formée (réponse dépendant du
contenu pour qui ne détient pas la clé, interdit par le principe II) ;
restriction par IP (non documentée par HelloAsso).

## R7. Disponibilité du moyen « Carte bancaire »

**Décision** : la carte est proposée à un adhérent si et seulement si :

1. HelloAsso est configuré (R2) ;
2. `payment_settings.card_enabled = 1` ;
3. le dernier test de connexion a réussi **pour l'environnement courant**
   (`helloasso_test_environment` = environnement configuré) ;
4. l'adhésion est payable et son montant ≥ 1 € (`membership.MinCardAmount`,
   constante à ajuster si le test en environnement de démonstration révèle un
   autre minimum).

`card_enabled` est conservé quand les identifiants disparaissent ou que
l'environnement change (US2 sc. 8, cas limite « bascule ») : la carte
redevient disponible dès qu'un nouveau test réussit. L'activation (`PUT`)
est refusée en `422` sur `card.enabled` si 1 ou 3 n'est pas rempli
(FR-013).

Le test de connexion (`POST /admin/settings/payment/helloasso-test`) appelle
`Organization(ctx)`, enregistre date, environnement, résultat, nom de
l'association et code d'erreur, et met à jour l'état de connexion (R9).

## R8. Balayage des tentatives

**Décision** : un worker interne `api.OnlinePaymentSweeper` (il réutilise la vérification R5 et la mise en file des e-mails du paquet `api`), lancé par `serve` à côté
du worker d'e-mails lorsque HelloAsso est configuré, toutes les 5 minutes :

- tentatives `pending` créées il y a plus de 60 minutes → une vérification
  (R5) ; sans paiement autorisé → `expired` ;
- HelloAsso injoignable → la tentative reste `pending` ; au-delà de 24 h,
  signalement `unverified` aux administrateurs (FR-008b) ;
- au plus 50 tentatives par passage, les plus anciennes d'abord.

Le balayage rattrape une notification perdue sans réintroduire de
sondage fréquent : chaque tentative est vérifiée une fois à son expiration.
Horloge injectée pour les tests.

**Alternative écartée** : sondage régulier de toutes les tentatives en cours
(écarté en clarification au profit des notifications).

## R9. État de la connexion et tableau de bord

**Décision** : `payment_settings` garde le dernier test explicite et le
dernier incident d'appel (`helloasso_error_at`, `helloasso_error_code`),
remis à zéro au premier appel réussi. Le tableau de bord
(`get-membership-dashboard`) ajoute :

- `helloassoFailing` : carte activée et (dernier test en échec, ou incident
  plus récent que le dernier succès) → encart « Connexion à HelloAsso en
  échec » (FR-019) ;
- `paymentAlerts` : nombre de signalements non traités → encart menant à la
  liste filtrée (FR-011).

## R10. Signalements

**Décision** : table `payment_alerts` (nature `duplicate`,
`amount_mismatch`, `refund_due`, `unverified`), créée dans la transaction de
la constatation ou de l'annulation, unique par `(online_payment_id, kind)`
pour rester idempotente. L'annulation par un administrateur d'une adhésion
payée par carte crée un signalement `refund_due` (US3 sc. 5). Un
administrateur traite un signalement (`POST
/admin/payment-alerts/{id}/resolve`) : date et auteur enregistrés, l'encart
disparaît quand il n'en reste plus.

## R11. Règles ajoutées à 006

- `Method` gagne `card` (« carte bancaire ») ; `Methods` le place en premier.
  `announce-my-payment` refuse `card` en `422` : on n'« annonce » pas une
  carte, on paie.
- `record-membership-payment` (constatation manuelle) refuse `card` en
  `422` : seul HelloAsso constate un paiement par carte.
- `cancel-membership-payment` est refusé (`409`) pour un paiement par carte
  (FR-010) ; `ActionsFor` renvoie `cancelPayment=false`.
- Correction (`update-membership`) et annulation (`cancel-membership`)
  d'une adhésion avec une tentative `pending` récente : l'API renvoie
  `cardPaymentPending=true` dans l'adhésion ; le dialogue avertit (pas de
  blocage, cas limite « Correction ou annulation pendant le paiement »).
- Chiffres de saison (`stats`) : `collectedByMethod` inclut `card` ; les
  paiements de test (`online_payments.environment='sandbox'`) sont exclus de
  `collected` et comptés à part dans `collectedTest`.

## R12. Migration

**Décision** : nouvelle migration `00009_online_payments.sql`.

Les `CHECK` de `memberships` (`announced_method`, `paid_method`) n'acceptent
pas `card` ; SQLite ne permet pas de modifier une contrainte. La migration
**reconstruit** `memberships` (nouvelle table, copie, suppression,
renommage, recréation des index, des triggers FTS et reconstruction de
`memberships_fts` par `INSERT INTO memberships_fts(memberships_fts)
VALUES('rebuild')`). Aucune table ne référence encore `memberships`, ce qui
rend la suppression sûre avec `foreign_keys=ON`.

Si le projet n'a encore été déployé sur aucune instance (règle
append-only de la constitution), modifier `00008` serait plus simple ; faute
de certitude, la reconstruction est retenue. Tests : migration sur base
vierge et sur une base `00008` peuplée (adhésions dans chaque état,
recherche par référence encore fonctionnelle).

## R13. Écrans

- **Page de paiement** (`/adhesions/:id`, existante) : « Carte bancaire »
  en premier dans le `RadioGroup` ; panneau `CardPaymentPanel` (montant,
  départ vers HelloAsso, contribution volontaire facultative, bouton « Payer
  40,00 € par carte », bandeau « Mode test » en sandbox). Le bouton appelle
  l'API puis `window.location.assign(redirectUrl)`. Au retour
  (`?carte=retour|erreur|annule&tentative=n`), la page appelle
  `confirm-my-card-payment`, annonce le résultat dans une région `aria-live`
  (« Paiement reçu, merci », « Le paiement n'a pas abouti… », ou « Paiement
  en cours de confirmation » avec nouvel essai automatique toutes les 3 s
  pendant 30 s), puis retire les paramètres de l'adresse.
- **Paramètres › Paiements** : carte « Carte bancaire (HelloAsso) » : état
  des identifiants, badge d'environnement, adresse de notification avec
  bouton Copier, bouton « Tester la connexion » et résultat, interrupteur
  désactivé avec explication tant que FR-013 n'est pas rempli.
- **Administration › Adhésion** : bloc « Paiements en ligne » (tentatives,
  état, montants, référence HelloAsso, lien vers l'espace HelloAsso),
  bouton « Vérifier auprès de HelloAsso » pour une tentative en cours,
  signalements avec « Marquer comme traité » (dialogue de confirmation).
- **Liste** : filtre « Moyen » (dont carte), badge de signalement ; recherche
  sur la référence HelloAsso (numéro de commande).
- **Tableau de bord** : encarts signalements et connexion en échec.
- Icônes `lucide-react` : `CreditCard`, `ExternalLink`, `TriangleAlert`,
  `RefreshCw`.

La CSP n'est pas modifiée : une navigation de premier niveau vers HelloAsso
n'est pas restreinte par `connect-src` ; aucun script ni iframe HelloAsso
n'est chargé (principe IV : aucune ressource tierce dans l'application).

## R14. Journalisation

Messages en français, clés en anglais, jamais de nom, d'e-mail, de secret,
de jeton, d'URL de redirection ni de donnée de carte :

- `paiement en ligne démarré` (`membership_id`, `online_payment_id`,
  `checkout_intent_id`, `environment`, `amount_cents`, `actor_id`) ;
- `paiement en ligne vérifié` (`source` = `return` | `notification` |
  `admin` | `sweep`, `outcome`, `order_id`) ;
- `signalement de paiement` (`kind`) ; `notification ignorée` (`reason`,
  niveau debug) ; `test de connexion HelloAsso` (`ok`, `error_code`) ;
- `appel HelloAsso en échec` (`operation`, `status`, `error_code`).

## R15. Tests de bout en bout

Un faux HelloAsso minimal (serveur HTTP Bun lancé par la configuration
Playwright) répond au jeton, au test d'organisation, à la création
d'intention (redirection vers une page « Payer / Refuser / Retour » qui
renvoie vers `returnUrl`, `errorUrl` ou `backUrl`) et à la lecture
d'intention ; il peut envoyer la notification à Frigg. Frigg est lancé avec
`FRIGG_HELLOASSO_API_URL` pointant vers lui. Ce faux service est un outil de
test (dossier `web/e2e/`), jamais embarqué dans le binaire.
