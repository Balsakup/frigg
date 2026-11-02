# Implementation Plan: Paiement des adhésions par carte bancaire (HelloAsso)

**Branch**: `008-helloasso-card-payment` | **Date**: 2026-11-02 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/008-helloasso-card-payment/spec.md`

## Summary

Ajouter « Carte bancaire » aux moyens de paiement des adhésions (006). Le
compte qui gère la fiche lance un paiement : Frigg crée une intention de
paiement HelloAsso pour le montant appliqué et redirige vers la page
sécurisée de HelloAsso. Le paiement est constaté automatiquement, **après
vérification authentifiée auprès de HelloAsso**, sur trois déclencheurs : le
retour du navigateur, la notification de HelloAsso (nouvelle adresse
publique autorisée par la constitution v1.7.0) et un balayage des tentatives
expirées. Les identifiants viennent de la configuration (`FRIGG_HELLOASSO_*`)
; « Paramètres » affiche l'état, l'adresse de notification, teste la
connexion et active le moyen. Une table `online_payments` trace chaque
tentative, une table `payment_alerts` signale doubles paiements, montants
inattendus et remboursements à faire. Un client HTTP maison
(`internal/helloasso`) parle à HelloAsso ; les règles de décision restent
dans le domaine pur `internal/membership`. Détails : [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.27.3 (backend) ; TypeScript 5.9 + React 19 (frontend)

**Primary Dependencies**: aucune nouvelle. Backend : `net/http` et
`encoding/json` (client HelloAsso), `crypto/hmac` (clé de l'adresse de
notification), huma/v2, sqlc, goose, `golang.org/x/time/rate` (déjà présent),
`internal/mail`. Frontend : base-ui (`RadioGroup`, `Switch`, `Dialog`),
TanStack Query, react-hook-form, lucide-react (`CreditCard`, `ExternalLink`,
`TriangleAlert`, `RefreshCw`).

**Storage**: SQLite ; migration `00009_online_payments.sql` : reconstruction
de `memberships` (`paid_method` accepte `card`, colonnes
`paid_online_payment_id` et `paid_amount_cents`), tables `online_payments`
et `payment_alerts`, colonnes HelloAsso de `payment_settings`
([data-model.md](./data-model.md)). Secret et jetons hors base.

**Testing**: `go test` : tests en table de `internal/membership`
(`SettleOnline`, `CardAvailability`, actions, chiffres) ; `internal/helloasso`
contre un faux HelloAsso `httptest` ; `internal/config` ; tests d'API
(humatest / httptest) dont retours falsifiés, notifications rejouées,
concurrence notification/retour, balayage à horloge injectée ; matrice
d'autorisation et d'isolation étendues ; test de migration sur base `00008`
peuplée. Playwright + axe (`card-payment.spec.ts`) avec un faux HelloAsso
Bun local ([R15](./research.md#r15-tests-de-bout-en-bout)).

**Target Platform**: serveur Linux, binaire unique `CGO_ENABLED=0`, joignable
en HTTPS depuis Internet pour recevoir les notifications ; navigateurs
récents, dès 320 px

**Project Type**: application web (API Go + SPA intégrée au binaire)

**Performance Goals**: test de connexion < 10 s (délai HTTP de 10 s) ;
constatation < 1 min après le paiement (SC-005a : notification immédiate,
vérification unique) ; démarrage d'un paiement en un appel HelloAsso

**Constraints**: aucune écriture SQLite pendant un appel HelloAsso (appels
hors transaction, relecture en `BEGIN IMMEDIATE`) ; constatation idempotente
; contenu du navigateur et des notifications jamais cru ; réponse de la
notification identique pour tout appel sans la clé valide ; aucun secret, jeton, nom, e-mail ni
URL de redirection dans les journaux ; aucune ressource HelloAsso chargée
dans l'application (simple redirection de premier niveau) ; commentaires
sqlc en ASCII

**Scale/Scope**: une association, quelques centaines à quelques milliers
d'adhésions par saison ; 6 opérations ajoutées, 9 modifiées ; 5 écrans
modifiés, aucune nouvelle route

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution v1.7.0.

| Principe | Vérification | Statut |
| --- | --- | --- |
| I. Une association, une instance | Une configuration HelloAsso par instance ; colonnes HelloAsso dans le singleton `payment_settings` ; aucune notion de locataire | ✅ |
| II. Authentification partout | 2 opérations `member`, 3 `admin`, 1 `public` : `receive-helloasso-notification`, **exception ajoutée en v1.6.0, précisée en v1.7.0** : `POST` seul, vérification auprès de HelloAsso comme seul effet ; sans la clé valide, `200` vide identique et aucun traitement ; avec la clé, protocole HelloAsso (`200` traitée ou ignorée, `503` erreur passagère) ; corps toujours vide, aucune donnée exposée ; ajoutée à `allowedPublic` ([R6](./research.md#r6-adresse-de-notification)). Le retour de HelloAsso arrive sur une page protégée (`/adhesions/:id`) | ✅ |
| III. Deux rôles | Niveaux déclarés par `register` ; aucun test de rôle dans les handlers ; matrice d'autorisation automatique | ✅ |
| IV. Un shell, un design system | Écrans existants étendus ; composants `ui/*`, `RadioGroup`, `Switch`, `Dialog` base-ui ; tokens `ok`/`warn`/`danger` ; icônes via `Icon` ; aucune ressource tierce chargée (redirection de premier niveau, pas d'iframe ni de script HelloAsso) ; confirmation avant traitement d'un signalement et annulation ; RGAA (régions `status`, lien externe annoncé) | ✅ |
| V. Binaire unique en Go pur | Client HelloAsso en bibliothèque standard, aucune dépendance ; HelloAsso est la dépendance externe **facultative** autorisée depuis la v1.6.0 : identifiants par variables d'environnement, absence et panne sans effet sur le démarrage ni sur les autres fonctions, secret jamais en base, renvoyé ni journalisé ; migration goose embarquée ; le faux HelloAsso des tests n'est pas embarqué | ✅ |
| VI. API typée et SQL généré | Énumérations (`card`, `OnlineStatus`, `AlertKind`) et formats (`{key}` en `pattern`) dans huma ; requêtes sqlc ; recherche par référence de paiement inchangée (FTS5 existante, reconstruite) ; numéro de commande HelloAsso en égalité exacte (identifiant numérique, pas de recherche textuelle) ; le frontend lit `paymentMethods`, `actions`, `card.available` sans recalculer de règle | ✅ |
| VII. Sécurité et RGPD | Aucune donnée de carte reçue ni stockée (FR-005) ; payeur limité à l'e-mail et au nom de la fiche « Vous » (FR-004) ; cloisonnement par compte (`…OfAccount`), `404` uniforme ; limitation de débit sur la notification (`KeyedLimiter` par IP) et de la vérification par tentative ; journaux sans données personnelles ([R14](./research.md#r14-journalisation)) | ✅ |
| Contraintes techniques | E-mail de constatation mis en file dans la transaction ; workers internes (e-mails existant, balayage ajouté) ; configuration par variables d'environnement documentées dans `.env.example` ; variables facultatives (règle amendée) ; écritures sérialisées en `BEGIN IMMEDIATE` | ✅ |
| Conventions de langue | Identifiants en anglais (`online_payments`, `card`, `refund_due`, `receive-helloasso-notification`) ; textes, messages et journaux en français | ✅ |
| Workflow | Nouvelle migration (append-only) ; matrices d'autorisation et d'isolation étendues ; contrôles de [quickstart.md](./quickstart.md) | ✅ |

**Re-check post-design (Phase 1)** : ✅ aucun écart. Points de vigilance :

1. **Seule `verifyOnlinePayment` constate un paiement par carte**, toujours
   après `GetCheckout` authentifié et contrôle des `metadata` ; aucune autre
   écriture ne met `paid_method = 'card'` (vérifié par revue et par le
   `CHECK` qui exige `paid_online_payment_id`).
2. L'appel HelloAsso a lieu **hors** transaction d'écriture ; la décision est
   prise après relecture dans `BEGIN IMMEDIATE` ; idempotence garantie par
   l'état relu et l'index unique `(environment, helloasso_payment_id)`.
3. Sans la clé valide, le gestionnaire de notification renvoie `200` vide
   sans rien traiter, quelle que soit l'entrée (clé mal formée, corps
   illisible ou volumineux) ; un test compare octet à octet ces réponses.
   Avec la clé, `503` uniquement sur erreur passagère pour une tentative
   connue ; corps toujours vide.
4. Le secret n'apparaît que dans `internal/config` et `internal/helloasso` ;
   un test vérifie qu'aucune ligne de journal ni réponse d'API ne le contient,
   pas plus que le jeton.
5. La reconstruction de `memberships` recrée à l'identique index, triggers et
   `CHECK` de 006 ; le test de migration sur base peuplée en est le garde-fou.

## Project Structure

### Documentation (this feature)

```text
specs/008-helloasso-card-payment/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── api.md
│   ├── helloasso.md
│   └── ui-routes.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
internal/
├── config/config.go                    # FRIGG_HELLOASSO_* (tout ou rien, HTTPS)
├── helloasso/                          # nouveau : client HTTP, jeton en mémoire, erreurs typées
│   ├── client.go
│   ├── checkout.go
│   ├── notification.go                 # NotificationKey, décodage tolérant
│   └── client_test.go
├── membership/
│   ├── membership.go                   # Method card, ActionsFor
│   ├── online.go                       # nouveau : CardAvailability, SettleOnline, AlertKind
│   ├── online_test.go
│   └── stats.go                        # carte, paiements de test
├── auth/ratelimit.go                   # KeyedLimiter NotificationIP
├── database/
│   ├── migrations/00009_online_payments.sql
│   ├── queries/online_payments.sql     # nouveau
│   ├── queries/payment_alerts.sql      # nouveau
│   ├── queries/memberships.sql         # constatation par carte, filtres
│   ├── queries/payment_settings.sql    # carte, test, incident
│   └── sqlc/                           # régénéré
├── api/
│   ├── card_payments.go                # start / confirm (membre), verify (admin), verifyOnlinePayment
│   ├── card_payments_test.go
│   ├── helloasso_webhook.go            # receive-helloasso-notification
│   ├── helloasso_webhook_test.go
│   ├── payment_alerts.go               # resolve-payment-alert
│   ├── online_sweeper.go               # OnlinePaymentSweeper
│   ├── settings.go                     # card, test de connexion
│   ├── memberships*.go                 # règles R11, champs ajoutés
│   └── access_test.go                  # allowedPublic
├── cli/serve.go                        # client HelloAsso, lancement du balayage
└── mail/templates/membership-paid.*    # libellé « carte bancaire »

web/
├── app/
│   ├── memberships/CardPaymentPanel.tsx          # nouveau
│   ├── memberships/PaymentMethodInfo.tsx         # carte
│   ├── routes/app/member/membership-detail.tsx   # retour HelloAsso
│   ├── routes/app/admin/settings-payments.tsx
│   ├── routes/app/admin/membership-detail.tsx
│   ├── routes/app/admin/memberships.tsx
│   ├── routes/app/admin/home.tsx
│   └── api/schema.d.ts                           # régénéré
└── e2e/
    ├── fake-helloasso.ts                         # nouveau, outil de test
    └── card-payment.spec.ts                      # nouveau

.env.example                            # variables HelloAsso documentées
README.md / documentation d'installation # mise en place HelloAsso
```

**Structure Decision**: structure existante (API Go + SPA dans `web/`). Un
seul paquet nouveau, `internal/helloasso`, isole le protocole HelloAsso
(testable sans base) ; l'orchestration (transactions, e-mails, journaux)
reste dans `internal/api` comme pour 006, et les règles dans
`internal/membership`.

## Complexity Tracking

Aucune violation de la constitution v1.7.0. Écarts de complexité retenus :

| Élément | Pourquoi | Alternative plus simple écartée |
| --- | --- | --- |
| Reconstruction de `memberships` dans `00009` | Les `CHECK` SQLite ne se modifient pas ; `paid_method` doit accepter `card` | Modifier `00008` : interdit si une instance l'a déjà reçue (append-only) ; à reconsidérer seulement s'il est certain qu'aucune instance n'a été déployée |
| Balayage périodique des tentatives | Rattrape une notification perdue (instance indisponible, adresse mal déclarée) et clôt les tentatives expirées ; une vérification par tentative | Sans balayage : paiements jamais constatés si l'adhérent ne revient pas et que la notification est perdue (SC-002) |
| Faux HelloAsso pour Playwright et `FRIGG_HELLOASSO_API_URL` | Tests de bout en bout reproductibles, sans secret ni réseau | Tests contre l'environnement de démonstration réel : dépendants du réseau et de secrets en CI |
