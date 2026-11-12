# Événements et inscriptions

**Rôle requis** : Membre pour s'inscrire ; Administrateur pour publier les
événements et gérer les inscriptions

Les événements sont les activités de l'association : assemblée générale,
sortie, stage… Les membres y inscrivent les fiches qu'ils gèrent, et payent
si l'événement est payant.

## Côté membre

- *Événements à venir* liste les événements publiés (ou annulés) qui ne sont
  pas terminés. La page d'un événement indique les dates, le lieu, le prix ou
  « Gratuit », les places restantes et, pour chacune de vos fiches, si elle
  est inscrite, inscriptible, ou pourquoi elle ne l'est pas.
- **S'inscrire** : pendant la période d'inscription, inscrivez toute fiche
  éligible que vous gérez. Pour un événement payant, la place est réservée
  dès l'inscription et l'inscription reçoit une référence de paiement (par
  exemple `EVT-2627-0001`).
- *Mes inscriptions* permet de suivre vos inscriptions, de les **payer** (par
  les mêmes moyens que les [adhésions](./memberships)) et de les **annuler**
  tant que les inscriptions sont ouvertes et que l'inscription n'est pas
  réglée.
- *À régler* rassemble les adhésions et les inscriptions qui restent à payer.

## Côté administration

### Créer et publier un événement

Dans *Événements* :

| Champ | Règle |
| --- | --- |
| Titre | 120 caractères au plus |
| Début et fin | Dates et heures de Paris ; la fin suit le début |
| Lieu | 200 caractères au plus |
| Description | Facultative, 5 000 caractères au plus |
| Ouverture | *Adhérents de la saison* (adhésion validée, par défaut) ou *Toutes les fiches* |
| Capacité | Facultative, de 1 à 10 000 places, sans liste d'attente |
| Prix | Facultatif, de 0 à 10 000 € |
| Période d'inscription | Facultative ; par défaut, de la publication au début de l'événement |

- Un événement est rattaché à la [saison](./seasons) qui contient sa date de
  début : il faut qu'une saison existe pour ces dates.
- Il est créé en **brouillon**, invisible des membres, puis **publié**. Une
  fois passé, il est présenté comme **terminé** et ses dates ne changent plus.
- Changer le prix ou l'ouverture ne concerne que les inscriptions
  suivantes : le prix est figé à l'inscription. Les inscrits devenus non
  éligibles sont signalés, mais pas retirés.
- La capacité ne peut pas descendre sous le nombre d'inscrits.

### Annuler, rétablir, supprimer

- **Annuler** un événement (avec un message facultatif aux inscrits), puis le
  **rétablir** tant qu'il n'est pas terminé.
- Un événement qui a des inscriptions, même annulées, ne peut pas être
  supprimé : annulez-le.
- **Frigg ne rembourse pas.** À l'annulation d'un événement ou au retrait
  d'une inscription payée, le remboursement se fait hors de Frigg, ou dans
  l'espace HelloAsso pour un paiement par carte.

### Gérer les inscriptions

- Un administrateur peut inscrire tout membre éligible, même hors période
  d'inscription, et retirer toute inscription avant la fin de l'événement.
  Il peut ajuster le montant à payer (voir [Remises et dons](./adjustments)).
- *Inscriptions* liste les inscriptions de tous les événements : recherche
  par nom ou référence, filtres par événement, état et état du paiement, et
  signalements de paiement à traiter.
- Les paiements se constatent comme pour les adhésions ; une inscription
  peut être marquée payée dès sa création.
- La page d'un événement affiche les totaux (dû, encaissé, remises, dons) et
  **exporte ses inscrits** au format CSV.

## E-mails

Le membre reçoit un e-mail à son inscription (avec le lien de paiement si
elle est payante), à la constatation de son paiement et s'il est retiré. Les
inscrits sont prévenus d'un changement de date, d'heure ou de lieu, d'une
annulation et d'un rétablissement, à raison d'un seul e-mail par adresse.
