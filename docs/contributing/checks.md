# Contrôles avant fusion

```sh
go tool task check
```

Cette commande rejoue les contrôles exigés par la constitution, qui doivent
tous passer avant une fusion :

- compilation du binaire avec `CGO_ENABLED=0` ;
- `go vet ./...` et `go test ./...` ;
- régénération du code sqlc, du schéma OpenAPI et des types TypeScript, sans
  aucune différence avec le code commité ;
- dans `web/` : vérification des types, Biome, Vitest et compilation ;
- tests de bout en bout Playwright, avec une analyse d'accessibilité axe sur
  les parcours touchés : aucune violation de niveau `serious` ou `critical`.

La CI rejoue ces contrôles sur chaque Pull Request (workflow `CI`).

## Ce qui est attendu d'une modification

- Des tests pour toute modification de comportement. Les tests d'API
  utilisent une base SQLite isolée par test.
- Les règles d'autorisation couvertes : pour chaque opération, un accès
  anonyme est refusé, un membre est refusé si l'opération est réservée aux
  administrateurs, et un administrateur est accepté.
- Les migrations ne modifient jamais une migration déjà publiée : on en
  ajoute une nouvelle.
- La [documentation](./documentation) mise à jour si le comportement visible
  change.

## Documentation

Une Pull Request qui modifie `docs/` déclenche aussi le workflow
`Documentation`, qui construit le site et échoue sur un lien interne cassé.
En local : `go tool task docs:build`.
