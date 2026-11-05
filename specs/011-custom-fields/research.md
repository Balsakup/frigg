# Research: Champs personnalisés des membres et des adhésions

**Feature**: `011-custom-fields` | **Date**: 2026-11-05 | **Spec**: [spec.md](./spec.md)

Aucun point de la spécification ne restait à clarifier. Les choix ci-dessous
partent du code existant (005 fiches, 006 adhésions, 010 paramètres) et de la
constitution **v1.8.0**.

## R1. Stockage : définitions + valeurs typées (EAV à colonnes typées)

**Decision** : migration `00011_custom_fields.sql` avec quatre tables
([data-model.md](./data-model.md)) :

- `custom_fields` : une ligne par définition (`scope` = `member` ou
  `membership`) ;
- `custom_field_choices` : choix des listes de choix ;
- `member_field_values` et `membership_field_values` : au plus une ligne par
  couple (fiche ou adhésion, champ), avec **une colonne par type**
  (`value_text`, `value_long`, `value_number`, `value_date`, `value_bool`,
  `value_choice_id`) et un `CHECK` « exactement une colonne renseignée ».

Absence de ligne = « Non renseigné » (et « Non » pour un oui/non). Une valeur
vidée supprime la ligne.

**Rationale** : des colonnes typées gardent les `CHECK` (longueurs, format de
date, clé étrangère vers le choix), sont lues par sqlc sans désérialisation et
permettent l'index FTS sur le seul texte court (R5). Deux tables de valeurs
plutôt qu'une table polymorphe : clés étrangères réelles avec `ON DELETE
CASCADE` vers `members` et `memberships` (suppression d'une fiche ou d'une
adhésion, Edge Cases), pas de colonne « type d'objet ».

**Alternatives considered** :

- colonne JSON `custom_values` sur `members` et `memberships` : pas de clé
  étrangère vers les choix (FR-006 non garanti par la base), filtrage et
  recherche FTS impossibles sans dénormaliser ;
- `ALTER TABLE ADD COLUMN` dynamique à chaque champ créé : SQL généré à
  l'exécution, interdit par le principe VI ;
- une seule colonne `value TEXT` : perd la validation par la base et oblige à
  exclure les textes longs de l'index par un trigger conditionnel.

## R2. Domaine pur `internal/customfield`

**Decision** : nouveau paquet sans base, comme `internal/member` et
`internal/membership` :

- `Definition`, `Type`, `Visibility`, `Scope` ; `ValidateDefinition`
  (libellé 1–80, aide ≤ 200, 2–50 choix distincts, unicité avec
  `textfold.Fold`) ;
- `Audience` (`Admin` ou `Owner`) et `CanRead(def, audience)`,
  `CanWrite(def, audience)` ;
- `NormalizeValue(def, Input) (Value, error)` : un seul champ de l'entrée
  renseigné et conforme au type, nombre décimal accepté avec virgule ou point,
  texte court ≤ 200, long ≤ 2 000, date `AAAA-MM-JJ` valide, choix appartenant
  au champ ;
- `Apply(defs, current, inputs, audience) (Plan, FieldErrors)` : calcule les
  insertions, mises à jour et suppressions, refuse tout champ inconnu, archivé
  ou non modifiable pour l'audience, puis vérifie les champs obligatoires
  (R4) ;
- `ParseFilter` pour les filtres de liste (R6).

**Rationale** : toutes les règles testées en table sans base ; les handlers
ne font que charger les définitions et appliquer le plan dans la transaction.

**Alternatives considered** : règles dans les handlers : dupliquées entre les
13 opérations qui lisent ou écrivent des valeurs.

## R3. Visibilité : l'audience dépend de l'espace, pas du rôle

**Decision** : les opérations `/admin/...` (rôle `admin`) travaillent avec
l'audience `Admin`, les opérations `/me/...` (rôle `member`) avec l'audience
`Owner`. Les handlers ne testent jamais le rôle de l'appelant :

| Visibilité | `Admin` | `Owner` |
| --- | --- | --- |
| `admin` (Réservé à l'administration) | lecture, écriture | absent |
| `visible` (Visible par le membre) | lecture, écriture | lecture |
| `editable` (Rempli par le membre) | lecture, écriture | lecture, écriture |

En audience `Owner`, les définitions `admin` sont filtrées **dans la
requête** (`visibility <> 'admin'`) et jamais chargées en mémoire du handler :
ni libellé, ni valeur ne peuvent fuiter (FR-020). Une écriture visant un
champ inconnu, archivé, réservé ou seulement visible reçoit le même `422`
« Ce champ n'existe pas ou n'est pas modifiable. », pour ne pas révéler
l'existence d'un champ réservé.

**Rationale** : principe III (fonction d'autorisation unique, pas de test de
rôle dispersé) : le rôle est déjà vérifié par `register`, l'espace détermine
l'audience. Un administrateur qui gère ses propres fiches depuis « Mes
membres » voit ce qu'y voit un membre, comme pour le reste de 005.

**Alternatives considered** : lire le rôle de la session dans les handlers
`/me` pour montrer plus à un administrateur : test de rôle ad hoc, interdit.

## R4. Champs obligatoires

**Decision** : à chaque écriture, pour chaque champ actif obligatoire que
l'audience peut écrire, la valeur finale (celle de la requête si le champ y
figure, sinon la valeur existante) doit être renseignée. Les champs absents
de la requête restent **inchangés** (pas d'effacement implicite des valeurs
réservées quand un membre enregistre). Le type `bool` ne peut pas être
obligatoire (`CHECK`). Pour les adhésions en audience `Admin`, le contrôle
porte sur `validate-membership`, `create-membership` et
`update-membership-fields` ; en audience `Owner`, sur
`create-my-membership` et `update-my-membership-fields`.

**Rationale** : FR-015 et Edge Case « champ obligatoire ajouté après coup »
(exigé au prochain enregistrement, pas rétroactivement).

**Alternatives considered** : envoi de la liste complète et remplacement :
un client membre effacerait les valeurs qu'il ne voit pas.

## R5. Recherche par champ texte court (FR-024)

**Decision** : table FTS5 à contenu externe `member_field_values_fts` sur
`member_field_values(value_text)`, `tokenize="trigram remove_diacritics 1"`,
synchronisée par trois triggers. Seule la colonne `value_text` (texte court)
est indexée ; les textes longs sont dans `value_long`, hors index (principe
VI : une table FTS5 n'indexe qu'une catégorie de champs). `ListMembers`
ajoute à la condition de recherche :

```sql
OR m.id IN (SELECT v.member_id FROM member_field_values v
            JOIN custom_fields cf ON cf.id = v.field_id AND cf.archived_at IS NULL
            WHERE v.id IN (SELECT rowid FROM member_field_values_fts
                           WHERE member_field_values_fts MATCH ?))
```

Les fragments d'une recherche (tous requis) doivent alors tous se trouver
soit dans l'identité de la fiche, soit dans une même valeur ; « Lucas 1234 »
(prénom + licence) ne trouve rien. Limite acceptée et documentée dans
l'aide du champ de recherche.

**Rationale** : mêmes mécanismes que `members_fts` (005 › R6). Les
déclencheurs `ON DELETE CASCADE` (suppression d'une fiche ou d'un champ)
exécutent bien les triggers `AFTER DELETE` dans SQLite, l'index reste
cohérent ; un test le vérifie.

**Alternatives considered** : ajouter les valeurs à `members_fts` : une table
FTS à contenu externe ne peut pas joindre plusieurs tables ; table contentless
mise à jour par le Go : synchronisation hors transaction SQL, fragile.

## R6. Filtres de liste sans SQL dynamique (FR-023)

**Decision** : paramètre de requête répété `field`, format
`<idChamp>:<valeur>` avec `valeur` ∈ `true`, `false`, `empty`, `c<idChoix>`
(par exemple `?field=4:false&field=7:c12`). Le Go valide chaque filtre
(champ actif de la bonne portée, type oui/non ou liste, choix appartenant au
champ ; sinon `422` sur `query.field`), puis passe la liste en JSON à une
requête sqlc **statique** qui la parcourt avec `json_each` :

```sql
AND NOT EXISTS (
  SELECT 1 FROM json_each(CAST(sqlc.narg(field_filters) AS TEXT)) f
  WHERE CASE json_extract(f.value, '$.op')
    WHEN 'true'   THEN NOT EXISTS (… v.value_bool = 1)
    WHEN 'false'  THEN EXISTS (… v.value_bool = 1)
    WHEN 'empty'  THEN EXISTS (… v.field_id = …)
    WHEN 'choice' THEN NOT EXISTS (… v.value_choice_id = json_extract(f.value, '$.choice'))
  END)
```

Vérifié avec sqlc 1.x (moteur SQLite) : la requête compile et produit un
paramètre `sql.NullString`. Au plus 10 filtres par requête.

**Rationale** : principe VI (requêtes écrites dans `queries/`, pas de
concaténation) avec un nombre variable de filtres. Les listes restent
chargées entières puis triées et paginées en Go (005 › R7, 006 › R10).

**Alternatives considered** : un paramètre par filtre (nombre fixe) : borne
arbitraire et requête illisible ; filtrage en Go après chargement : charge
toutes les valeurs de toutes les fiches à chaque liste.

## R7. Contrat des valeurs

**Decision** : une valeur saisie est un objet `{fieldId, text?, date?,
checked?, choiceId?}` (`text` porte la saisie brute d'un texte **ou d'un
nombre** : « 12,5 » est converti par le serveur, sans règle de saisie
recopiée dans le navigateur). En sortie, la valeur est typée (`text`,
`number`, `date`, `checked`, `choiceId`) et la définition
nécessaire à l'affichage accompagne la valeur (`label`, `type`, `help`,
`required`, `editable`, `choices`). Les sorties de fiche et d'adhésion
listent **tous** les champs actifs lisibles par l'audience, renseignés ou non,
dans l'ordre des positions, pour que le formulaire n'ait pas à joindre deux
requêtes. Pour les formulaires de création (pas encore de fiche ni
d'adhésion), `list-my-custom-fields` (membre) et `list-custom-fields`
(administration) fournissent les définitions.

**Rationale** : types générés par `openapi-typescript` exploitables
directement (principe VI), sans chaîne à interpréter côté client.

**Alternatives considered** : `value: string` unique : typage perdu, format
des booléens et des nombres à reconvertir dans le navigateur.

## R8. Écriture des valeurs d'adhésion

**Decision** :

- `create-my-membership`, `create-membership` et `validate-membership`
  acceptent `customFields` dans leur corps, écrits dans la même transaction ;
- nouvelle opération `update-membership-fields` (administration) : adhésion
  `pending` ou `validated`, **quel que soit l'état du paiement** ;
- nouvelle opération `update-my-membership-fields` (membre) : adhésion
  `pending` ou `validated` d'une fiche gérée, champs `editable` seulement ;
- `refused` et `cancelled` : `409` (« Cette adhésion ne peut plus être
  modifiée. »), valeurs toujours renvoyées en lecture (FR-016).

**Rationale** : `update-membership` (006) est bloqué dès que l'adhésion est
payée (`payment_status IS NOT 'paid'`) et touche au tarif et au montant ;
cocher « A payé sa licence fédérale » ne doit dépendre ni de l'un ni de
l'autre (US3, scénario 3).

**Alternatives considered** : élargir `update-membership` : mélangerait une
correction financière et une saisie d'information, avec des préconditions
différentes.

## R9. Définitions : modification, ordre, archivage, suppression

**Decision** :

- `update-custom-field` reçoit la définition complète. Le `type` est
  immuable dès qu'une valeur existe (`409`). Les choix sont envoyés avec leur
  `id` (existants, renommables) ou sans (nouveaux) ; un choix existant absent
  de la liste est retiré, sauf s'il est utilisé (`422`, et `ON DELETE
  NO ACTION DEFERRABLE INITIALLY DEFERRED` sur `value_choice_id` en filet,
  vérifié à la fin de la transaction, voir data-model.md) ;
- `reorder-custom-fields` reçoit la liste ordonnée complète des
  identifiants actifs d'une portée ; toute liste incomplète ou périmée est
  refusée (`409`) ; l'interface propose des boutons « Monter » et
  « Descendre » (clavier et souris), sans glisser-déposer ;
- `archive-custom-field` pose `archived_at` ; `restore-custom-field` le
  retire, place le champ en fin de liste et refuse au-delà de 30 actifs ;
- l'unicité du libellé est contrôlée avec `textfold.Fold` sur **tous** les
  champs de la portée, archivés compris, pour qu'un rétablissement ne crée
  jamais de doublon ;
- `delete-custom-field` supprime le champ ; ses valeurs et ses choix
  partent par `ON DELETE CASCADE` (index FTS nettoyé par trigger).

**Rationale** : FR-005 à FR-010 ; boutons de déplacement = motif accessible
sans dépendance nouvelle (liste fermée du frontend).

**Alternatives considered** : bibliothèque de glisser-déposer : dépendance
nouvelle, accessibilité clavier à reconstruire ; positions éditées champ par
champ : conflits d'ordre entre deux administrateurs.

## R10. Compteurs de la section (FR-026, US5)

**Decision** : `list-custom-fields` renvoie pour chaque champ `valueCount`
(nombre de valeurs, pour le dialogue de suppression) et, pour un champ
obligatoire, `missingCount` : fiches sans valeur (champ de membre) ou
adhésions `pending`/`validated` sans valeur de la saison en cours
(`season.IsCurrent`) ; `null` s'il n'y a pas de saison en cours. Requêtes
d'agrégat `GROUP BY field_id` sur les tables de valeurs.

**Rationale** : quelques milliers de lignes au plus ; calcul à la lecture,
sans compteur stocké à maintenir.

## R11. Journaux

**Decision** : `custom_field.create|update|reorder|archive|restore|delete`
avec `actor_id`, `field_id`, `scope` ; les écritures de valeurs sont
couvertes par les actions existantes (`member.update`, `membership.*`) avec
un attribut `custom_fields` = nombre de champs modifiés. Aucune valeur ni
libellé de champ dans les journaux (un libellé peut être sensible :
« Allergies »).

## R12. Tests

**Decision** :

- `internal/customfield` : tests en table (types, bornes, virgule décimale,
  visibilités × audiences, obligatoire, choix, filtres) ;
- API (httptest, base isolée) : CRUD des définitions, `409` sur changement de
  type, choix utilisé non retirable, limite de 30, ordre périmé ; écriture des
  valeurs par audience ; **fuite** : en membre, la réponse JSON de
  `get-my-member` et `get-my-membership` ne contient ni l'identifiant ni le
  libellé d'un champ réservé ; filtres et recherche ; suppression en cascade
  et cohérence de `member_field_values_fts` ;
- matrice d'autorisation : 10 opérations ajoutées (8 `admin`, 2 `member`),
  couvertes automatiquement par `TestAccessMatrix` ;
- migration appliquée sur une base vierge et sur une base `00010` peuplée ;
- Playwright + axe : `custom-fields.spec.ts` (définition, saisie admin et
  membre, filtre licence non payée).
