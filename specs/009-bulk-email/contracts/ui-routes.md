# Contrat des écrans : communications

La rubrique « Communication » de la section administration existe déjà dans
`web/app/shell/navigation.ts` (`/admin/communication`) et pointe aujourd'hui
vers `coming-soon`. Elle est retirée de `adminComingSoon` dans
`web/app/routes.ts` et remplacée par :

| Route | Fichier | Contenu |
| --- | --- | --- |
| `/admin/communication` | `routes/app/admin/communications-layout.tsx` (layout) + `routes/app/admin/communications.tsx` (index) | Historique |
| `/admin/communication/nouvelle` | `routes/app/admin/communication-new.tsx` | Préparation et envoi |
| `/admin/communication/:id` | `routes/app/admin/communication-detail.tsx` | Détail d'une communication |

Fil d'Ariane : « Administration › Communication › Nouvelle communication » /
« … › {objet} ».

## `/admin/communication` (historique, US4)

- En-tête de page : titre « Communication », introduction « Écrivez aux
  adhérents d'une saison et retrouvez les messages envoyés. », action
  principale « Nouvelle communication ».
- Tableau (`ui/Table`) : Date, Objet (lien vers le détail), Cible, Auteur,
  Destinataires, Envoi (« 18 envoyés · 1 en échec » ; badge `danger` si au
  moins un échec, `info` si des envois sont en attente). Pagination
  (`ui/Pagination`), page reflétée dans l'adresse (`?page=`).
- État vide : icône `Mail`, « Aucune communication pour l'instant », texte
  expliquant la rubrique, bouton « Nouvelle communication ».

## `/admin/communication/nouvelle` (US1, US2, US3)

Formulaire `react-hook-form` en trois sections, une seule page.

1. **Destinataires**
   - `Select` « Saison » : saisons ouvertes et closes, courante par défaut.
   - `RadioGroup` « Cible » : « Tous les adhérents », « À jour »,
     « En impayé » (sélection par défaut), « Personnalisée ».
   - « Personnalisée » révèle deux groupes de cases (`ui/Checkbox`,
     `CheckboxGroup` de base-ui) : « Situation de paiement » (À jour,
     Paiement annoncé, À payer) et « Tarifs » (tous les tarifs de la saison,
     aucun coché = tous, texte d'aide le précise).
   - Encart de décompte dans une région `role="status"` `aria-live="polite"` :
     « 23 adhérents ciblés · 19 destinataires » et, s'il y en a, alerte `warn`
     « 2 adhérents n'ont aucune adresse et ne recevront pas ce message. ».
     Boutons « Voir les adhérents ciblés » et « Voir les non joignables »
     ouvrant un `Dialog` avec tableau (Nom, Prénom, Situation, Tarif, Adresse
     de contact). Pendant le recalcul : indicateur de chargement, ancien
     décompte conservé.
2. **Message**
   - Champ « Objet » (`maxLength` 150, compteur), zone de texte « Message »
     (`maxLength` 10 000, compteur, retours à la ligne préservés). Aide :
     « Texte simple. Les liens commençant par https:// deviennent cliquables. »
3. **Pièces jointes**
   - Bouton « Ajouter des fichiers » associé à un `<input type="file"
     multiple accept=".pdf,.jpg,.jpeg,.png,.docx,.xlsx,.odt,.ods">` (l'attribut
     `accept` n'est qu'un confort ; le serveur fait autorité). Aide : formats
     acceptés, 5 fichiers, 5 Mo par fichier, 10 Mo au total.
   - Liste : icône selon le type, nom, taille (« 1,2 Mo »), état (« Envoi… »,
     erreur en `danger` sous le fichier), bouton « Retirer » (`aria-label`
     « Retirer convocation.pdf »). Total affiché « 2 fichiers · 3,4 Mo sur
     10 Mo ».
   - Un fichier refusé reste affiché avec son message jusqu'à ce qu'on le
     ferme ; les autres fichiers ne sont pas affectés.

Actions en pied de formulaire :

- « Aperçu » (secondaire) : `Dialog` avec l'objet, la liste des pièces
  jointes et le message rendu (`iframe` `sandbox=""` `srcdoc`, titre
  « Aperçu du message »).
- « Envoyer » (principal) : désactivé tant qu'un dépôt est en cours ;
  `ConfirmDialog` « Envoyer cette communication ? » rappelant objet, cible
  (`targetLabel`), « 19 destinataires » et les pièces jointes ; bouton
  « Envoyer à 19 destinataires ».
  - `409` : le dialogue reste ouvert, alerte `warn` « Le nombre de
    destinataires a changé : 18 au lieu de 19. », décompte mis à jour, la
    confirmation porte le nouveau nombre.
  - `201`/`200` : redirection vers `/admin/communication/:id`, notification
    « Communication envoyée à 19 destinataires. » annoncée.
- Quitter la page avec un formulaire modifié : confirmation du navigateur
  (`useBlocker` de React Router) « Abandonner cette communication ? ».

La clé de préparation (`crypto.randomUUID()`) est créée au montage de la page
et conservée pendant la préparation ; elle n'est pas mise dans l'adresse.

## `/admin/communication/:id` (US4, US2)

- En-tête : objet en titre, introduction « Envoyée le 3 novembre 2026 à 14 h 05
  par Marc Dupont ».
- Bloc « Cible » : saison, cible (`targetLabel`), adhérents ciblés, non
  joignables, destinataires.
- Bloc « Message » : texte affiché avec retours à la ligne préservés
  (`whitespace-pre-wrap`).
- Bloc « Pièces jointes » (si présentes) : liens de téléchargement
  `<a href="/api/admin/communications/{id}/attachments/{aid}" download>` avec
  nom et taille.
- Bloc « Destinataires » : résumé « 18 envoyés · 1 en attente · 1 en échec »,
  tableau Adresse (« Adresse supprimée » en `muted` si null), Adhérents,
  État (badges `ok` « Envoyé », `info` « En attente », `danger` « En
  échec »), Envoyé le. Échecs en tête. Rafraîchissement automatique toutes
  les 15 s tant qu'au moins un envoi est en attente
  (`refetchInterval`).

## Composants nouveaux

| Composant | Fichier | Base |
| --- | --- | --- |
| `Checkbox`, `CheckboxGroup` | `web/app/ui/Checkbox.tsx` | `@base-ui/react` `Checkbox`, `CheckboxGroup` |
| `AudienceSummary` | `web/app/communications/AudienceSummary.tsx` | région `status`, `Dialog` |
| `AttachmentsField` | `web/app/communications/AttachmentsField.tsx` | input natif, `openapi-fetch` |
| `MessagePreview` | `web/app/communications/MessagePreview.tsx` | `Dialog`, `iframe` sandbox |
| `DeliveryBadge` | `web/app/communications/DeliveryBadge.tsx` | `ui/Badge` |

Données : `web/app/api/communications.ts` (requêtes et mutations
`openapi-react-query`, invalidation de `list-communications` après envoi).
