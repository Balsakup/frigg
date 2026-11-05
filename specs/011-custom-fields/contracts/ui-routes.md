# Contrat des écrans : champs personnalisés

## Nouvelle section des paramètres

`settingsSections` (`routes/app/admin/SettingsPage.tsx`) et `web/app/routes.ts`
sous `parametres` reçoivent **Champs personnalisés** entre Apparence et
Paiements.

| Route | Fichier | Contenu |
| --- | --- | --- |
| `/admin/parametres/champs` | `routes/app/admin/settings-custom-fields.tsx` | Section « Champs personnalisés » (US1, US5) |

Fil d'Ariane : « Administration › Paramètres › Champs personnalisés ».

### Contenu

Deux régions titrées (`h2`) : **Champs des membres** (« Informations sur la
personne, conservées d'une saison à l'autre. Exemple : numéro de licence
fédérale. ») et **Champs des adhésions** (« Informations propres à une
saison. Exemple : a payé sa licence fédérale. »). Dans chacune :

- liste ordonnée (`ol`) des champs actifs ; chaque ligne : libellé, type,
  badge de visibilité (« Réservé à l'administration », « Visible par le
  membre », « Rempli par le membre »), badge « Obligatoire », alerte
  `--warn` « 12 fiches sans valeur » si `missingCount > 0` ; actions
  « Monter », « Descendre » (boutons icône avec libellé accessible,
  désactivés aux extrémités, focus conservé sur le bouton après
  déplacement), « Modifier », menu « Plus » (Archiver, Supprimer) ;
- bouton « Ajouter un champ » (désactivé avec explication à 30 champs
  actifs) ;
- état vide : explication et exemple de la portée, bouton « Ajouter un
  champ » ;
- partie repliable « Champs archivés (n) », construite sur `Collapsible` de
  `@base-ui/react` (`Collapsible.Trigger` avec `aria-expanded`,
  `Collapsible.Panel`), habillée avec les tokens, repliée par défaut : libellé, type, nombre de
  valeurs, actions « Rétablir » et « Supprimer ».

### Dialogue « Ajouter un champ » / « Modifier le champ »

`Dialog` base-ui, formulaire `react-hook-form` :

1. **Libellé** (obligatoire, compteur 80) ;
2. **Type** (`RadioGroup` : Texte court, Texte long, Nombre, Date, Oui/non,
   Liste de choix) ; désactivé avec aide « Le type ne peut plus changer : des
   valeurs sont enregistrées. » si `valueCount > 0` ;
3. **Choix** (type Liste de choix) : une ligne par choix, bouton « Ajouter un
   choix », bouton « Retirer » par ligne (refus du serveur affiché sur la
   ligne si le choix est utilisé) ;
4. **Visibilité** (`RadioGroup`, défaut « Réservé à l'administration »), avec
   une phrase d'explication par option ;
5. **Obligatoire** (`Checkbox`, masquée pour Oui/non) ;
6. **Texte d'aide** (facultatif, compteur 200).

Erreurs `422` reliées aux champs (`problem.ts`), notification « Champ
ajouté. » / « Champ enregistré. ».

### Suppression

`ConfirmDialog` danger : « Supprimer le champ « Numéro de licence
fédérale » ? » ; si `valueCount > 0` : « Les 12 valeurs enregistrées seront
définitivement perdues. Pour les conserver, archivez plutôt le champ. » avec
un bouton secondaire « Archiver à la place ». L'archivage n'a pas de
dialogue (réversible).

## Composants partagés (`web/app/customfields/`)

| Composant | Rôle |
| --- | --- |
| `CustomFieldInputs.tsx` | Rendu des `FormField`/`FieldValue` d'un formulaire : `fieldset` « Informations complémentaires » (absent s'il n'y a aucun champ), un contrôle par type (`input` texte, `textarea`, `input inputmode="decimal"`, `input type="date"`, `Checkbox`, `Select` avec option vide « — » si facultatif), libellé, aide et erreur reliées par `aria-describedby`, astérisque + « obligatoire » pour les lecteurs d'écran ; champ non `editable` rendu en lecture seule (texte, pas de contrôle désactivé) |
| `CustomFieldValues.tsx` | Affichage en lecture (liste de définitions `dl`) : nombre au format français, date « 12 mars 2026 », Oui/Non, libellé du choix, « Non renseigné » en `--muted` |
| `CustomFieldFilters.tsx` | Un `Select` par champ oui/non ou liste de choix actif : « Tous », « Oui »/« Non » ou chaque choix + « Non renseigné » |
| `form.ts` | Conversion `FieldValue[]` ↔ valeurs du formulaire (`customFields.f<id>`) et vers `FieldInput[]` (seulement les champs `editable`) |

## Écrans existants modifiés

| Écran | Modification |
| --- | --- |
| `routes/app/admin/member-new.tsx`, `member-detail.tsx` (`members/AdminMemberForm.tsx`) | `CustomFieldInputs` sous l'adresse (définitions : `list-custom-fields` à la création, `customFields` de la fiche ensuite) |
| `routes/app/member/my-member-new.tsx`, `my-member-detail.tsx` (`members/MyMemberForm.tsx`) | Idem avec `list-my-custom-fields` et l'audience membre |
| `routes/app/admin/members.tsx` | `CustomFieldFilters` (portée `member`) à côté du filtre de rattachement ; l'aide de la recherche mentionne les champs texte ; filtres portés par l'URL (`?champ=4:false`) |
| `routes/app/admin/memberships.tsx` | `CustomFieldFilters` (portée `membership`) combinés à la saison, l'état et le paiement ; `?champ=…` |
| `routes/app/admin/membership-detail.tsx` | Partie « Informations complémentaires » : formulaire `CustomFieldInputs` + « Enregistrer » (`update-membership-fields`) pour une adhésion en attente ou validée ; inclus dans le formulaire de validation ; lecture seule si refusée ou annulée |
| `routes/app/admin/membership-new.tsx` | `CustomFieldInputs` (portée `membership`) |
| `routes/app/member/membership-request.tsx` | `CustomFieldInputs` des seuls champs d'adhésion `editable` ; les champs `visible` apparaissent ensuite sur le détail de l'adhésion |
| `routes/app/member/membership-detail.tsx` | Valeurs visibles ; champs `editable` modifiables (`update-my-membership-fields`) tant que l'adhésion est en attente ou validée |
| `lib/member-filters.ts`, `lib/membership-filters.ts` | Lecture et écriture du paramètre `champ` dans l'URL, conversion en `field` pour l'API |

Après chaque écriture : invalidation de `custom-fields` (définitions et
compteurs), de la fiche ou de l'adhésion concernée et des listes.

## Accessibilité (RGAA)

- chaque contrôle a un libellé visible ; l'aide et l'erreur sont reliées ;
- déplacement au clavier : « Monter »/« Descendre », annonce dans la région
  `status` (« Numéro de licence fédérale déplacé en position 2 sur 4. ») ;
- 320 px : les lignes de la liste passent sur deux lignes, actions
  regroupées dans le menu « Plus ».
