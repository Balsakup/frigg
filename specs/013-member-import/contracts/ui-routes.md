# Contrat des écrans : importation des membres

Shell, composants `ui/*`, tokens et base-ui de l'artefact de référence
(principe IV). Aucune entrée de navigation nouvelle : l'import se trouve
dans la rubrique « Membres » de l'administration.

| Route | Fichier | Contenu |
| --- | --- | --- |
| `/admin/membres` | `routes/app/admin/members.tsx` | action « Importer » ajoutée à l'en-tête de page (icône `Upload`, variante secondaire) |
| `/admin/membres/importer` | `routes/app/admin/member-import.tsx` | assistant d'import (US1 à US4) |

Dans `web/app/routes.ts`, `route("importer", …)` est déclarée avant
`route(":id", …)` sous `membres`. Fil d'Ariane : Administration › Membres ›
Importer. Titre de page : « Importer des membres ».

## Assistant

Étapes affichées en liste ordonnée (`ol`, `aria-label="Étapes de
l'import"`), l'étape courante portant `aria-current="step"` : **1. Fichier**,
**2. Colonnes**, **3. Aperçu**, **4. Rapport**. L'étape vit dans l'état du
composant (pas dans l'adresse) ; à chaque changement, le focus va sur le
`h2` de l'étape. Quitter la page à l'étape 2 ou 3 ne demande pas de
confirmation (rien n'est écrit).

### 1. Fichier

- Introduction : déroulé en quatre étapes ; « Seul le format CSV est
  accepté » avec un `Collapsible` « Comment enregistrer mon tableur en
  CSV ? » (Excel : Fichier › Enregistrer sous › CSV UTF-8 ; LibreOffice :
  Fichier › Enregistrer sous › Texte CSV) ; champs exigés : prénom, nom,
  date de naissance ; formats de date acceptés ; les fiches importées sont
  créées « Sans compte » et aucun e-mail n'est envoyé ; les personnes déjà
  présentes sont ignorées.
- Lien-bouton « Télécharger le modèle » (icône `Download`).
- Zone de dépôt : `input type="file" accept=".csv,text/csv"` étiqueté
  « Fichier CSV », habillé en zone de glisser-déposer (le dépôt au clavier
  passe par l'input). Le choix d'un fichier lance `analyze` ; bouton occupé
  pendant l'analyse.
- Erreur `422 form.file` : `Alert` `--danger` sous la zone, reliée par
  `aria-describedby`, focus sur la zone.

### 2. Colonnes

- Rappel du fichier : nom, « 214 lignes », encodage et séparateur détectés
  en texte secondaire.
- Tableau (cartes sous 640 px) : une ligne par colonne du fichier :
  intitulé (`label`), exemples (`samples`, `--muted`), et un `Select` base-ui
  « Champ Frigg » : « Ne pas importer », groupe « Fiche » (Prénom *, Nom *,
  Date de naissance *, Adresse électronique, Téléphone, Adresse, Complément
  d'adresse, Code postal, Commune), groupe « Informations complémentaires »
  (champs personnalisés actifs). Une cible déjà prise par une autre colonne
  apparaît désactivée avec « (colonne « Mail ») ».
- Pré-rempli avec `suggested`. Résumé au-dessus du tableau : « 7 colonnes
  sur 9 associées ». Si un champ exigé n'est associé à aucune colonne :
  `Alert` `--warn` le nommant, bouton « Voir l'aperçu » désactivé avec la
  raison reliée.
- Actions : « Changer de fichier » (retour étape 1), « Voir l'aperçu »
  (`preview`). Erreurs `422 form.mapping[i]` affichées sur le `Select` de
  la colonne i.

### 3. Aperçu

- Compteurs (région `aria-live="polite"`) : trois `Badge`-statistiques
  « 212 à importer » (`--ok`), « 3 en erreur » (`--danger`), « 2 doublons »
  (`--warn`). Si `incompleteCount > 0` : `Alert` `--info` « 15 fiches
  importées n'auront pas de valeur pour un champ obligatoire : Numéro de
  licence (15). Vous pourrez les compléter ensuite. »
- Filtre `RadioGroup` : Toutes, À importer, En erreur, Doublons (avec
  nombres). Par défaut : « En erreur » s'il y en a, sinon « Toutes ».
- Tableau paginé côté client (50 lignes) : Ligne, Nom, Prénom, Naissance,
  État (`Badge`), Détail : liste des erreurs « Mail : adresse électronique
  invalide » ; pour un doublon « Déjà dans Frigg : » lien vers
  `/admin/membres/{id}` (nouvel onglet non imposé) ou « Même personne qu'à
  la ligne 12 ».
- Actions : « Modifier les colonnes » (retour étape 2, correspondance
  conservée), « Télécharger les lignes rejetées » (si erreurs ou doublons ;
  `rejected`, fichier `lignes-rejetees.csv`), bouton principal « Importer
  212 membres » ; absent si `counts.ready = 0`, remplacé par une `Alert`
  `--warn` « Aucune ligne à importer : corrigez le fichier ou les
  colonnes. »
- Confirmation : pas de `ConfirmDialog` (action non destructive, aperçu déjà
  affiché) ; bouton occupé pendant l'import.
- `409` à l'import : message `--warn` annoncé, nouvel appel à
  `preview`, l'aperçu est rafraîchi et reste à l'étape 3.

### 4. Rapport

- `Alert` `--ok` « 212 membres importés. » ; liste : lignes en erreur,
  doublons (dont « 1 personne ajoutée entre-temps » si `newDuplicates`),
  fiches à compléter.
- Actions : « Voir les membres sans compte » (`/admin/membres?rattachement=sans-compte`),
  « Télécharger les lignes rejetées » (si besoin, même fichier qu'à
  l'étape 3), « Importer un autre fichier » (retour étape 1, état vidé).
- Message de succès annoncé aux lecteurs d'écran (mécanisme de notification existant).
- Invalidations : liste des membres (toutes les clés `admin-members`) et
  décompte des valeurs obligatoires manquantes de 011.

## Composants

```text
web/app/members/import/
├── ImportSteps.tsx        # ol des étapes, aria-current
├── FileStep.tsx           # aide, modèle, dépôt
├── MappingStep.tsx        # tableau des colonnes et Select
├── PreviewStep.tsx        # compteurs, filtre, tableau, actions
├── ReportStep.tsx
└── download.ts            # téléchargement d'un Blob (factorisé avec l'export des inscrits)
web/app/api/memberImport.ts   # analyze, preview, run, rejected, template
```
