# Modèle de données : Adhésions

Complète [005 › data-model.md](../005-member-management/data-model.md) et
[004 › data-model.md](../004-membership-fees/data-model.md).
Décisions : [research.md](./research.md).

Migration unique `00008_memberships.sql` (tables `memberships` et
`payment_settings`). Commentaires SQL des requêtes sqlc en ASCII.

## Table `memberships`

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `id` | `INTEGER` | `PRIMARY KEY` | Identifiant |
| `member_id` | `INTEGER` | `NOT NULL REFERENCES members(id) ON DELETE CASCADE` | Fiche qui adhère |
| `season_id` | `INTEGER` | `NOT NULL REFERENCES seasons(id) ON DELETE RESTRICT` | Saison |
| `fee_id` | `INTEGER` | `NOT NULL REFERENCES membership_fees(id) ON DELETE RESTRICT` | Tarif choisi puis retenu (même saison, contrôlé par le domaine) |
| `status` | `TEXT` | `NOT NULL CHECK (status IN ('pending','validated','refused','cancelled'))` | État |
| `declared_kind` | `TEXT` | `NULL CHECK (declared_kind IN ('first','renewal'))` | Type déclaré par le compte ; `NULL` si créée par un administrateur |
| `kind` | `TEXT` | `NULL CHECK (kind IN ('first','renewal'))` | Type retenu à la validation |
| `amount_cents` | `INTEGER` | `NULL CHECK (amount_cents BETWEEN 0 AND 1000000)` | Montant appliqué, figé à la validation |
| `reason` | `TEXT` | `NULL CHECK (reason IS NULL OR length(reason) BETWEEN 1 AND 500)` | Motif de refus ou d'annulation |
| `requested_by` | `INTEGER` | `NULL REFERENCES accounts(id) ON DELETE SET NULL` | Compte qui a déposé la demande |
| `created_by` | `INTEGER` | `NULL REFERENCES accounts(id) ON DELETE SET NULL` | Administrateur qui l'a créée directement |
| `requested_at` | `TEXT` | `NOT NULL` | Date de la demande ou de la création (UTC) |
| `decided_by` | `INTEGER` | `NULL REFERENCES accounts(id) ON DELETE SET NULL` | Administrateur qui a validé ou refusé |
| `decided_at` | `TEXT` | `NULL` | Date de la décision |
| `cancelled_by` | `INTEGER` | `NULL REFERENCES accounts(id) ON DELETE SET NULL` | Auteur de l'annulation |
| `cancelled_at` | `TEXT` | `NULL` | Date de l'annulation |
| `payment_reference` | `TEXT` | `NULL UNIQUE CHECK (length(payment_reference) BETWEEN 8 AND 16)` | Référence de paiement ([R4](./research.md#r4-référence-de-paiement)) |
| `payment_status` | `TEXT` | `NULL CHECK (payment_status IN ('due','announced','paid'))` | `NULL` = pas de paiement (non validée ou gratuite) |
| `announced_method` | `TEXT` | `NULL CHECK (announced_method IN ('cash','cheque','transfer'))` | Moyen choisi par le compte |
| `paid_method` | `TEXT` | `NULL CHECK (paid_method IN ('cash','cheque','transfer'))` | Moyen constaté |
| `paid_on` | `TEXT` | `NULL CHECK (paid_on IS NULL OR date(paid_on) = paid_on)` | Date de réception |
| `paid_by` | `INTEGER` | `NULL REFERENCES accounts(id) ON DELETE SET NULL` | Administrateur qui a constaté |
| `paid_at` | `TEXT` | `NULL` | Date d'enregistrement de la constatation |
| `payment_note` | `TEXT` | `NULL CHECK (payment_note IS NULL OR length(payment_note) BETWEEN 1 AND 200)` | Commentaire (n° de chèque…) |
| `updated_at` | `TEXT` | `NOT NULL` | Dernière modification |

Table `STRICT`. `CHECK` de cohérence (filets derrière le domaine) :

- `status <> 'pending' OR (kind IS NULL AND amount_cents IS NULL)` et
  `status <> 'validated' OR (kind IS NOT NULL AND amount_cents IS NOT NULL)`
  (une adhésion annulée après validation conserve `kind` et `amount_cents`) ;
- `payment_status IS NULL OR (status IN ('validated', 'cancelled') AND amount_cents > 0 AND payment_reference IS NOT NULL)` ;
- `status <> 'validated' OR amount_cents = 0 OR payment_status IS NOT NULL` ;
- `(payment_status IS 'paid') = (paid_method IS NOT NULL AND paid_on IS NOT NULL)` ;
- `payment_status IS NOT 'announced' OR announced_method IS NOT NULL` ;
- `status NOT IN ('refused') OR reason IS NOT NULL` ;
- `requested_by IS NULL OR declared_kind IS NOT NULL`.

Une adhésion annulée garde son `payment_status` d'origine pour l'historique
mais n'entre dans aucun chiffre (seules les `validated` comptent).

Index :

- `memberships_one_active` : `UNIQUE (member_id, season_id) WHERE status IN ('pending','validated')` (FR-003) ;
- `memberships_season_status` sur `(season_id, status)` (liste, chiffres, aide) ;
- `memberships_member_id` sur `member_id` (fiche, « Mes adhésions ») ;
- `memberships_fee_id` sur `fee_id` (tarif utilisé, R9) ;
- `memberships_status_requested` sur `(status, requested_at)` (demandes en attente, tableau de bord).

## Table FTS `memberships_fts`

```text
memberships_fts USING fts5(payment_reference,
                           content='memberships', content_rowid='id',
                           tokenize="trigram remove_diacritics 1")
```

Triggers `memberships_fts_ai` (insertion), `memberships_fts_ad` (suppression),
`memberships_fts_au` (`AFTER UPDATE OF payment_reference`), sur le modèle de
`members_fts`. Une référence `NULL` est indexée comme absente. Recherche par
fragment de référence (principe VI, [R10](./research.md#r10-liste-recherche-et-chiffres-administration)).

## Table `payment_settings` (singleton)

| Colonne | Type | Contraintes | Rôle |
| --- | --- | --- | --- |
| `id` | `INTEGER` | `PRIMARY KEY CHECK (id = 1)` | Ligne unique, créée par la migration |
| `cash_enabled` | `INTEGER` | `NOT NULL DEFAULT 0 CHECK (cash_enabled IN (0,1))` | Espèces activées |
| `cash_instructions` | `TEXT` | `NULL`, ≤ 1 000 | Lieu, permanences… |
| `cheque_enabled` | `INTEGER` | idem | Chèque activé |
| `cheque_payee` | `TEXT` | `NULL`, ≤ 120 | Ordre du chèque |
| `cheque_address` | `TEXT` | `NULL`, ≤ 300 | Adresse d'envoi (multiligne) |
| `cheque_instructions` | `TEXT` | `NULL`, ≤ 1 000 | |
| `transfer_enabled` | `INTEGER` | idem | Virement activé |
| `transfer_holder` | `TEXT` | `NULL`, ≤ 120 | Titulaire du compte |
| `transfer_iban` | `TEXT` | `NULL CHECK (transfer_iban IS NULL OR transfer_iban GLOB '[A-Z][A-Z][0-9][0-9]*')`, 15 à 34 | IBAN normalisé (sans espaces, majuscules) |
| `transfer_bic` | `TEXT` | `NULL`, 8 ou 11 | BIC normalisé |
| `transfer_instructions` | `TEXT` | `NULL`, ≤ 1 000 | |
| `updated_at` | `TEXT` | `NOT NULL` | |

`CHECK` : `cash_enabled = 0 OR cash_instructions IS NOT NULL` ;
`cheque_enabled = 0 OR cheque_payee IS NOT NULL` ;
`transfer_enabled = 0 OR (transfer_holder IS NOT NULL AND transfer_iban IS NOT NULL)`.
Champs vides après nettoyage → `NULL`.

## Règles (contrôlées par `internal/membership`)

| Règle | Détail | Exigence |
| --- | --- | --- |
| Saison ouverte à l'adhésion | état `open` (non close) et au moins un tarif ; création par le compte ou l'administrateur | FR-006, FR-014 |
| Tarif | appartient à la saison de l'adhésion | FR-001, FR-012 |
| Unicité | une adhésion `pending` ou `validated` par fiche et saison | FR-003 |
| Montant appliqué | `AppliedAmount(fee, kind)` à la validation, à la création directe et à la correction ; figé ensuite | FR-004, FR-016 |
| Motif | obligatoire (1 à 500 caractères, espaces nettoyés) pour refuser et pour annuler (administration) | FR-012, FR-015 |
| Annulation par le compte | seulement `pending` | FR-009 |
| Correction | `validated`, `payment_status` ≠ `paid`, saison non close | FR-016 |
| Annulation par l'administrateur | `validated`, saison non close | FR-015 |
| Décision sur saison close | validation et refus d'une demande `pending` admis | FR-017 |
| Choix du moyen | `validated`, `payment_status` ∈ {`due`, `announced`}, moyen activé | FR-022b |
| Constatation | `payment_status` ∈ {`due`, `announced`} ; date ≤ aujourd'hui et ≥ date de validation ; moyen parmi les trois (activé ou non) ; commentaire ≤ 200 | FR-022d |
| Règlement à la création directe | date ≤ aujourd'hui et ≥ date de début de la saison ; mêmes règles de moyen et de commentaire | FR-022d, US6 §4 |
| Annulation de la constatation | `payment_status = paid` ; retour à `announced` si `announced_method`, sinon `due` | US4 §10 |
| Paramètres | voir [R7](./research.md#r7-paramètres-des-moyens-de-paiement) | FR-022g |

## Attributs calculés (jamais stockés)

| Attribut | Calcul | Exigences |
| --- | --- | --- |
| `displayAmountCents` (demande en attente) | `AppliedAmount(tarif actuel, declared_kind)` | US1 §2, spec › tarif modifié |
| `amounts` (chaque tarif exposé) | `{ first: AppliedAmount(tarif, first), renewal: AppliedAmount(tarif, renewal) }` | US1 §2, US2 §4 (principe VI) |
| `previousSeason` | `yes` / `no` / `unknown` ([R6](./research.md#r6-aide--adhérent-la-saison-précédente-dans-frigg-)) | FR-013 |
| `actions` | `membership.ActionsFor` selon l'espace | FR-009, FR-015 à FR-017 |
| `paymentMethods` (espace membre) | moyens activés et leurs informations, seulement si à payer | FR-022b |
| `seasonLabel` | libellé calculé de la saison (003) | partout |

## Transitions

```text
                     create-my-membership
 (aucune) ───────────────────────────────► pending ──refuse──► refused
    │                                       │   │
    │ create-membership                     │   └─cancel-my-membership─► cancelled
    ▼                                       ▼ validate-membership
 validated ◄────────────────────────────────┘
    │  ├─ update-membership (tarif, type ; non payée, saison non close)
    │  └─ cancel-membership (motif ; saison non close) ──► cancelled
    │
    └─ paiement (amount_cents > 0) :
         due ──announce-my-payment──► announced ──announce (autre moyen)──┐
          │                              │   ▲─────────────────────────────┘
          └────record-membership-payment─┴──► paid
                                              │ cancel-membership-payment
                                              ▼
                                  announced (si moyen annoncé) / due
```

Une correction qui fait passer le montant de 0 à un montant non nul attribue
une référence (si absente) et `payment_status = due` ; de non nul à 0 :
`payment_status = NULL`, moyen annoncé effacé, référence conservée.

## Entités existantes touchées

- **Membre** (005) : sa suppression par un compte est refusée s'il porte une
  adhésion `pending` ou `validated` ; par un administrateur, elle emporte ses
  adhésions (`CASCADE`).
- **Tarif** (004) : non supprimable dès qu'une adhésion le référence ;
  `inUse` exposé.
- **Saison** (003) : inchangée ; `RESTRICT` en filet.
- **Compte** (001/002) : sa suppression met à `NULL` les colonnes d'auteur ;
  ses fiches et leurs adhésions suivent le choix de 005.
