# Contrat des écrans : paramètres généraux et identité visuelle

Les sections de la rubrique « Paramètres » sont déclarées dans
`settingsSections` (`routes/app/admin/SettingsPage.tsx`) et dans
`web/app/routes.ts` sous `parametres`. Ordre : **Général**, **Apparence**,
Paiements. L'index `/admin/parametres` redirige vers
`/admin/parametres/general` (FR-001).

| Route | Fichier | Contenu |
| --- | --- | --- |
| `/admin/parametres/general` | `routes/app/admin/settings-general.tsx` | Section « Général » (US1, US2, US4) |
| `/admin/parametres/apparence` | `routes/app/admin/settings-appearance.tsx` | Section « Apparence » (US3) |

Fil d'Ariane : « Administration › Paramètres › Général » / « … › Apparence ».

## `/admin/parametres/general`

Formulaire `react-hook-form` unique, un bouton « Enregistrer » en bas, des
groupes `fieldset` avec `legend` :

1. **Identité** : Nom (obligatoire), Tagline (aide : « Affichée sous le nom
   dans le menu et sur l'accueil des membres. », compteur de caractères), Nom
   court, Objet (`textarea`).
2. **Identité légale** : Numéro RNA, Numéro SIRET (`inputmode="numeric"`),
   Date de création (`type="date"`, `max` = aujourd'hui).
3. **Coordonnées** : Adresse, Complément, Code postal
   (`inputmode="numeric"`, `autocomplete="postal-code"`), Ville, Adresse
   électronique de contact (`type="email"`), Téléphone (`type="tel"`), Site
   web (`type="url"`).
4. **Réseaux sociaux** : six champs `type="url"` libellés « Lien Facebook »,
   « Lien Instagram »… (pas d'icône de marque : lucide-react 1.x n'en fournit
   plus).

Comportement :

- Enregistrement réussi : notification « Paramètres enregistrés. », les
  valeurs renvoyées (normalisées : RNA en majuscules, SIRET compact,
  téléphone) remplacent celles du formulaire, `session` et `identity` sont
  invalidées (FR-027).
- `422` : erreurs affichées sous chaque champ (`aria-describedby`,
  `aria-invalid`), focus sur le premier champ en erreur, bandeau d'alerte
  récapitulatif en tête de formulaire.
- Quitter avec des modifications : `ConfirmDialog` « Quitter sans enregistrer ? »
  (FR-030).

## `/admin/parametres/apparence`

Deux blocs et un aperçu ; sur grand écran, l'aperçu est à droite et reste
visible au défilement ; à 320 px il passe sous les champs.

1. **Logo** : logo actuel (ou initiales) ; bouton « Choisir une image » (champ
   fichier natif, `accept="image/png,image/jpeg,image/webp"`) ; texte d'aide
   « PNG, JPEG ou WebP, 1 Mo au plus. » ; bouton « Retirer le logo »
   (`ConfirmDialog` « Retirer le logo ? Les initiales de l'association
   s'afficheront à la place. »). Un fichier choisi est montré dans l'aperçu
   (URL d'objet locale) et déposé à l'enregistrement.
2. **Couleurs** : « Couleur principale » et « Couleur d'accent », chacune un
   `input type="color"` associé à un champ texte hexadécimal synchronisé
   (même libellé, saisie au clavier possible) ; bouton « Rétablir les couleurs
   par défaut » (`ConfirmDialog`).
3. **Aperçu** (`aria-label="Aperçu"`, région non interactive) : en-tête de
   barre latérale (logo, nom, tagline), élément de menu actif, bouton
   principal, bouton d'accent, lien, et un bandeau d'e-mail miniature. Les
   couleurs viennent de `preview-palette` (appel 150 ms après la dernière
   modification) et sont appliquées en variables CSS **sur le conteneur de
   l'aperçu seulement** (FR-019). Un texte en région `status` signale quand
   un texte a été assombri pour rester lisible.

Enregistrement : logo d'abord s'il a changé (erreur sur le champ et arrêt en
cas de refus), puis couleurs ; notification « Apparence enregistrée. » ;
`identity`, `session` et `appearance` invalidées.

## Shell et écrans existants modifiés

| Élément | Fichier | Changement |
| --- | --- | --- |
| Variables de couleur | `web/app/shell/BrandStyle.tsx` (nouveau), `root.tsx` | `root` précharge `identity` ; `BrandStyle` écrit les six variables sur `<html>` |
| Barre latérale | `shell/Sidebar.tsx` | logo (`<img alt="">`, `object-contain`) à défaut initiales ; tagline sous le nom, tronquée avec `title` ; replié : nom et tagline en `sr-only` |
| Accueil membre | `routes/app/member/home.tsx` | tagline sous le titre ; encart « Contacter l'association » (`get-association-contact`), absent si vide |
| Écrans hors session | `routes/auth/layout.tsx` | logo ou initiales et nom de l'association, couleurs appliquées ; à défaut identité Frigg |
| Titre d'onglet, pied de page | `shell/Page.tsx`, `shell/Footer.tsx` | inchangés (lisent déjà le nom de la session) |

## Accessibilité (RGAA)

- Chaque champ a son `label` ; les groupes ont un `legend`.
- Le logo est décoratif partout où le nom est écrit à côté (`alt=""`).
- Liens sociaux de l'encart : texte visible du nom du réseau, icône générique
  `Share2` `aria-hidden`, nom accessible « Facebook de {nom} », ouverture dans un nouvel onglet signalée (« nouvelle
  fenêtre » en texte masqué).
- Contrastes garantis par la palette ([research.md › R3](../research.md#r3-palette-dérivée-et-contraste)) ;
  contrôlés par axe sur les deux sections, l'accueil membre et l'écran de
  connexion habillé avec une couleur claire.
