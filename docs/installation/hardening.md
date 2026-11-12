# Durcissement

L'image fonctionne avec un système de fichiers racine en lecture seule et sans
aucun privilège ; seul `/data` doit être accessible en écriture.

| Docker | Compose | Kubernetes (`securityContext`) |
| --- | --- | --- |
| `--read-only` | `read_only: true` | `readOnlyRootFilesystem: true` |
| `--cap-drop=ALL` | `cap_drop: [ALL]` | `capabilities: { drop: [ALL] }` |
| `--security-opt=no-new-privileges` | `security_opt: [no-new-privileges:true]` | `allowPrivilegeEscalation: false` |
| (image : utilisateur 65532) | | `runAsNonRoot: true`, `runAsUser: 65532`, `fsGroup: 65532` |

## Vérifier l'origine d'une image

Chaque image publiée porte une attestation de provenance signée et un SBOM.
Pour vérifier qu'une image vient bien du dépôt officiel :

```sh
gh attestation verify oci://ghcr.io/balsakup/frigg:1.0.0 --owner Balsakup
```
