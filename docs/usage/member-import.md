# Importation des membres

**Rôle requis** : Administrateur

Pour démarrer avec Frigg sans tout ressaisir, importez vos fiches depuis un
tableur. L'importation se lance depuis la rubrique *Membres* ; elle crée des
fiches **sans compte**, sans adhésion, et n'envoie aucun e-mail.

## 1. Préparer le fichier

- Format **CSV** uniquement (« Enregistrer sous… CSV » dans votre tableur).
  Téléchargez le **modèle** proposé par Frigg : ses colonnes sont reconnues
  d'office.
- 5 Mo et 2 000 lignes au plus.
- La première ligne contient les intitulés des colonnes.
- L'encodage (UTF-8 ou Windows-1252) et le séparateur (virgule,
  point-virgule ou tabulation) sont détectés automatiquement.
- **Dates** : `31/12/2010`, `31-12-2010`, `31.12.2010` ou `2010-12-31`. Une
  année sur deux chiffres est refusée.
- **Oui/non** : `oui`, `non`, `vrai`, `faux`, `1`, `0` ou `x`.
- **Listes de choix** : la valeur doit correspondre à un choix existant (sans
  tenir compte des majuscules ni des accents).

## 2. Associer les colonnes

Frigg propose pour chaque colonne un champ de la fiche (prénom, nom, date de
naissance, adresse électronique, téléphone, adresse postale) ou un
[champ personnalisé](./custom-fields) de membre, d'après son intitulé. Une
colonne non reconnue est proposée en « Ne pas importer ».

Vous pouvez modifier chaque association en voyant des valeurs d'exemple. Le
prénom, le nom et la date de naissance doivent être associés pour continuer.

## 3. Vérifier l'aperçu

Avant toute écriture, l'aperçu compte les lignes **à importer**, **en
erreur** et **en doublon**, et les liste avec leur numéro dans le fichier.

- Une ligne **en erreur** ne respecte pas les règles de la fiche ; chaque
  erreur est expliquée avec la colonne concernée. Rien n'est corrigé ni
  tronqué par supposition.
- Une ligne **en doublon** a le même prénom, le même nom et la même date de
  naissance qu'une fiche existante (lien vers celle-ci) ou qu'une ligne
  précédente du fichier.
- Une valeur manquante pour un champ personnalisé obligatoire n'empêche pas
  l'import ; l'aperçu indique combien de fiches seront à compléter.

Téléchargez si besoin le fichier des **lignes rejetées**, avec une colonne
« Raison du rejet », pour les corriger et les réimporter.

## 4. Importer

À la confirmation, Frigg crée une fiche par ligne à importer et ignore les
lignes en erreur et en doublon. L'importation est tout ou rien : en cas
d'incident, aucune fiche n'est créée. Un rapport récapitule le résultat.

Le fichier déposé n'est jamais conservé par Frigg.

## Ensuite

Pour donner accès à l'application, [invitez](./admin-accounts) les personnes
puis rattachez leur compte à leur fiche depuis la rubrique *Membres*.
