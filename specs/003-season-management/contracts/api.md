# Contrat d'API : Gestion des saisons

Mêmes conventions que [001 › api.md](../../001-auth-app-shell/contracts/api.md)
et [002 › api.md](../../002-admin-account-management/contracts/api.md)
(préfixe `/api`, camelCase, `problem+json` en français, CSRF, `401` sans
session, `403` rôle insuffisant). Le schéma OpenAPI produit par `frigg openapi`
fait foi. Dates : chaînes `AAAA-MM-JJ` (`format: date`).

## Types

```text
SeasonActions { editStartsOn: bool, editEndsOn: bool, open: bool,
                duplicate: bool, delete: bool,
                openBlockedReason: string|null }

Season { id: int, label: string, startsOn: date, endsOn: date,
         status: "draft"|"open"|"closed", isCurrent: bool,
         openedAt: datetime|null, createdAt: datetime,
         endsOnMin: date, endsOnMax: date,
         shiftsSeasonLabel: string|null, // saison suivante dont le début suivra la fin
         actions: SeasonActions }

SeasonList { items: Season[],            // de la plus récente à la plus ancienne
             today: date,                // date du jour à Europe/Paris
             currentId: int|null,
             nextStartsOn: date|null,    // début imposé de la prochaine création
             nextEndsOnMin: date|null,   // bornes de la fin de la prochaine création
             nextEndsOnMax: date|null,
             setup: "missing"|"no-current"|"current-draft"|"ready" }
```

Sans saison, `nextStartsOn`, `nextEndsOnMin` et `nextEndsOnMax` valent `null` :
les deux dates sont libres (fin ≥ `today`). Sinon `nextEndsOnMin` vaut
`nextStartsOn` + 1 jour : une saison suivante peut être entièrement passée,
pour combler le trou laissé par une interruption.

## Opérations `admin` (tag `Saisons`)

Aucune opération publique ni `member` n'est ajoutée.

| Opération | Méthode et chemin | Entrée | Réponses |
| --- | --- | --- | --- |
| `list-seasons` | `GET /api/admin/seasons` | — | `200 SeasonList` |
| `get-season` | `GET /api/admin/seasons/{id}` | — | `200 Season` · `404` |
| `create-season` | `POST /api/admin/seasons` | `{ startsOn: date, endsOn: date }` | `201 Season` · `409` liste changée · `422` |
| `update-season` | `PATCH /api/admin/seasons/{id}` | `{ startsOn?: date, endsOn: date }` | `200 Season` · `404` · `409` (close, suivante ouverte, début non modifiable) · `422` |
| `open-season` | `POST /api/admin/seasons/{id}/open` | — | `200 Season` (déjà ouverte : inchangée) · `404` · `409` |
| `duplicate-season` | `POST /api/admin/seasons/{id}/duplicate` | — | `201 Season` (la nouvelle) · `404` · `409` |
| `delete-season` | `DELETE /api/admin/seasons/{id}` | — | `204` · `404` · `409` |

`startsOn` absent de `update-season` = inchangé ; présent alors que
`actions.editStartsOn` est faux → `409`.

## Erreurs de champ (`422`)

| Emplacement | Cas | `message` |
| --- | --- | --- |
| `body.startsOn`, `body.endsOn` | date invalide | « Saisissez une date valide. » |
| `body.endsOn` | fin ≤ début | « La date de fin doit être postérieure à la date de début. » |
| `body.endsOn` | fin < aujourd'hui (première saison, modification) | « La date de fin ne peut pas être passée. » |
| `body.endsOn` | plus de 24 mois | « Une saison ne peut pas durer plus de 24 mois. » |
| `body.endsOn` | saison suivante rendue vide ou trop longue | « Cette date rendrait la saison {libellé} invalide. » |

## Refus métier (`409`)

| Cas | `detail` |
| --- | --- |
| Saison inexistante (`404`) | « Cette saison n'existe plus. » |
| Création avec un `startsOn` différent du début imposé, duplication d'une saison qui n'est plus la dernière | « La liste des saisons a changé. Rechargez la page. » |
| Modification d'une saison close | « Une saison close ne peut plus être modifiée. » |
| Modification de la date de début non autorisée | « La date de début de cette saison ne peut pas être modifiée. » |
| Modification de la fin alors que la suivante est ouverte | « La saison suivante est déjà ouverte : la date de fin ne peut plus changer. » |
| Ouverture d'une saison close | « Une saison close ne peut plus être ouverte. » |
| Ouverture alors que la précédente est en brouillon | « Ouvrez d'abord la saison {libellé}. » |
| Suppression d'une saison qui a été ouverte | « Une saison qui a été ouverte ne peut pas être supprimée. » |
| Suppression d'une saison qui n'est pas la dernière | « Seule la dernière saison peut être supprimée. » |

## Effets de bord

| Opération | Effet |
| --- | --- |
| `update-season` | si la saison suivante est en brouillon, son `startsOn` est décalé au lendemain du nouveau `endsOn` (même transaction) |
| `open-season` | `opened_at` renseigné |
| toute écriture | ligne de journal `INFO` (`action`, `actor_id`, `season_id`, dates) — [research.md › R13](../research.md#r13-journalisation) |

## Changements d'opérations existantes

Aucun. `Deps` gagne un champ `Now func() time.Time` (horloge injectable,
[research.md › R3](../research.md#r3-aujourdhui--fuseau-et-horloge-injectable)),
sans effet sur le contrat.
