# E-mails groupés

**Rôle requis** : Administrateur

La rubrique *Communication* envoie un e-mail aux adhérents d'une saison,
par exemple pour relancer les cotisations impayées ou convoquer à
l'assemblée générale.

## 1. Choisir la cible

- **Saison** : une saison ouverte ou close (la saison courante par défaut).
  Seuls les **adhérents** comptent, c'est-à-dire les adhésions validées.
- **Situation de paiement** :

  | Cible | Adhérents retenus |
  | --- | --- |
  | Tous les adhérents | Tous |
  | À jour | Payée ou gratuite |
  | En impayé | Paiement annoncé ou à payer |
  | Personnalisée | Au choix parmi « À jour », « Paiement annoncé » et « À payer » |

- **Tarifs** : facultativement, un ou plusieurs tarifs de la saison.

## 2. Vérifier les destinataires

- L'adresse retenue est celle du **compte qui gère la fiche**, à défaut celle
  de la fiche.
- Une même adresse ne reçoit qu'**un exemplaire** : un parent pour plusieurs
  enfants, par exemple.
- Avant l'envoi, Frigg indique le nombre d'adhérents ciblés, de destinataires
  et d'adhérents **sans adresse**, avec leur liste.

## 3. Rédiger

- **Objet** : 150 caractères au plus. **Texte** : 10 000 caractères au plus,
  en texte simple (les retours à la ligne sont conservés).
- **Pièces jointes** : 5 fichiers au plus, 5 Mo par fichier et 10 Mo au
  total, aux formats PDF, JPEG, PNG, Word (.docx), Excel (.xlsx) et
  OpenDocument (.odt, .ods). Le type est vérifié sur le contenu ; les anciens
  formats (.doc, .xls) et les documents à macros sont refusés. Les fichiers
  d'une préparation abandonnée sont supprimés après 24 heures.
- Un **aperçu** montre le message tel qu'il sera reçu.

## 4. Envoyer

L'envoi se confirme dans un dialogue qui rappelle l'objet, la cible, le
nombre de destinataires et les pièces jointes. Si les destinataires ont
changé entre-temps, Frigg présente le nouveau décompte au lieu d'envoyer.

Chaque destinataire reçoit un e-mail individuel : aucune adresse n'est
visible des autres. Le nom de l'association apparaît comme expéditeur, et les
réponses arrivent à l'adresse d'expédition configurée sur le serveur.

Les e-mails partent en arrière-plan (voir [E-mails](../installation/email)).

## Historique

La rubrique liste les communications envoyées, la plus récente en premier.
Le détail d'une communication indique, pour chaque destinataire, si l'e-mail
est parti, en attente ou en échec, et permet de télécharger les pièces
jointes. Une communication envoyée ne se modifie plus.
