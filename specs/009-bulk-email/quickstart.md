# Quickstart : valider l'envoi d'e-mails groupés

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Modèle : [data-model.md](./data-model.md).
Prérequis : ceux de [006 › quickstart.md](../006-memberships/quickstart.md)
(Mailpit, `create-admin`, `go tool task dev`, saison ouverte avec tarifs).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 008 :

- `00009_communications.sql` s'applique sur une base vierge et sur une base
  `00008` peuplée (e-mails existants conservés, `communication_id` et
  `from_name` à NULL) ;
- matrice d'autorisation : 8 nouvelles opérations, toutes `admin` (anonyme →
  `401`, membre → `403`) ; aucune entrée ajoutée à `allowedPublic` ;
- `internal/communication` (tests en table) : préréglages ↔ situations,
  `BuildAudience` (adresse du compte prioritaire, adresse de fiche à défaut,
  non joignable sinon, dédoublonnage sans casse), `DetectType` (PDF, JPEG,
  PNG, DOCX, XLSX, ODT, ODS acceptés ; DOCM, ZIP quelconque, exécutable
  renommé `.pdf`, `.doc` refusés), `SanitizeFilename` ;
- `internal/mail` : rendu `communication` (balisage échappé, liens `https`
  cliquables, paragraphes, mention de saison), worker joignant les pièces
  (noms d'origine, contenu identique), fichier manquant → `failed` sans
  nouvelle tentative, `from_name` appliqué ;
- tests d'API :
  - audience : chaque situation (payée, gratuite, annoncée, à payer, carte
    échouée → à payer), demandes en attente/refusées/annulées exclues,
    restriction par tarif, saison brouillon → `422`, tarif d'une autre saison
    → `422` ;
  - dépôt : 6 types acceptés, type refusé → `422`, 5 Mo + 1 octet → `413`
    ou `422`, 6ᵉ fichier → `422`, total > 10 Mo → `422`, pièce d'un autre
    administrateur → `404` à la suppression ;
  - envoi : `201` et un `outbound_emails` par adresse distincte ; même
    `draftKey` renvoyé → `200`, aucune ligne ajoutée ; décompte changé entre
    audience et envoi → `409` avec la nouvelle audience ; cible vide → `422` ;
    pièces rattachées, plus supprimables ;
  - déclencheurs : `MarkEmailSent` / `MarkEmailFailed` mettent à jour
    l'état du destinataire ; purge de la file → état conservé ;
  - balayage (horloge injectée) : pièce de préparation de 25 h supprimée
    (ligne et fichier), pièce rattachée conservée, fichier orphelin supprimé ;
  - téléchargement : en-têtes `Content-Disposition` (nom accentué),
    `nosniff`, pièce non rattachée → `404` ;
  - suppression d'un compte destinataire : adresse effacée dans l'historique,
    e-mail en attente supprimé → destinataire « en échec » ;
  - journaux : aucune adresse, aucun objet, aucun nom de fichier.
- Playwright + axe : `web/e2e/communications.spec.ts` sans violation
  `serious` ni `critical`.

## Validation manuelle

Préparer une saison 2026-2027 ouverte, deux tarifs (« Jeune », « Adulte »), et
des adhésions validées :

| Membre | Compte | Tarif | Paiement |
| --- | --- | --- | --- |
| Léo et Inès Martin | Claire (`claire@example.test`) | Jeune | à payer (les deux) |
| Paul Durand | Paul (`paul@example.test`) | Adulte | paiement annoncé (chèque) |
| Anne Petit | sans compte, fiche `anne@example.test` | Adulte | payée |
| Hugo Roux | sans compte, sans adresse | Jeune | à payer |
| Zoé Blanc | Zoé | Jeune | gratuite (0 €) |
| Marc Noir | Marc | Adulte | demande en attente |

1. **Relance des impayés (US1)** — `/admin/communication` → « Nouvelle
   communication ». Saison 2026-2027, cible « En impayé ». Attendu :
   « 4 adhérents ciblés · 2 destinataires » et « 1 adhérent n'a aucune
   adresse » (Hugo). « Voir les adhérents ciblés » liste Léo, Inès et Paul ; « Voir les non joignables » liste Hugo.
2. Saisir l'objet « Rappel : cotisation » et un texte contenant
   `<b>test</b>` et `https://example.test/infos`. « Aperçu » : le balisage
   apparaît tel quel, le lien est cliquable, la mention de saison figure en
   pied.
3. **Pièces jointes (US2)** — ajouter `convocation.pdf` (≈ 1 Mo) et
   `planning.ods` : tous deux listés. Ajouter un `.exe` renommé `.pdf` :
   refusé avec la liste des formats. Ajouter un fichier de 6 Mo : refusé
   (« dépasse 5 Mo »). Retirer `planning.ods`.
4. « Envoyer » : le dialogue rappelle objet, cible, « 2 destinataires » et
   `convocation.pdf`. Confirmer. Attendu : redirection vers le détail,
   notification annoncée.
5. Dans Mailpit (<http://localhost:8025>) : un seul e-mail pour
   `claire@example.test` (malgré deux enfants), un pour `paul@example.test`,
   expéditeur affiché avec le nom de l'association, chacun avec
   `convocation.pdf` identique (même taille, s'ouvre), aucune autre adresse
   visible. Aucun e-mail pour Anne, Zoé ni Marc.
6. **Décompte changé** — préparer une nouvelle communication « En impayé »,
   puis, dans un autre onglet, constater le paiement de Paul. Confirmer
   l'envoi : le dialogue indique « 1 au lieu de 2 » et redemande la
   confirmation.
7. **Cibles (US3)** — « Tous les adhérents » : 6 adhérents ciblés (Léo,
   Inès, Paul, Anne, Hugo, Zoé), 4 destinataires, 1 non joignable. « À jour » :
   Anne et Zoé. « Personnalisée » avec seulement « À payer » et le tarif
   « Adulte » : 0 → « Envoyer » impossible, message explicite.
8. **Historique (US4)** — `/admin/communication` : communications listées,
   plus récente en tête, avec cible et nombre de destinataires. Détail : texte,
   pièce téléchargeable sous son nom d'origine, états « Envoyé ». Arrêter
   Mailpit, envoyer une communication : états « En attente » puis, après
   relance de Mailpit, « Envoyé » sans recharger la page.
9. **Abandon** — ajouter une pièce puis quitter la page : confirmation
   demandée. Après 24 h (ou balayage avec horloge avancée en test), le fichier
   a disparu de `FRIGG_DATA_DIR/communications/`.
10. **Accès** — connecté en membre, `/admin/communication` affiche l'accès
    refusé ; `GET /api/admin/communications` → `403`.
11. **Clavier et lecteur d'écran** — tout le parcours au clavier ; le
    décompte est annoncé à chaque changement de cible ; à 320 px, aucun
    défilement horizontal hors tableaux (`ScrollRegion`).
