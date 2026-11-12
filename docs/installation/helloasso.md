# Paiement par carte (HelloAsso)

Les membres peuvent payer leurs adhésions et leurs inscriptions par carte
bancaire via [HelloAsso](https://www.helloasso.com), la plateforme de paiement
gratuite des associations françaises.

Cette fonction est facultative : sans les variables ci-dessous, Frigg
fonctionne normalement et ne propose pas la carte. Une indisponibilité de
HelloAsso n'empêche que le paiement par carte.

## Activer le paiement par carte

1. Créez un client API dans l'espace HelloAsso de l'association
   (*Mon compte › Intégrations et API*), avec le droit « Checkout ».
   Commencez par l'environnement de test,
   [helloasso-sandbox.com](https://www.helloasso-sandbox.com) : aucune somme
   réelle n'y est débitée.
2. Renseignez les trois premières variables ensemble, jamais dans un fichier
   commité :

   | Variable | Rôle |
   | --- | --- |
   | `FRIGG_HELLOASSO_CLIENT_ID` | Identifiant du client API |
   | `FRIGG_HELLOASSO_CLIENT_SECRET` | Secret du client API |
   | `FRIGG_HELLOASSO_ORGANIZATION` | Identifiant (slug) de l'association sur HelloAsso |
   | `FRIGG_HELLOASSO_ENVIRONMENT` | `sandbox` (défaut) ou `production` |

   `FRIGG_BASE_URL` doit alors être en `https://`, même en mode test :
   HelloAsso n'accepte que des adresses de retour sécurisées. L'instance doit
   aussi être joignable depuis Internet pour recevoir les notifications de
   paiement. En développement, exposez-la par un tunnel HTTPS (expose.dev,
   cloudflared, ngrok…) et indiquez son adresse dans `FRIGG_BASE_URL`.
3. Redémarrez Frigg. Dans *Administration › Paramètres › Paiements*, copiez
   l'adresse de notification affichée et déclarez-la dans l'espace HelloAsso,
   puis choisissez « Tester la connexion » et activez la carte.
4. Faites un paiement d'essai avec une carte de test.

## Passer en production

1. Créez un client API dans l'espace HelloAsso de production.
2. Renseignez ses identifiants et `FRIGG_HELLOASSO_ENVIRONMENT=production`.
3. Redémarrez Frigg, testez la connexion et déclarez la nouvelle adresse de
   notification dans l'espace HelloAsso de production.

## À savoir

- Changer de secret change l'adresse de notification : déclarez la nouvelle
  adresse dans HelloAsso.
- Les remboursements se font dans l'espace HelloAsso ; Frigg signale aux
  administrateurs ceux qui sont à faire.
- Le déroulement d'un paiement côté membre et côté administrateur est décrit
  dans [Paiement par carte](../usage/card-payment).
