# Comptes et invitations

**Rôle requis** : Administrateur

La rubrique *Configuration › Comptes* liste tous les comptes de l'instance et
les invitations en attente. Le premier administrateur est créé en ligne de
commande (voir [Premier administrateur](../installation/first-admin)).

## Inviter une personne

Invitez une personne par son adresse électronique en choisissant son rôle
(membre par défaut, ou administrateur). Elle reçoit un e-mail avec un lien
valable 7 jours.

- Une adresse qui a déjà un compte ou une invitation en attente est refusée.
- Une invitation en attente peut être **renvoyée** (le lien précédent devient
  invalide) ou **annulée**.

## Trouver un compte

La liste est triée par adresse et indique pour chaque compte son rôle, son
état (actif ou désactivé), si son adresse est confirmée, sa date de création
et sa dernière connexion.

- **Recherche** : tout fragment d'au moins 3 caractères de l'adresse (`pont`
  trouve `jean.dupont@exemple.fr`), sans tenir compte des majuscules ni des
  accents. Plusieurs fragments séparés par des espaces doivent tous
  correspondre.
- **Filtres** : par rôle et par état. Les critères restent dans l'adresse de
  la page, si bien qu'un rechargement les conserve.

## Gérer un compte

Depuis la fiche d'un compte :

| Action | Effet |
| --- | --- |
| **Désactiver** | Ferme immédiatement toutes ses sessions et invalide ses liens en cours. Le compte ne peut plus se connecter |
| **Réactiver** | Le compte retrouve son rôle, son mot de passe et l'état de confirmation de son adresse |
| **Changer le rôle** | Membre ↔ administrateur, appliqué dès sa prochaine action, sans le déconnecter |
| **Envoyer un lien de réinitialisation** | Le compte reçoit le même lien que depuis « Mot de passe oublié ». Aucun administrateur ne voit ni ne définit le mot de passe d'un autre |
| **Supprimer** | Efface définitivement le compte, ses sessions et ses liens. Irréversible : préférez la désactivation |

Chaque action demande confirmation.

## Garde-fous

- Vous ne pouvez pas désactiver, supprimer, changer le rôle de votre propre
  compte, ni vous envoyer un lien de réinitialisation.
- Frigg refuse toute action qui laisserait l'instance sans administrateur
  actif.
