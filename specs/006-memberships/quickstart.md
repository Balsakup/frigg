# Quickstart : valider les adhésions

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Prérequis et mise en place : ceux de
[001 › quickstart.md](../001-auth-app-shell/quickstart.md) (Mailpit,
`create-admin`, `go tool task dev`).

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 005 :

- la migration `00008_memberships.sql` s'applique sur une base vierge et sur
  une base de développement existante ; la ligne `payment_settings` existe ;
- la matrice d'autorisation couvre les 19 nouvelles opérations (anonyme →
  401 ; membre → 403 sur les 13 opérations `admin` ; administrateur accepté
  partout) ;
- `memberships_isolation_test.go` : pour `get-my-membership`,
  `cancel-my-membership`, `announce-my-payment`, un compte B reçoit `404` sur
  une adhésion d'une fiche de A, d'une fiche sans compte et sur un
  identifiant inexistant, sans modification ; `create-my-membership` sur une
  fiche de A → `404` ; `list-my-memberships` ne renvoie jamais d'adhésion
  d'une autre fiche ;
- tests en table de `internal/membership` : `AppliedAmount` (avec et sans
  montant de renouvellement), toutes les transitions et `ActionsFor` par
  état, état de paiement et état de saison, `PreviousSeasonHint` (oui, non,
  inconnu sans saison précédente, inconnu avec saison précédente sans
  adhésion validée), références (`ADH-2627-0001`, saison civile, passage à
  10 000), IBAN (FR valide avec et sans espaces, clé fausse, pays inconnu,
  longueur), BIC, statistiques ;
- tests d'API : demande, doublon simultané → un `201` et un `409` ; saison
  en brouillon, close ou sans tarif → `422` ; validation avec changement de
  tarif et de type (montant recalculé), refus sans motif → `422`, transition
  depuis un état périmé → `409` ; validation et refus sur saison close
  admis, correction et annulation refusées ; adhésion gratuite sans
  paiement ; annonce, changement, moyen désactivé → `422` ; constatation
  (date future, antérieure à la validation) et annulation (retour à
  `announced` ou `due`) ; correction d'une adhésion payée → `409` ; tarif
  utilisé non supprimable ; fiche avec adhésion en cours non supprimable par
  le compte ; suppression par l'administrateur avec adhésions ; e-mails mis
  en file avec le bon destinataire (aucun pour une fiche sans compte ou un
  compte désactivé) ; chiffres d'une saison ; tableau de bord ;
- Vitest : libellés et formatage (`formatIban`, états) ; Playwright + axe :
  `memberships.spec.ts` (US1, US3, US4), `admin-memberships.spec.ts` (US2,
  US6, US7, US8), `admin-settings.spec.ts` (US5).

## Parcours manuels

Avec un administrateur `admin@exemple.fr`, un membre `claire@exemple.fr`
gérant « Léo Martin » et « Inès Martin », et une saison 2026-2027 ouverte
portant « Adulte » (60 €, renouvellement 40 €), « Jeune » (30 €) et
« Bienfaiteur gratuit » (0 €) :

1. **Paramètres (US5)** — en administrateur, tableau de bord : encart
   « Aucun moyen de paiement… ». « Paramètres » : activer le virement sans
   IBAN → erreur ; saisir `FR76 3000 6000 0112 3456 7890 189` → enregistré ;
   activer le chèque (ordre) ; l'encart disparaît.
2. **Demande (US1)** — en Claire, « Adhésions » : demander pour Léo, tarif
   « Adulte », « Oui, renouvellement » : montant 40,00 € affiché ; envoyer.
   Léo n'est plus proposé pour cette saison. Demander pour Inès, « Jeune ».
3. **Annulation (US3)** — annuler la demande d'Inès, la redéposer.
4. **Décision (US2)** — en administrateur : encart « 2 demandes… ».
   Ouvrir celle de Léo : aide « Inconnu » (pas de saison précédente) ;
   valider en « Renouvellement » : 40 €, référence `ADH-2627-0001`. Refuser
   celle d'Inès avec un motif. Mailpit : e-mail de validation avec le lien,
   e-mail de refus avec le motif.
5. **Paiement (US4)** — en Claire, suivre le lien de l'e-mail (déconnectée :
   connexion puis retour sur la page) ; choisir « Virement » : titulaire,
   IBAN groupé, référence, boutons « Copier » ; confirmer : « Paiement
   annoncé — virement ». En administrateur, « Marquer comme payée » : e-mail
   « payée » ; chiffres : encaissé 40 €.
6. **Création directe (US6)** — créer une adhésion « Jeune » pour une fiche
   sans compte avec « Règlement déjà reçu » (chèque) : directement payée,
   aucun e-mail.
7. **Correction et annulation (US7)** — tenter de corriger l'adhésion payée :
   message ; annuler la constatation, corriger le tarif : montant recalculé ;
   annuler l'adhésion avec motif : Léo redevient proposable pour une
   demande.
8. **Règles ajoutées** — supprimer le tarif « Adulte » : refusé ; en Claire,
   supprimer la fiche de Léo pendant une demande en cours : refusé.
9. **Recherche et chiffres (US8)** — rechercher `adh-2627` puis `mart` ;
   filtrer « Paiement : à payer » ; vérifier les chiffres de la saison.
10. **Journaux** — les lignes `membership.*` et `settings.payment_update`
    ne contiennent ni nom, ni motif, ni IBAN.
11. **Accessibilité** — parcours complet au clavier, à 320 px de large.
