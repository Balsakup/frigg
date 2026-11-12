# Saisons

**Rôle requis** : Administrateur

Une **saison** est la période couverte par une adhésion (par exemple du
1ᵉʳ septembre au 31 août). Les [tarifs](./fees), les
[adhésions](./memberships) et les [événements](./events) sont rattachés à une
saison. Elles se gèrent dans *Configuration › Saisons*.

## Règles

- Une saison a une date de début et une date de fin, toutes deux incluses, et
  dure au plus 24 mois.
- Les saisons se suivent **sans chevauchement ni trou** : chaque saison
  commence le lendemain de la fin de la précédente.
- Son libellé est calculé à partir des dates : « 2027 » si elle tient dans une
  année civile, sinon « 2026-2027 ».
- La **saison courante** est celle qui contient la date du jour ; elle porte
  la mention « Saison courante » dans la liste.

## États

| État | Signification |
| --- | --- |
| **Brouillon** | Saison en préparation, invisible des membres. Les dates et les tarifs se modifient librement |
| **Ouverte** | Les membres peuvent y adhérer. Seule la date de fin peut encore changer, sans être antérieure à la date du jour |
| **Close** | La date de fin est passée. Plus rien n'est modifiable |

- On **ouvre** une saison en brouillon après confirmation, quelle que soit sa
  date de début, à condition que la saison précédente ne soit pas en
  brouillon. Plusieurs saisons peuvent être ouvertes en même temps (la
  saison en cours et la suivante, par exemple).
- La **clôture** est automatique, le lendemain de la date de fin ; elle n'est
  jamais une action manuelle.

## Créer une saison

- **Première saison** : vous choisissez les deux dates. La date de début peut
  être passée, mais la date de fin doit être aujourd'hui ou plus tard.
- **Saisons suivantes** : la date de début est imposée au lendemain de la fin
  de la dernière saison.
- **Dupliquer** la dernière saison crée la suivante, de même durée, avec une
  copie de tous ses tarifs. Les tarifs des deux saisons évoluent ensuite
  séparément.

Après une interruption, vous pouvez recréer une à une les saisons manquantes
(dates passées) : elles sont closes dès leur création.

Une saison créée est à l'état brouillon.

## Modifier ou supprimer

- En brouillon, la date de fin se modifie, ainsi que la date de début s'il
  s'agit de la première saison.
- Modifier la date de fin d'une saison suivie d'une saison en brouillon décale
  le début de celle-ci au lendemain. C'est refusé si la saison suivante est
  ouverte.
- Seule la **dernière** saison peut être supprimée, et seulement si elle n'a
  jamais été ouverte. Ses tarifs sont supprimés avec elle.

## Tableau de bord

Le tableau de bord de l'administration vous invite à agir quand aucune saison
n'existe, quand aucune ne contient la date du jour, ou quand la saison
courante est encore en brouillon.
