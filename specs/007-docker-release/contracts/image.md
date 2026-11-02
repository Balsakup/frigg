# Contrat de l'image `ghcr.io/balsakup/frigg`

Interface offerte aux exploitants. Approche :
[research.md › R1, R2, R5, R9](../research.md).

## Étiquettes

| Étiquette | Désigne |
| --- | --- |
| `<version>` (`1.2.0`, `1.3.0-rc.1`) | l'image de cette release, immuable sauf relance de la même release |
| `latest` | la plus haute version stable publiée |
| `frigg:dev` | image locale (`go tool task docker:build`), jamais publiée |

Plateformes : `linux/amd64`, `linux/arm64` (index multi-architecture).

## Exécution

| Élément | Valeur |
| --- | --- |
| Point d'entrée | `/frigg` |
| Commande par défaut | `serve` |
| Autres commandes | `create-admin`, `backup <fichier>`, `openapi`, `healthcheck`, `help` |
| Utilisateur | `65532:65532` (`nonroot`) |
| Répertoire de travail | `/data` |
| Volume | `/data` |
| Port | `8080/tcp` |
| Signal d'arrêt | `SIGTERM` (arrêt propre en 10 s au plus) ; délai d'arrêt recommandé côté moteur : 20 s (`--stop-timeout 20`, `stop_grace_period: 20s`) |
| Vérification de santé | `/frigg healthcheck`, intervalle 30 s, délai 5 s, démarrage 30 s (sondage toutes les 2 s), 3 essais |
| Shell, gestionnaire de paquets | aucun |

## Variables d'environnement

Valeurs par défaut fixées par l'image (modifiables) :

| Variable | Défaut dans l'image |
| --- | --- |
| `FRIGG_DATABASE_PATH` | `/data/frigg.db` |
| `FRIGG_DATA_DIR` | `/data/files` |
| `FRIGG_LISTEN_ADDR` | `:8080` |

Obligatoires, sans valeur par défaut : `FRIGG_BASE_URL`, `FRIGG_SMTP_HOST`,
`FRIGG_MAIL_FROM`. Les autres variables gardent les valeurs par défaut
documentées dans `.env.example`. Un fichier `/data/.env` est lu s'il existe ;
l'environnement du conteneur prime.

## Contenu du volume `/data`

```text
/data/
├── frigg.db          # base SQLite (+ frigg.db-wal, frigg.db-shm)
├── files/            # fichiers déposés
└── backups/          # pour `frigg backup backups/<fichier>` (créé par l'image : pas de shell pour le faire)
```

Propriétaire attendu : UID/GID `65532`. Un volume nommé vide reçoit ces droits
automatiquement ; un répertoire de l'hôte monté doit être préparé
(`chown -R 65532:65532`).

## Durcissement supporté

L'image fonctionne avec toutes ces options à la fois, le volume `/data` restant
accessible en écriture :

- `--read-only`
- `--cap-drop=ALL`
- `--security-opt=no-new-privileges`
- `--user 65532:65532`

## Métadonnées (étiquettes OCI)

| Clé | Valeur (release) | Valeur (locale) |
| --- | --- | --- |
| `org.opencontainers.image.title` | `Frigg` | `Frigg` |
| `org.opencontainers.image.description` | gestion d'association auto-hébergée | idem |
| `org.opencontainers.image.version` | `1.2.0` | `dev-<sha court>[-modifie]` |
| `org.opencontainers.image.revision` | SHA complet de l'étiquette | SHA de `HEAD` |
| `org.opencontainers.image.created` | date de construction (RFC 3339) | idem |
| `org.opencontainers.image.source` | `https://github.com/Balsakup/frigg` | idem |
| `org.opencontainers.image.licenses` | `AGPL-3.0-or-later` | idem |

## Attestations (releases uniquement)

- provenance SLSA et SBOM attachées à l'index de l'image ;
- attestation de provenance signée GitHub, vérifiable par
  `gh attestation verify oci://ghcr.io/balsakup/frigg:<version> --owner Balsakup`.
