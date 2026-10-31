# Modèle de données : Gestion des saisons

Complète [002 › data-model.md](../002-admin-account-management/data-model.md).
Décisions : [research.md](./research.md).

## Table `seasons` (migration `00004_seasons.sql`)

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `id` | `INTEGER` | `PRIMARY KEY` | Identifiant |
| `starts_on` | `TEXT` | `NOT NULL`, `UNIQUE`, `CHECK (date(starts_on) = starts_on)` | Premier jour (inclus), `AAAA-MM-JJ` |
| `ends_on` | `TEXT` | `NOT NULL`, `CHECK (date(ends_on) = ends_on)`, `CHECK (ends_on > starts_on)` | Dernier jour (inclus), `AAAA-MM-JJ` |
| `opened_at` | `TEXT` | `NULL` | Date et heure d'ouverture (UTC, format de `timeFormat`) ; `NULL` = brouillon |
| `created_at` | `TEXT` | `NOT NULL` | Date et heure de création (UTC) |

Table `STRICT`. Aucun index supplémentaire : `UNIQUE (starts_on)` sert au tri
de `ListSeasons`. Aucune clé étrangère pour l'instant ; adhésions, tarifs et
écritures comptables y feront référence plus tard (et la suppression devra
alors vérifier qu'aucune donnée n'y est rattachée).

## Invariants (contrôlés par `internal/season` dans la transaction)

1. **Continuité** : triées par `starts_on`, chaque saison sauf la première
   commence le lendemain du `ends_on` de la précédente (FR-006).
2. **Durée** : `ends_on > starts_on` et `ends_on ≤ starts_on + 24 mois − 1 jour`
   (FR-004).
3. **Ordre d'ouverture** : une saison ne peut avoir `opened_at` renseigné que
   si la précédente l'a aussi ou est close (FR-011).

## Attributs calculés (jamais stockés)

| Attribut | Calcul | Exigences |
| --- | --- | --- |
| `status` | `closed` si `ends_on < aujourd'hui` ; sinon `open` si `opened_at` renseigné ; sinon `draft` | FR-010, FR-012 |
| `isCurrent` | `starts_on ≤ aujourd'hui ≤ ends_on` | FR-014 |
| `label` | voir [research.md › R6](./research.md#r6-libellé) | FR-005 |
| `actions`, `endsOnMin`, `endsOnMax` | voir ci-dessous | FR-015 à FR-019, FR-022 |
| `shiftsSeasonLabel` | libellé de `N` si `N` est en brouillon et `editEndsOn` vrai, sinon `null` | FR-017 |

« Aujourd'hui » = date du jour à Europe/Paris ([R3](./research.md#r3-aujourdhui--fuseau-et-horloge-injectable)).

## Cycle de vie

```text
            créer / dupliquer
                   │
                   ▼
            ┌────────────┐   ouvrir (manuel)   ┌──────────┐
            │  brouillon │ ──────────────────▶ │ ouverte  │
            └────────────┘                     └──────────┘
                   │  fin < aujourd'hui              │  fin < aujourd'hui
                   ▼        (automatique)            ▼
            ┌──────────────────────────────────────────────┐
            │                    close                     │
            └──────────────────────────────────────────────┘
  supprimer : dernière saison jamais ouverte (brouillon, ou close sans
             avoir été ouverte : comblement raté)
```

## Actions autorisées

Notations : `S` la saison, `P` la précédente, `N` la suivante, `J` la date du
jour.

| Action | Condition | Exigence |
| --- | --- | --- |
| `editStartsOn` | `S` brouillon et première saison | FR-015 |
| `editEndsOn` | `S` brouillon ou ouverte, et `N` absente ou en brouillon | FR-015, FR-016, FR-017 |
| `open` | `S` brouillon, et `P` absente, ouverte ou close | FR-011 |
| `duplicate` | `S` est la dernière saison (tout état) | FR-009 |
| `delete` | `S` dernière saison et jamais ouverte (`opened_at` nul), close comprise | FR-019 |

`openBlockedReason` est renseigné quand `S` est en brouillon mais que `P` l'est
aussi : « Ouvrez d'abord la saison {libellé de P}. ». Une saison close n'a
aucune action sauf, si elle est la dernière, `duplicate` (préparer la suite
après une interruption ; la copie peut être passée elle aussi) et `delete` si
elle n'a jamais été ouverte (comblement raté).

## Bornes de saisie

| Champ | Minimum | Maximum |
| --- | --- | --- |
| `endsOn` (création d'une saison suivante) | `startsOn` + 1 jour (fin passée autorisée : trou à combler) | `startsOn` + 24 mois − 1 jour |
| `endsOn` (première saison, modification) | le plus tardif de : `J` ; `startsOn` + 1 jour ; si `N` en brouillon, `N.ends_on` − 24 mois | le plus précoce de : `startsOn` + 24 mois − 1 jour ; si `N` en brouillon, `N.ends_on` − 2 jours |
| `startsOn` (première saison, brouillon) | aucun (peut être passé) | `endsOn` − 1 jour |

Décalage de `N` (FR-017) : nouveau `N.starts_on` = nouveau `S.ends_on` + 1 jour ;
il doit rester strictement avant `N.ends_on` (d'où le maximum) et `N` ne doit
pas dépasser 24 mois (d'où le minimum). `endsOnMin` et `endsOnMax` renvoyés
par l'API intègrent toutes ces contraintes ; si `endsOnMin > endsOnMax`,
`editEndsOn` vaut `false`.

## État de mise en place (`setup`)

| Valeur | Condition |
| --- | --- |
| `missing` | aucune saison |
| `no-current` | des saisons, aucune ne contient `J` |
| `current-draft` | la saison courante est en brouillon |
| `ready` | la saison courante est ouverte |
