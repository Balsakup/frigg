# Contrat API : point de santé

Une seule opération, publique au titre du principe II (constitution 1.5.0).
Approche : [research.md › R8](../research.md#r8-point-de-santé).

## `GET /api/health` — `get-health`

| Élément | Valeur |
| --- | --- |
| Niveau d'accès | `public` (`register(api, auth.Public, …)`) |
| Tag OpenAPI | `Santé` |
| Session | ni lue ni créée (aucun cookie attendu) |
| Limitation de débit | aucune |
| Corps de requête | aucun |
| En-têtes de réponse | `Cache-Control: no-store` + en-têtes de sécurité habituels |

### Réponses

**200** — l'instance sert des requêtes et la base répond.

```json
{ "status": "ok" }
```

**503** — la base ne répond pas dans le délai (2 s) ou renvoie une erreur.

```json
{
  "title": "Service Unavailable",
  "status": 503,
  "detail": "Service indisponible."
}
```

Le corps ne contient jamais : nom ou donnée de l'association, version,
configuration, chemin de fichier, message d'erreur SQLite.

### Journalisation

| Cas | Ligne | Niveau |
| --- | --- | --- |
| Réponse < 400 | `requête traitée` (méthode, chemin, statut, durée, identifiant) | `debug` |
| Réponse ≥ 400 | `requête traitée` | `info` |
| Base en échec | `vérification de santé en échec` + `error` | `warn` |

### Tests attendus

- `TestAccessMatrix` : `get-health` figure dans `allowedPublic` ; aucune autre
  opération publique ajoutée.
- Appel anonyme → `200`, corps exactement `{"status":"ok"}`, `no-store`.
- Base fermée → `503`, corps sans détail interne, ligne `warn` dans les
  journaux.
- Appel réussi → aucune ligne de niveau `info` pour ce chemin.
- `POST /api/health` → réponse d'erreur (`404` JSON du routeur, la route
  `/` du SPA captant les autres méthodes), jamais `2xx`, aucune modification
  d'état.
