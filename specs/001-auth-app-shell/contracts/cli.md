# Contrat CLI : binaire `frigg`

Sorties en français ; erreurs sur la sortie d'erreur, code de retour non nul.
Configuration lue depuis l'environnement et `.env` (voir
[research.md › R10](../research.md#r10-configuration)).

## `frigg` / `frigg serve`

Démarre le serveur : applique les migrations (échec = arrêt avec code 1),
démarre le worker d'e-mails, écoute sur `FRIGG_LISTEN_ADDR`. Arrêt propre sur
`SIGINT`/`SIGTERM` (fin des requêtes en cours, arrêt du worker).

- Variable obligatoire manquante → `configuration invalide : FRIGG_BASE_URL est obligatoire`, code 1.

## `frigg create-admin`

Interactif ; applique d'abord les migrations.

```text
$ frigg create-admin
Nom de l'association : Les Sentiers du Val        ← seulement si non défini
Adresse électronique : claire.dubois@exemple.fr
Mot de passe :                                     ← saisie masquée
Confirmez le mot de passe :                        ← saisie masquée
Administrateur claire.dubois@exemple.fr créé.
```

| Cas | Sortie (stderr) | Code |
| --- | --- | --- |
| Adresse déjà utilisée | `un compte existe déjà avec l'adresse claire.dubois@exemple.fr` | 1 |
| Règle de mot de passe non respectée | une ligne par critère (messages de [api.md](./api.md#messages-de-règle-de-mot-de-passe-422)) | 1 |
| Saisies différentes | `les deux mots de passe ne correspondent pas` | 1 |
| Entrée standard non interactive | `create-admin doit être lancé dans un terminal` | 1 |

Option `-email <adresse>` pour préremplir l'adresse ; le mot de passe n'est
jamais accepté en argument.

## `frigg openapi`

Écrit le schéma OpenAPI 3.1 (JSON) sur la sortie standard, sans base de données
ni configuration SMTP. Utilisé par `bun run gen:api` dans `web/`.

## `frigg backup <fichier>`

Sauvegarde cohérente à chaud via `VACUUM INTO '<fichier>'`. Refuse d'écraser un
fichier existant (`le fichier … existe déjà`, code 1). Rappelle en sortie que le
répertoire `FRIGG_DATA_DIR` doit être sauvegardé à part.
