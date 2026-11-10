# Quickstart : valider l'importation des membres

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Modèle : [data-model.md](./data-model.md).
Prérequis : ceux de [005 › quickstart.md](../005-member-management/quickstart.md)
(`create-admin`, `go tool task dev`) ; une fiche existante « Marie Dupont »
née le 12/03/2012 ; deux champs personnalisés de membre actifs (011) :
« Numéro de licence » (texte court, obligatoire) et « Catégorie » (liste :
Poussin, Benjamin, Minime).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 012 :

- aucune migration ajoutée ; `sqlc generate` (requête
  `ListMemberIdentities`), `frigg openapi` et `bun run gen:api` sans diff ;
- matrice d'autorisation : 5 opérations `admin` ajoutées, aucune publique ;
- tests en table de `internal/csvfile` et `internal/memberimport` ;
- tests d'API de l'import : concurrence (deux imports simultanés du même
  fichier → les fiches existent une seule fois), atomicité, empreinte
  changée → `409`, réimport des lignes rejetées corrigées, journaux sans
  donnée personnelle ;
- `CGO_ENABLED=0 go build` : aucune dépendance nouvelle ;
- Playwright + axe : `member-import.spec.ts` sans violation `serious` ni
  `critical`.

## Fichier d'exemple

`exemple.csv`, enregistré en Windows-1252 (« CSV » d'Excel), séparateur `;` :

```text
NOM;Prénom;Date de naissance;Mail;Tél.;Code postal;Ville;Adresse;Licence;Catégorie;Remarques
Lefèvre;Léa;03/04/2014;lea@example.org;06 12 34 56 78;69001;Lyon;3 rue Neuve;L-001;poussin;
Dupont;Marie;12/03/2012;;;;;;L-002;Benjamin;déjà là
Martin;Paul;3/4/12;paul.example.org;;;;;;Minime;
Martin;Paul;15-09-2013;;;;;;;minime;
Martin;Paul;15.09.2013;;;;;;;Minime;répété
;Sans nom;01/01/2015;;;;;;;Cadet;
Durand;Hugo;2011-06-30;;+32 470 12 34 56;1000;Lyon;1 place;;;
```

## Scénarios manuels

### 1. Importer (US1)

1. Admin › Membres : bouton « Importer » dans l'en-tête → page « Importer
   des membres », étape 1 avec aide CSV et lien vers le modèle.
2. Déposer `exemple.csv`. Attendu : étape 2, « 7 lignes », encodage
   Windows-1252 et séparateur « ; » détectés, accents corrects dans les
   exemples. Colonnes associées d'office : NOM → Nom, Prénom → Prénom,
   Date de naissance, Mail → Adresse électronique, Tél. → Téléphone, Code
   postal, Ville → Commune, Adresse, Catégorie ; « Licence » et
   « Remarques » en « Ne pas importer ».
3. Associer « Licence » à « Numéro de licence ». Ouvrir le `Select` de
   « Remarques » : « Numéro de licence » est désactivé « (colonne
   « Licence ») ».
4. Associer temporairement « Date de naissance » à « Ne pas importer » :
   alerte nommant le champ, « Voir l'aperçu » désactivé. Rétablir.
5. « Voir l'aperçu ». Attendu : 2 à importer (Léa Lefèvre, Paul Martin ligne
   5), 3 en erreur, 2 doublons ; filtre sur « En erreur ». Aucune fiche
   créée (liste « Membres » inchangée dans un autre onglet).
6. « Importer 2 membres ». Attendu : rapport « 2 membres importés », 3 en
   erreur, 2 doublons, « 1 fiche à compléter » (Paul sans licence). « Voir
   les membres sans compte » : Léa et Paul y figurent ; la fiche de Léa porte
   « Catégorie : Poussin », « Numéro de licence : L-001 », téléphone
   `0612345678`.

### 2. Lignes rejetées (US2)

Sur l'aperçu de l'étape 1.5 :

| Ligne | État | Raison attendue |
| --- | --- | --- |
| 3 (Dupont Marie) | doublon | « Déjà dans Frigg » avec lien vers la fiche |
| 4 (Martin Paul, 3/4/12) | erreur | date : année sur 4 chiffres ; Mail : adresse électronique invalide |
| 6 (Martin Paul, 15.09.2013) | doublon | « Même personne qu'à la ligne 5 » |
| 7 (sans nom) | erreur | Nom obligatoire ; Catégorie : choix possibles Poussin, Benjamin, Minime |
| 8 (Durand Hugo) | erreur | Code postal : 5 chiffres attendus (zéro initial perdu) |

1. « Télécharger les lignes rejetées » : `lignes-rejetees.csv` s'ouvre dans
   un tableur avec les accents ; colonnes d'origine + « Raison du rejet » ;
   le téléphone de Hugo apparaît `'+32 470…` (neutralisé).
2. Corriger la ligne de Hugo (`01000`), celle de Paul (date, adresse électronique), puis
   réimporter ce fichier. Attendu : « Raison du rejet » en « Ne pas
   importer » ; Hugo importé avec le téléphone `+32470123456` ; Paul
   (ligne 4 corrigée, 03/04/2012) importé ; Marie et Paul né en 2013
   restent des doublons ; aucune nouvelle erreur sur les lignes corrigées
   (SC-005).

### 3. Modèle (US4)

Télécharger le modèle : intitulés « Prénom * », « Nom * », « Date de
naissance * (JJ/MM/AAAA) », …, « Numéro de licence », « Catégorie (Poussin,
Benjamin, Minime) ». Le remplir de deux lignes et le déposer : toutes les
colonnes sont associées d'office.

### 4. Cas limites

- Déposer un classeur `.xlsx` renommé en `.csv` : refus « n'est pas un
  fichier CSV lisible » avec la marche à suivre.
- Fichier avec un guillemet non refermé : refus nommant la ligne.
- Fichier de 2 001 lignes : refus invitant à découper.
- Fichier UTF-8 séparé par des virgules avec une adresse entre guillemets
  contenant une virgule : lu correctement.
- Pendant l'aperçu, archiver « Catégorie » dans un autre onglet puis
  importer : retour à l'étape « Colonnes », « Ce champ n'existe plus ou
  n'est pas importable. » sous la colonne « Catégorie » (`form.mapping[9]`).
- Pendant l'aperçu, renommer un champ personnalisé non associé puis
  importer : message « Les champs personnalisés ont changé… » et aperçu
  recalculé (`409`).
- Pendant l'aperçu, créer à la main la fiche « Léa Lefèvre » puis importer :
  rapport avec « 1 personne ajoutée entre-temps », Léa non dupliquée.
- Connecté en membre : `/admin/membres/importer` refusé ; `POST
  /api/admin/members/import/analyze` → `403`.
- Journaux : une ligne `member.import` avec les compteurs et les
  identifiants extrêmes, sans nom, adresse ni nom de fichier.
