# Premier administrateur

Le premier compte administrateur se crée uniquement en ligne de commande,
depuis le serveur. La commande demande l'adresse électronique, le mot de
passe (saisi sans écho) et, si besoin, le nom de l'association.

```sh
docker exec -it frigg /frigg create-admin
```

Les comptes suivants (administrateurs ou membres) sont ensuite invités depuis
l'application : voir [Comptes et invitations](../usage/admin-accounts).

## Commandes du binaire

| Commande | Rôle |
| --- | --- |
| `frigg serve` | Démarre le serveur (commande par défaut) |
| `frigg create-admin` | Crée un administrateur, et le nom de l'association si besoin |
| `frigg backup <fichier>` | Sauvegarde la base (voir [Sauvegarder et restaurer](./backup)) |
| `frigg healthcheck` | Vérifie l'état de santé de l'instance locale (code 0 si saine) |
| `frigg openapi` | Écrit le schéma OpenAPI sur la sortie standard |
