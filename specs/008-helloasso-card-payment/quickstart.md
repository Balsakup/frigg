# Quickstart : valider le paiement par carte (HelloAsso)

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[helloasso.md](./contracts/helloasso.md), [ui-routes.md](./contracts/ui-routes.md).
Prérequis : ceux de [006 › quickstart.md](../006-memberships/quickstart.md)
(Mailpit, `create-admin`, `go tool task dev`, saison ouverte avec tarifs).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 006 :

- `00009_online_payments.sql` s'applique sur une base vierge **et** sur une
  base `00008` peuplée : adhésions conservées à l'identique, recherche par
  référence de paiement encore fonctionnelle, `payment_settings.card_enabled
  = 0` ;
- matrice d'autorisation : 6 nouvelles opérations ;
  `receive-helloasso-notification` est la seule nouvelle opération publique
  et figure dans `allowedPublic` ;
- isolation : `start-my-card-payment` et `confirm-my-card-payment` par un
  autre compte → `404`, aucune tentative créée ;
- `internal/config` : aucune variable HelloAsso → démarrage normal ;
  variables partielles → erreur explicite ; environnement inconnu → erreur ;
  `FRIGG_BASE_URL` en `http://` avec HelloAsso → erreur (sauf
  `FRIGG_HELLOASSO_API_URL`) ;
- `internal/helloasso` (faux serveur `httptest`) : jeton obtenu puis
  réutilisé, rafraîchi avant expiration, `401` → un renouvellement et une
  nouvelle tentative, délai dépassé → `ErrUnavailable`, intention sans
  `order`, paiement `Authorized` avec contribution, `redirectUrl` d'hôte
  inattendu refusée, secret absent de toute erreur ;
- `internal/membership` (tests en table) : `CardAvailability`, `CanPayByCard`, `MemberActionsFor`/`AdminActionsFor`
  avec carte, `SettleOnline` pour chaque ligne du tableau de
  [R5](./research.md#r5-vérification-et-constatation-cœur-de-la-fonctionnalité),
  `Stats` avec carte et paiements de test ;
- tests d'API :
  - lancement → tentative `pending`, montant = montant appliqué ; second
    lancement dans les 14 min → `resumed: true`, aucune nouvelle intention ;
  - HelloAsso injoignable → `503`, tentative `failed` ;
  - retour `code=succeeded` alors que HelloAsso ne renvoie pas d'`order` →
    adhésion **non** payée (retour falsifié) ;
  - notification et retour simultanés → une constatation, un e-mail en file ;
  - sans clé valide (absente, fausse, mal formée, corps illisible ou > 1 Mio)
    → `200` vide identique, aucune écriture ni appel à HelloAsso ;
  - clé valide : tentative inconnue, autre vente, notification rejouée →
    `200` vide sans effet ; HelloAsso injoignable ou erreur interne → `503`
    vide, puis constatation au renvoi ;
  - paiement autorisé sur adhésion déjà payée par chèque → signalement
    `duplicate` ; montant différent → `amount_mismatch` ; adhésion annulée →
    `refund_due` ;
  - `failed` puis paiement autorisé découvert → `succeeded` ;
  - annonce `card`, constatation manuelle `card` → `422` ; annulation de la
    constatation d'un paiement par carte → `409` ;
  - activation sans configuration ou sans test réussi → `422` sur
    `body.card.enabled` ;
  - balayage (horloge injectée) : tentative de plus de 60 min sans paiement
    → `expired` ; HelloAsso injoignable plus de 24 h → `unverified` ;
  - chaîne HTTP complète : la notification passe `CrossOriginProtection` sans
    cookie, une requête `Sec-Fetch-Site: cross-site` est refusée, le débit
    est limité (`429`) ;
  - journaux : aucun e-mail, nom, secret, jeton ni URL de redirection.
- Playwright + axe (`card-payment.spec.ts`, faux HelloAsso de
  [R15](./research.md#r15-tests-de-bout-en-bout)) : paiement réussi,
  refusé, abandonné, retour sans revenir (notification seule), paramètres
  HelloAsso ; aucune violation `serious` ou `critical`.

## Validation manuelle en environnement de test HelloAsso

1. Créer une association sur `helloasso-sandbox.com`, puis un client API
   (droit « Checkout ») dans *Mon compte › Intégrations et API*.
2. Exposer l'instance de développement en HTTPS (tunnel), puis dans `.env` :

   ```sh
   FRIGG_BASE_URL=https://<tunnel>
   FRIGG_HELLOASSO_CLIENT_ID=…
   FRIGG_HELLOASSO_CLIENT_SECRET=…
   FRIGG_HELLOASSO_ORGANIZATION=<slug>
   FRIGG_HELLOASSO_ENVIRONMENT=sandbox
   ```

3. Administrateur › Paramètres › Paiements : la carte affiche « Mode test »
   et l'adresse de notification. La copier dans l'espace HelloAsso de test.
   « Tester la connexion » → nom de l'association en moins de 10 s. Activer
   la carte.
4. Membre : demander une adhésion, la faire valider, suivre le lien de
   l'e-mail, choisir « Carte bancaire » : bandeau « Mode test », bouton
   « Payer … par carte ». Payer avec une carte de test : retour sur Frigg,
   « Paiement reçu », e-mail de constatation dans Mailpit (SC-001, SC-005a).
5. Recommencer en fermant l'onglet juste après la validation chez HelloAsso :
   l'adhésion passe à « Payée » en moins d'une minute grâce à la
   notification (SC-002).
6. Refuser un paiement (carte de test refusée) : message d'échec, adhésion
   toujours à payer.
7. Administrateur › Adhésion payée par carte : montant reçu, commande
   HelloAsso, lien vers l'espace HelloAsso ; annuler l'adhésion → signalement
   « remboursement à effectuer » ; le traiter.
8. Retirer les variables HelloAsso et redémarrer : Frigg démarre, la carte
   n'est plus proposée, « Paramètres » l'explique.
9. Profiter de cette session pour confirmer les points ouverts de
   [R1](./research.md#r1-ce-que-fournit-helloasso-faits-vérifiés) (unités des
   montants, forme des notifications, URL d'une commande, montant minimal)
   et ajuster `contracts/helloasso.md` si besoin.
