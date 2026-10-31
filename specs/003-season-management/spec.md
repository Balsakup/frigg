# Feature Specification: Gestion des saisons par les administrateurs

**Feature Branch**: `003-season-management`

**Created**: 2026-10-31

**Status**: Draft

**Input**: User description: "Gestion des saisons par les administrateurs. Périmètre : les saisons seules, sans formules ni tarifs. Nouvelle rubrique « Saisons » dans la section du bas de la barre latérale, renommée « Configuration ». Dates libres, libellé déduit des dates, saisons sans chevauchement ni trou, états brouillon → ouverte → close, ouverture manuelle, clôture automatique par les dates, plusieurs saisons ouvertes possibles, saison courante, règles de modification et de suppression, action « Dupliquer », état vide et appel à l'action sur le tableau de bord."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Créer la première saison et consulter les saisons (Priority: P1)

Un administrateur vient d'installer Frigg. Dans l'espace administration, il
ouvre la rubrique « Saisons », placée dans la section « Configuration » de la
barre latérale, à côté de « Comptes ». Aucune saison n'existe : la page lui
explique à quoi sert une saison et l'invite à créer la première. Il saisit une
date de début et une date de fin ; la saison est créée en brouillon et son
libellé (« 2026-2027 ») est calculé à partir des dates. Par la suite, la liste
présente toutes les saisons, de la plus récente à la plus ancienne, avec leurs
dates, leur état, et signale la saison courante. Chaque saison a une fiche.

**Why this priority**: sans saison, aucune adhésion ni aucun exercice comptable
ne pourra être rattaché ; c'est la première brique de la mise en place de
l'outil.

**Independent Test**: sur une base sans saison, ouvrir la rubrique, constater
l'état vide, créer une saison, la retrouver dans la liste avec son libellé, son
état et ses dates, puis ouvrir sa fiche.

**Acceptance Scenarios**:

1. **Given** un administrateur, **When** il affiche la barre latérale de
   l'espace administration, **Then** la section du bas s'intitule
   « Configuration » et contient la rubrique « Saisons » à côté de « Comptes ».
2. **Given** aucune saison, **When** l'administrateur ouvre la rubrique
   « Saisons », **Then** un état vide explique le rôle des saisons et propose
   un bouton « Créer la première saison ».
3. **Given** le formulaire de création, **When** l'administrateur saisit le
   1er septembre 2026 et le 31 août 2027 puis valide, **Then** la saison
   « 2026-2027 » est créée à l'état brouillon et un message confirme l'action.
4. **Given** le formulaire de création, **When** l'administrateur saisit le
   1er janvier 2027 et le 31 décembre 2027, **Then** la saison s'intitule
   « 2027 ».
5. **Given** une date de fin antérieure ou égale à la date de début, ou une
   date de fin déjà passée, **When** l'administrateur valide, **Then** la
   saison n'est pas créée et le champ concerné affiche l'erreur.
6. **Given** plusieurs saisons, **When** la liste s'affiche, **Then** elles
   sont triées de la plus récente à la plus ancienne, avec libellé, date de
   début, date de fin et état ; la saison courante porte une mention
   « Saison courante ».
7. **Given** la liste, **When** l'administrateur ouvre une saison, **Then** la
   fiche présente son libellé, ses dates, son état, si elle est la saison
   courante, et les actions disponibles dans cet état.
8. **Given** un membre connecté, **When** il tente d'obtenir la liste, la fiche
   ou de créer une saison, **Then** l'accès est refusé.

---

### User Story 2 - Préparer et ouvrir la saison suivante (Priority: P1)

En fin de saison, l'administrateur prépare la suivante : depuis la dernière
saison, il choisit « Dupliquer ». Une nouvelle saison est créée en brouillon,
qui commence le lendemain de la fin de la précédente et dure autant qu'elle.
Quand il est prêt (par exemple pour ouvrir les pré-inscriptions avant la fin de
la saison en cours), il ouvre la nouvelle saison : deux saisons sont alors
ouvertes en même temps.

**Why this priority**: c'est le geste annuel de l'administrateur ; sans lui,
l'association ne peut pas passer d'une saison à l'autre ni accepter de
pré-inscriptions.

**Independent Test**: avec une saison 2026-2027 ouverte, dupliquer, vérifier
que 2027-2028 est créée en brouillon avec les bonnes dates, l'ouvrir, et
constater que les deux saisons sont ouvertes et que la saison courante reste
2026-2027.

**Acceptance Scenarios**:

1. **Given** la dernière saison du 1er septembre 2026 au 31 août 2027,
   **When** l'administrateur choisit « Dupliquer », **Then** une saison
   « 2027-2028 » du 1er septembre 2027 au 31 août 2028 est créée à l'état
   brouillon et sa fiche s'affiche.
2. **Given** une saison qui n'est pas la dernière, **When** l'administrateur
   consulte sa fiche, **Then** l'action « Dupliquer » n'est pas proposée.
3. **Given** des saisons existantes, **When** l'administrateur crée une saison
   sans dupliquer, **Then** sa date de début est imposée au lendemain de la fin
   de la dernière saison et il ne choisit que la date de fin.
4. **Given** une saison en brouillon dont la précédente est ouverte ou close,
   **When** l'administrateur choisit « Ouvrir la saison » et confirme, **Then**
   la saison passe à l'état ouverte, même si sa date de début est à venir.
5. **Given** une saison en brouillon dont la précédente est encore en
   brouillon, **When** l'administrateur consulte sa fiche, **Then** l'ouverture
   n'est pas proposée et un message indique qu'il faut d'abord ouvrir la saison
   précédente.
6. **Given** 2026-2027 ouverte et en cours et 2027-2028 ouverte, **When**
   l'administrateur consulte la liste, **Then** les deux saisons sont à l'état
   ouverte et seule 2026-2027 est signalée comme saison courante.

---

### User Story 3 - Corriger ou supprimer une saison (Priority: P2)

L'administrateur s'est trompé de date de fin, ou a créé une saison par erreur.
Tant qu'une saison est en brouillon, il peut modifier ses dates ou la
supprimer. Une fois ouverte, seule sa date de fin reste modifiable (pour
prolonger ou raccourcir la saison). Une saison close ne bouge plus.

**Why this priority**: les erreurs de saisie arrivent, mais sont rares ; la
création et l'ouverture apportent déjà la valeur principale.

**Independent Test**: modifier la date de fin d'une saison en brouillon puis
d'une saison ouverte, constater que la date de début d'une saison ouverte n'est
pas modifiable, supprimer une saison en brouillon, constater qu'une saison
close n'offre aucune action de modification.

**Acceptance Scenarios**:

1. **Given** la première saison, en brouillon, **When** l'administrateur
   modifie sa date de début et sa date de fin, **Then** les nouvelles dates et
   le libellé recalculé sont enregistrés.
2. **Given** une saison ouverte, **When** l'administrateur ouvre la
   modification, **Then** seule la date de fin est modifiable, et elle ne peut
   pas être fixée avant la date du jour.
3. **Given** une saison suivie d'une saison en brouillon, **When**
   l'administrateur modifie sa date de fin, **Then** la date de début de la
   saison suivante est décalée au lendemain de la nouvelle date de fin.
4. **Given** une saison suivie d'une saison ouverte, **When** l'administrateur
   tente de modifier sa date de fin, **Then** la modification est refusée avec
   un message expliquant que la saison suivante est déjà ouverte.
5. **Given** la dernière saison, en brouillon, **When** l'administrateur
   choisit « Supprimer » et confirme dans le dialogue, **Then** la saison
   disparaît de la liste.
6. **Given** une saison qui a été ouverte, ou une saison suivie d'une autre,
   **When** l'administrateur consulte sa fiche, **Then** la suppression n'est
   pas proposée.
7. **Given** une saison close, **When** l'administrateur consulte sa fiche,
   **Then** aucune action de modification ni d'ouverture n'est proposée ; seule
   la suppression l'est, si c'est la dernière et qu'elle n'a jamais été ouverte
   (saison de comblement saisie avec une erreur).

---

### User Story 4 - Clôture automatique et saison courante (Priority: P2)

Personne n'a à penser à clore une saison : le lendemain de sa date de fin,
elle est close d'elle-même. La saison courante est toujours celle dont les
dates contiennent la date du jour ; elle sert de référence aux autres écrans
(adhésions, finances) livrés plus tard.

**Why this priority**: indispensable à la cohérence des données, mais sans
écran propre : elle se constate dans la liste et la fiche.

**Independent Test**: avec une saison dont la date de fin est hier et la
suivante en cours, constater que la première est close et que la seconde est
la saison courante, sans aucune action de l'administrateur.

**Acceptance Scenarios**:

1. **Given** une saison ouverte dont la date de fin était hier, **When**
   l'administrateur consulte la liste, **Then** elle apparaît à l'état close,
   sans qu'aucune action n'ait été nécessaire.
2. **Given** une saison en brouillon dont la date de fin est passée, **When**
   l'administrateur consulte la liste, **Then** elle apparaît à l'état close.
3. **Given** une saison dont la date de fin est aujourd'hui, **When**
   l'administrateur consulte la liste, **Then** elle est encore dans son état
   précédent et reste la saison courante.
4. **Given** aucune saison ne contient la date du jour, **When**
   l'administrateur consulte la liste, **Then** aucune saison n'est signalée
   comme courante.

---

### User Story 5 - Être guidé depuis le tableau de bord (Priority: P3)

En arrivant sur le tableau de bord de l'administration, l'administrateur voit
un appel à l'action quand la mise en place des saisons est incomplète : créer
la première saison, préparer la suivante quand plus aucune saison n'est en
cours, ou ouvrir la saison courante restée en brouillon.

**Why this priority**: un confort de prise en main ; la rubrique « Saisons »
reste accessible sans lui.

**Independent Test**: sur une base sans saison, constater l'appel à l'action et
son lien ; créer et ouvrir une saison en cours, constater qu'il disparaît.

**Acceptance Scenarios**:

1. **Given** aucune saison, **When** l'administrateur ouvre le tableau de bord,
   **Then** un encart l'invite à créer la première saison pour mettre en place
   l'outil, avec un lien vers la création.
2. **Given** des saisons dont aucune ne contient la date du jour, **When**
   l'administrateur ouvre le tableau de bord, **Then** un encart signale
   qu'aucune saison n'est en cours et propose de créer la suivante.
3. **Given** une saison courante en brouillon, **When** l'administrateur ouvre
   le tableau de bord, **Then** un encart signale qu'elle n'est pas ouverte et
   mène à sa fiche.
4. **Given** une saison courante ouverte, **When** l'administrateur ouvre le
   tableau de bord, **Then** aucun encart lié aux saisons n'est affiché.

---

### Edge Cases

- **Installation en cours de saison** : la première saison peut commencer dans
  le passé (une association qui adopte Frigg en novembre crée la saison qui a
  commencé en septembre) ; seule sa date de fin doit être à venir.
- **Saisons de même année civile** : deux saisons tenant chacune dans la même
  année (par exemple deux semestres) auraient le même libellé ; le libellé
  précise alors les mois (« 2027 (janv.–juin) », « 2027 (juil.–déc.) »).
- **Saison de plus d'un an** : une saison du 1er janvier 2026 au 30 juin 2027
  s'intitule « 2026-2027 » ; une saison couvrant trois années civiles
  s'intitule « 2026-2028 ».
- **Interruption** : l'association n'a pas utilisé Frigg depuis la fin de la
  saison 2022-2023 ; l'administrateur recrée (ou duplique) 2023-2024,
  2024-2025 et 2025-2026, closes dès leur création, puis la saison en cours.
- **Durée maximale** : une saison ne peut pas dépasser 24 mois (durée maximale
  d'un exercice comptable).
- **Duplication d'une saison qui n'est pas annuelle** : la nouvelle saison
  dure autant que la saison dupliquée, en mois entiers quand celle-ci couvre
  des mois entiers (1er avril – 30 septembre → 1er octobre – 31 mars), sinon
  en nombre de jours.
- **Raccourcir une saison ouverte jusqu'à aujourd'hui** : autorisé ; elle sera
  close le lendemain. La fixer à hier est refusé : la clôture n'est jamais une
  action manuelle.
- **Décalage d'une saison suivante en brouillon** : si la nouvelle date de fin
  rend la saison suivante vide ou trop longue (plus de 24 mois), la
  modification est refusée.
- **Changement de jour pendant la consultation** : l'état et la saison courante
  sont recalculés à chaque affichage ; une page restée ouverte après minuit se
  met à jour au prochain chargement.
- **Deux administrateurs simultanés** : si deux administrateurs dupliquent ou
  créent une saison en même temps, une seule est créée ; l'autre reçoit un
  message indiquant que la liste a changé.
- **Saison supprimée ou modifiée entre-temps** : une action sur une saison qui
  n'existe plus ou dont l'état a changé est refusée avec un message explicite
  et la fiche est rafraîchie.

## Requirements *(mandatory)*

### Functional Requirements

**Navigation**

- **FR-001**: Dans l'espace administration, la section du bas de la barre
  latérale MUST s'intituler « Configuration » et contenir, dans cet ordre,
  « Saisons » puis « Comptes », puis « Aide ».
- **FR-002**: La rubrique « Saisons » MUST être réservée aux administrateurs,
  comme toutes les opérations de cette fonctionnalité.

**Saison et libellé**

- **FR-003**: Une saison MUST être définie par une date de début et une date
  de fin, toutes deux incluses, choisies librement par l'administrateur.
- **FR-004**: La date de fin MUST être postérieure à la date de début et la
  saison MUST durer au plus 24 mois.
- **FR-005**: Le libellé MUST être calculé à partir des dates : l'année si la
  saison tient dans une année civile (« 2027 »), sinon « année de début-année
  de fin » (« 2026-2027 »). Si deux saisons ont le même libellé, celui-ci MUST
  préciser les mois de début et de fin.
- **FR-006**: Les saisons MUST se suivre sans chevauchement ni trou : chaque
  saison, hormis la première, commence le lendemain de la fin de la précédente.

**Création et duplication**

- **FR-007**: Les administrateurs MUST pouvoir créer une saison. S'il n'en
  existe aucune, ils choisissent les deux dates, la date de début pouvant être
  passée ; sinon la date de début est imposée au lendemain de la fin de la
  dernière saison. La date de fin de la première saison MUST être postérieure
  ou égale à la date du jour ; celle des saisons suivantes peut être passée,
  pour combler le trou laissé par une interruption (les saisons manquantes
  sont recréées une à une, closes dès leur création).
- **FR-008**: Une saison nouvellement créée MUST être à l'état brouillon.
- **FR-009**: Les administrateurs MUST pouvoir dupliquer la dernière saison :
  la nouvelle saison commence le lendemain de sa fin et dure autant qu'elle (en
  mois entiers lorsque la saison dupliquée couvre des mois entiers, sinon en
  nombre de jours). Seule la dernière saison peut être dupliquée, y compris
  close ; la copie peut être entièrement passée (trou à combler).

**États**

- **FR-010**: Une saison MUST avoir l'un des états brouillon, ouverte ou close.
- **FR-011**: Les administrateurs MUST pouvoir ouvrir une saison en brouillon,
  après confirmation, quelle que soit sa date de début, à condition que la
  saison précédente ne soit pas en brouillon.
- **FR-012**: Une saison dont la date de fin est passée MUST être close, quel
  que soit son état antérieur, sans action de quiconque. La clôture n'est
  jamais une action manuelle.
- **FR-013**: Plusieurs saisons MUST pouvoir être ouvertes en même temps.
- **FR-014**: La saison courante MUST être la saison dont les dates contiennent
  la date du jour ; il en existe au plus une. Elle MUST être déterminée quel que
  soit son état.

**Modification et suppression**

- **FR-015**: En brouillon, les administrateurs MUST pouvoir modifier la date
  de fin, et la date de début uniquement s'il s'agit de la première saison.
  La date de fin ne peut pas être antérieure à la date du jour.
- **FR-016**: Une saison ouverte MUST n'admettre que la modification de sa date
  de fin, qui ne peut pas être antérieure à la date du jour.
- **FR-017**: Modifier la date de fin d'une saison suivie d'une saison en
  brouillon MUST décaler la date de début de celle-ci au lendemain ; la
  modification MUST être refusée si la saison suivante est ouverte ou si le
  décalage la rendrait invalide (FR-004).
- **FR-018**: Une saison close MUST NOT être modifiée ni ouverte.
- **FR-019**: Les administrateurs MUST pouvoir supprimer la dernière saison si
  elle n'a jamais été ouverte, close comprise (saison de comblement saisie
  avec une erreur), après confirmation dans un dialogue. Aucune autre saison
  ne peut être supprimée.

**Écrans**

- **FR-020**: La liste MUST présenter toutes les saisons de la plus récente à
  la plus ancienne, avec libellé, date de début, date de fin, état, et la
  mention « Saison courante » sur celle qui l'est.
- **FR-021**: Sans aucune saison, la liste MUST afficher un état vide
  expliquant le rôle des saisons et proposant de créer la première.
- **FR-022**: La fiche d'une saison MUST afficher ses informations et ne
  proposer que les actions autorisées dans son état et sa position ; une action
  indisponible dont l'absence pourrait surprendre (ouverture bloquée par la
  saison précédente) MUST être expliquée.
- **FR-023**: Le tableau de bord de l'administration MUST afficher un appel à
  l'action lorsque : aucune saison n'existe (créer la première) ; aucune
  saison ne contient la date du jour (créer la suivante) ; la saison courante
  est en brouillon (l'ouvrir). Aucun encart n'apparaît lorsque la saison
  courante est ouverte.
- **FR-024**: Chaque erreur de saisie MUST être affichée sur le champ concerné
  avec un message en français expliquant comment la corriger.

### Key Entities

- **Saison** : période de l'activité de l'association, qui sert aussi
  d'exercice comptable. Attributs : date de début, date de fin, état enregistré
  (brouillon ou ouverte), date d'ouverture. Attributs déduits : libellé, état
  affiché (close si la date de fin est passée), indicateur de saison courante.
  Les saisons forment une suite continue ordonnée par date. Les adhésions,
  tarifs et opérations comptables livrés plus tard s'y rattacheront.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Un administrateur qui découvre Frigg crée sa première saison en
  moins d'une minute à partir du tableau de bord, sans aide.
- **SC-002**: Préparer et ouvrir la saison suivante demande au plus trois
  actions depuis la fiche de la saison en cours (dupliquer, ouvrir, confirmer).
- **SC-003**: Aucune combinaison d'actions ne permet d'obtenir deux saisons qui
  se chevauchent, un trou entre deux saisons, ou une saison close modifiée.
- **SC-004**: Le lendemain de sa date de fin, 100 % des saisons apparaissent
  closes sans intervention.
- **SC-005**: Pour toute opération sur les saisons, un accès anonyme ou par un
  membre est refusé dans 100 % des cas.
- **SC-006**: Les parcours de la rubrique et du tableau de bord sont
  utilisables au clavier et ne présentent aucune violation d'accessibilité
  grave ou critique.

## Assumptions

- **Périmètre** : formules, tarifs, adhésions et codes promo sont hors de
  cette fonctionnalité ; « Dupliquer » ne reprend donc que les dates. Quand les
  tarifs existeront, la duplication les reprendra aussi.
- **Saisons passées** : la première saison peut commencer dans le passé mais
  doit se terminer aujourd'hui ou plus tard (pas de reprise d'historique avant
  elle). Après une interruption, même de plus de 24 mois, les saisons
  manquantes sont recréées une à une, entièrement passées, pour respecter la
  continuité ; elles resteront le plus souvent vides. Une saison créée passée
  est close aussitôt : elle ne peut plus être modifiée, mais tant qu'elle est
  la dernière et n'a jamais été ouverte, elle peut être supprimée pour corriger
  une erreur de saisie.
- **Date de début des saisons suivantes** : la continuité sans trou impose la
  date de début de toute saison qui n'est pas la première ; « toutes les dates
  modifiables en brouillon » s'applique donc pleinement à la première saison,
  et se limite à la date de fin pour les suivantes.
- **Fuseau horaire** : « la date du jour » s'entend dans le fuseau de
  l'association, Europe/Paris (association française).
- **Espace membre** : la section du bas de l'espace membre, qui ne contient que
  « Aide », garde son titre « Ressources » ; les membres ne voient pas les
  saisons dans cette fonctionnalité.
- **Historique** : pas de journal des modifications des saisons dans cette
  fonctionnalité ; seule la date d'ouverture est conservée.
- **Volume** : quelques dizaines de saisons au plus sur la vie d'une instance ;
  la liste n'a besoin ni de pagination ni de recherche.
