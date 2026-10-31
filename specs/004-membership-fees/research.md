# Recherche : Gestion des tarifs de l'adhésion

Décisions techniques de la fonctionnalité 004. Elles prolongent celles de
[003 › research.md](../003-season-management/research.md), qui restent valables
(horloge injectable `Deps.Now`, état « close » calculé, écritures sérialisées
par `Store.WriteTx`, domaine pur dans `internal/season`, journalisation des
écritures).

## R1. Stockage d'un tarif

- **Décision** : table `membership_fees` (migration `00005_membership_fees.sql`)
  rattachée à `seasons` par `season_id` (`REFERENCES seasons(id) ON DELETE
  CASCADE`), avec `label`, `amount_cents` (`INTEGER`), `description` (`NULL`
  si absente), `created_at`, `updated_at`. Contraintes `CHECK` sur la longueur
  du libellé et de la description et sur l'intervalle du montant, en filet de
  sécurité derrière la validation de l'API.
- **Rationale** : le montant en centimes entiers évite toute erreur d'arrondi
  (les futures adhésions et la comptabilité additionneront ces montants) ; la
  cascade réalise FR-014 sans code (`foreign_keys=ON` est activé sur chaque
  connexion, contraintes techniques de la constitution).
- **Nouvelle migration** : la table dépend de `seasons` ; ajouter une
  migration reste la règle par défaut.
- **Alternatives écartées** : montant en `REAL` (arrondis binaires) ; montant
  en `TEXT` décimal (comparaisons et sommes malcommodes) ; tarifs en JSON dans
  `seasons` (ni contrainte, ni requête possible par tarif, et les adhésions
  devront référencer un tarif).

## R2. Unicité du libellé sans casse ni accents

- **Décision** : l'unicité dans une saison est vérifiée en Go, dans la
  transaction d'écriture, en comparant les libellés repliés par un nouveau
  package `internal/textfold` (`Fold` : suppression des espaces en début et
  fin, réduction des espaces intérieurs consécutifs, décomposition NFD par
  `golang.org/x/text/unicode/norm`, suppression des diacritiques, minuscules).
  Aucune colonne repliée ni index unique en base. Le libellé est enregistré
  tel que saisi, espaces de début et de fin retirés.
- **Rationale** : la constitution (principe VI) impose `internal/textfold`
  pour les comparaisons d'unicité et interdit les colonnes repliées dupliquées ;
  les écritures sérialisées (une connexion, `BEGIN IMMEDIATE`) rendent la
  vérification sans course : deux ajouts simultanés du même libellé donnent
  un succès et une erreur `422` (edge case de la spec). Au plus 20 tarifs par
  saison : la relecture est négligeable.
- **Dépendance** : `golang.org/x/text` est déjà dans le graphe de modules
  (indirect) et figure dans la liste fermée de la constitution
  (« Normalisation Unicode ») ; il devient direct. Premier usage du package
  `textfold`, prévu par la constitution mais pas encore créé.
- **Alternatives écartées** : `COLLATE NOCASE` (ignore la casse ASCII
  seulement, pas les accents) ; colonne `label_key` avec index unique
  (interdite par la constitution) ; fonction SQL personnalisée enregistrée
  dans modernc (couplage au pilote, non lisible par sqlc).

## R3. Saisie et format du montant

- **Décision** : l'API reçoit le montant en texte (`amount`, par exemple
  `"15,50"`), l'analyse côté serveur (fonction pure `season.ParseAmount`) et
  renvoie `amountCents` (entier). Format accepté : chiffres, séparateur décimal
  virgule ou point, au plus deux décimales ; les espaces (y compris insécables)
  sont ignorés (`"1 000"`), de même que les zéros de tête (`"0030"`) ; plus de
  5 chiffres entiers significatifs donne directement « trop élevé », sans
  conversion ni risque de débordement. Erreurs `422` sur `body.amount` : format invalide
  (dont montant négatif) et montant supérieur à 10 000 €. Le frontend utilise
  un champ `type="text"` avec `inputmode="decimal"` et affiche les montants avec
  `Intl.NumberFormat("fr-FR", { style: "currency", currency: "EUR" })`, ou
  « Gratuit » pour 0.
- **Rationale** : le serveur fait autorité sur l'analyse (principe VI) : le
  frontend n'a ni à convertir des euros en centimes ni à valider le format ;
  la virgule française est acceptée telle quelle. `<input type="number">` est
  écarté : sa gestion de la virgule dépend de la langue du navigateur et il
  renvoie une valeur vide sur une saisie invalide, ce qui empêcherait d'afficher
  l'erreur serveur.
- **Alternatives écartées** : `amountCents` en entrée (conversion et contrôle
  de format à écrire côté frontend) ; nombre décimal JSON (`15.5` : arrondis
  flottants, virgule impossible).

## R4. Règles des tarifs dans le domaine

- **Décision** : nouveau fichier `internal/season/fee.go`, pur comme le reste
  du package : `ParseAmount`, `ValidateFee` (plafond de 20 à la création,
  puis libellé et description nettoyés, libellé vide refusé, montant analysé
  et unicité parmi les autres tarifs de la saison ; toutes les erreurs de
  champ sont cumulées dans un `FieldErrors` pour un seul `422`, FR-017 ; les
  longueurs maximales sont validées par huma via `maxLength`),
  `FeeActionsFor(state, count)` et les constantes `MaxFees`, `MaxAmountCents`,
  `MaxFeeLabel`, `MaxFeeDescription`. Erreurs typées par champ
  (`FieldError` existant) ou métier, traduites par l'API.
- **Rationale** : l'éditabilité dépend de l'état de la saison, calculé par ce
  package ; les règles se testent en table, sans base. Un package séparé
  dépendrait de `season` pour une poignée de fonctions.
- **Alternatives écartées** : règles dans les handlers (non testables
  isolément, dupliquées entre ajout et modification).

## R5. Éditabilité selon l'état de la saison

- **Décision** : les tarifs sont modifiables (ajout, modification, suppression)
  tant que l'état calculé de la saison n'est pas `closed` (FR-007 à FR-009).
  Chaque écriture relit la saison dans la transaction et calcule son état avec
  la date du jour fournie par `Deps.Now` ; une saison close donne un `409`
  « Une saison close ne peut plus être modifiée. ». L'API expose
  `actions: { add, edit, delete }` (booléens) : `edit` et `delete` valent
  « saison non close », `add` y ajoute « moins de 20 tarifs ».
- **Rationale** : même modèle que les actions des saisons (003 › R8) ; le
  frontend n'affiche que les actions annoncées et ne réimplémente pas l'état.
  La clôture à minuit pendant la saisie est couverte par la relecture
  (edge case de la spec).

## R6. Opérations d'API imbriquées sous la saison

- **Décision** : quatre opérations `admin` sous `/admin/seasons/{id}/fees` :
  liste, création, modification (`PUT`, les trois champs sont fournis à chaque
  fois), suppression. Un tarif d'une autre saison que celle du chemin est
  traité comme inexistant (`404`). La liste renvoie les tarifs triés (montant
  décroissant, puis libellé replié, puis identifiant) et les `actions`.
  `get-season` est inchangé.
- **Rationale** : le chemin porte l'appartenance à la saison ; la liste se
  charge et s'invalide indépendamment de la fiche, dont les actions ne
  dépendent pas des tarifs (l'ouverture n'exige pas de tarif, FR-015). Le tri
  est fait en Go après repli (cohérent avec l'unicité ; SQLite ne trie pas
  sans accents).
- **Alternatives écartées** : tarifs embarqués dans `Season` (alourdit la
  liste des saisons et oblige à invalider toutes les fiches à chaque écriture
  de tarif) ; `PATCH` partiel (formulaire toujours complet, sans intérêt).

## R7. Duplication des tarifs

- **Décision** : `duplicate-season` recopie les tarifs de la saison source dans
  la même transaction que la création de la saison, par une requête
  `INSERT … SELECT` (`CopyMembershipFees`). La ligne de journal
  `season.duplicate` gagne `fees_copied` (nombre de tarifs recopiés). La
  création sans duplication ne crée aucun tarif (FR-013).
- **Rationale** : atomicité (une saison dupliquée n'existe jamais sans ses
  tarifs) ; les copies sont des lignes indépendantes (FR-012). Les règles
  (unicité, plafond) étant respectées dans la source, la copie n'a rien à
  revalider.
- **Cas de la copie close** : une duplication entièrement passée (comblement
  d'une interruption) porte les tarifs recopiés, en lecture seule
  (edge case de la spec).

## R8. Message « saison non proposée à l'adhésion »

- **Décision** : affiché par la section des tarifs quand `actions.edit` est
  vrai (saison non close) et que la liste est vide : `Alert` `info` « Cette
  saison ne sera pas proposée à l'adhésion tant qu'elle n'a aucun tarif. ».
  Aucun champ dédié dans l'API.
- **Rationale** : la condition se lit directement dans la réponse (liste vide
  et saison modifiable) sans règle métier à dupliquer ; la règle d'exclusion
  elle-même appartiendra à la fonctionnalité d'adhésion (Assumptions de la
  spec).

## R9. Écran

- **Décision** : composant `SeasonFees.tsx` rendu dans la fiche de la saison
  (`season-detail.tsx`), sous la liste de définitions, avec son propre titre
  `h2` « Tarifs de l'adhésion ». Un seul bouton « Ajouter un tarif » à la fois :
  dans l'en-tête de section s'il existe des tarifs, sinon dans l'état vide.
  Liste en `ui/Table` (Tarif · Montant ·
  Actions), description sous le libellé. Ajout et modification dans un
  `Dialog` base-ui partagé (`FeeDialog`) ; suppression via `ConfirmDialog`
  `danger` (principe IV). Erreurs `422` par l'adaptateur
  `applyProblemToForm` ; `404`/`409` dans une `Alert` et invalidation de la
  liste des tarifs et de la fiche.
- **Rationale** : mêmes motifs que la modification des dates (003 › R12) ;
  trois champs tiennent dans un dialogue, sans nouvelle route.
- **Alternatives écartées** : édition en ligne dans le tableau (motif clavier
  complexe, peu lisible à 320 px) ; page dédiée par tarif (navigation inutile).

## R10. Journalisation

- **Décision** : une ligne `INFO` par écriture réussie : `action`
  (`fee.create`, `fee.update`, `fee.delete`), `actor_id`, `season_id`,
  `fee_id`, et pour la création et la modification le montant en centimes.
  Le libellé n'est pas journalisé (texte libre).
- **Rationale** : cohérent avec 003 › R13 ; un texte libre saisi n'a pas sa
  place dans les journaux.

## R11. Requêtes sqlc

- **Décision** : `internal/database/queries/membership_fees.sql` :
  `ListMembershipFees` (par saison), `CreateMembershipFee`,
  `UpdateMembershipFee` (`:execrows`, conditionnée à `season_id`),
  `DeleteMembershipFee` (`:execrows`, idem), `CopyMembershipFees`
  (`:execrows`). Commentaires de requêtes en ASCII uniquement.

## R12. Montant de renouvellement

- **Décision** (2026-10-31, après essai avec des tarifs réels) : un tarif peut
  porter un second montant, celui du renouvellement, activé par la case
  « Tarif de renouvellement différent ». Colonne `renewal_amount_cents`
  (`NULL` = même montant pour toute adhésion), ajoutée par une nouvelle
  migration `00006_membership_fee_renewal.sql` (`ALTER TABLE … ADD COLUMN`).
  API : entrée `hasRenewalAmount` (booléen) et `renewalAmount` (texte analysé
  comme `amount`) ; sortie `renewalAmountCents` (entier ou `null`). Case
  cochée et montant vide → `422` « Saisissez le montant du renouvellement. ».
  Aucune relation imposée entre les deux montants. Recopié par la
  duplication.
- **Rationale** : l'association de l'utilisateur distingue première adhésion
  et renouvellement pour chaque pratique ; deux montants par tarif évitent de
  dédoubler chaque ligne et laissent l'écart varier d'un tarif à l'autre. Le
  booléen explicite distingue « case cochée, montant oublié » (erreur) de
  « pas de montant de renouvellement ». L'association existant avant Frigg,
  le renouvellement sera déclaré par l'adhérent et validé par un
  administrateur (fonctionnalité d'adhésion), qui stockera le type retenu pour
  les statistiques.
- **Nouvelle migration plutôt que modifier `00005`** : les bases de
  développement ont déjà appliqué `00005` et portent des tarifs réels ; goose
  ne rejouerait pas une migration modifiée.
- **Alternatives écartées** : supplément « première adhésion » global à la
  saison (écart unique imposé à tous les tarifs) ; deux tarifs distincts par
  pratique (lignes dédoublées, lien entre elles perdu) ; `renewalAmount` vide
  interprété comme absent (une case cochée sans montant passerait
  silencieusement).
