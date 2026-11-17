# Parcours publics

**Rôle requis** : aucun pour les personnes extérieures ; Administrateur pour
ouvrir les parcours, rattacher les demandes et supprimer les fiches inactives

Les **parcours publics** permettent à une personne **sans compte** de demander
son adhésion ou de s'inscrire à un événement, puis de suivre et de payer sa
demande grâce à un **lien personnel** reçu par e-mail. Ils sont désactivés
par défaut : l'association choisit ce qu'elle ouvre.

## Demandes d'adhésion publiques

### Activer et diffuser

1. Dans *Configuration › Paramètres › Général*, section **Adhésions
   publiques**, activez « Accepter les demandes d'adhésion publiques », puis
   enregistrez.
2. L'**adresse de la page d'adhésion** s'affiche (elle se termine par
   `/adherer`). Copiez-la avec le bouton « Copier » et diffusez-la : site de
   l'association, réseaux sociaux, affiche, courriel.

::: warning Tarifs visibles de tous
Une fois l'accueil activé, les saisons ouvertes et leurs tarifs sont visibles
de toute personne qui connaît l'adresse. Désactivez l'interrupteur pour
fermer la page : elle affiche alors « Les demandes d'adhésion ne sont pas
ouvertes pour le moment. »
:::

### Ce que fait la personne

1. Sur `/adherer`, elle saisit son identité, son adresse électronique, choisit
   le tarif et le type (première adhésion ou renouvellement), remplit les
   [champs personnalisés](./custom-fields) et accepte le traitement de ses
   données. Elle peut cocher « Créer un compte pour suivre mes demandes ».
2. Elle reçoit un e-mail de **confirmation d'adresse**. Tant qu'elle n'a pas
   cliqué sur le lien, rien n'est créé : aucune fiche, aucune demande, et
   rien n'apparaît dans l'administration.
3. Une fois l'adresse confirmée, Frigg crée une fiche d'origine
   **« Publique »** et une demande d'adhésion **en attente**, puis envoie un
   lien personnel de suivi.

La demande apparaît dans *Adhésions* avec le badge « Publique » et l'auteur
« Demande publique ». Elle se valide comme toute autre
[adhésion](./memberships).

Si l'adresse appartient déjà à un compte, la demande est rangée dans l'espace
membre de ce compte : l'e-mail invite la personne à se connecter, et aucun
lien personnel n'est envoyé.

### Rattacher un doublon

Une personne déjà connue de l'association (fiche saisie ou
[importée](./member-import)) peut faire une demande publique. Frigg crée alors
une seconde fiche. À l'ouverture de la demande en attente, l'encart **Doublon
probable** désigne la fiche existante (même prénom, même nom, même date de
naissance).

Cliquez sur **Rattacher à cette fiche** et confirmez. Le dialogue nomme les
deux fiches :

- la demande passe sur la fiche existante ;
- l'adresse confirmée y est reportée si elle n'en avait pas ;
- la fiche publique, devenue vide, est supprimée.

Le rattachement est refusé si la fiche existante a déjà une demande ou une
adhésion validée sur la saison.

## Événements ouverts au public

1. Dans *Événements*, choisissez l'ouverture **« Ouvert au public »** à la
   création ou à la modification de l'événement, puis publiez-le.
2. La page de l'événement affiche l'encart **Lien public** : copiez
   l'adresse avec « Copier » et diffusez-la. Le lien ne fonctionne qu'une
   fois l'événement publié.
3. La personne qui ouvre le lien voit le titre, les dates, le lieu, la
   description, le prix et les places restantes, mais jamais la liste des
   inscrits. Elle s'inscrit avec son identité et son adresse, sans champ
   personnalisé.
4. Son inscription est **confirmée lorsqu'elle confirme son adresse**, pas
   avant : la place n'est pas réservée entre-temps. Si l'événement est
   complet à ce moment, aucune inscription n'est créée.

Si le lien a circulé trop largement, cliquez sur **Régénérer** et confirmez :
l'ancien lien affiche aussitôt « Événement introuvable ». Les inscriptions
déjà faites sont conservées. Repasser l'événement en « Toutes les fiches » ou
en « Adhérents de la saison » coupe aussi le lien public.

Les inscriptions publiques portent le badge « Publique » dans *Inscriptions*,
et suivent toutes les règles habituelles : prix figé, référence de paiement,
retrait possible par un administrateur.

## Le lien de suivi

Le **lien personnel** donne accès, sans mot de passe, à l'**espace de suivi**
(`/suivi`). Depuis cet espace, la personne voit les adhésions et les
inscriptions de ses fiches, avec leur état et ce qu'il reste à payer. Elle
peut **payer** (avec les mêmes moyens que les membres, carte comprise) et
**annuler** une demande ou une inscription. Elle ne peut modifier aucune
information : pour une correction, elle contacte l'association.

Le lien ne donne accès qu'aux fiches **sans compte** dont l'adresse est celle
qui a reçu le lien. C'est le cas des fiches publiques, mais aussi des fiches
saisies par un administrateur ou importées, dès qu'elles ont une adresse.

- Chaque e-mail de notification d'une fiche sans compte contient un lien neuf.
- La personne peut en redemander un sur `/suivi/lien` (« Je n'ai rien
  reçu »). La réponse affichée est toujours la même, que l'adresse soit
  connue ou non.
- Sur la fiche d'un membre sans compte qui a une adresse, un administrateur
  peut cliquer **Envoyer un lien de suivi**. C'est utile pour une fiche
  importée qui n'a encore jamais reçu d'e-mail.
- « Fermer l'accès sur cet appareil » retire le lien du navigateur utilisé.

::: tip Vérifier l'adresse d'une fiche sans compte
Dans l'administration, un avertissement accompagne le champ d'adresse d'une
fiche sans compte : cette adresse recevra les notifications et un accès au
suivi et au paiement. Vérifiez-la avant d'enregistrer.
:::

## Création d'un compte

La personne peut créer un compte :

- en cochant « Créer un compte » au moment de sa demande : après la
  confirmation, elle choisit son mot de passe ;
- plus tard, depuis l'encart « Créer un compte » de son espace de suivi.

Le compte est créé avec le rôle membre et l'adresse déjà confirmée. Toutes
les fiches suivies par ce lien lui sont rattachées (20 au plus). S'il y en a
plusieurs, la personne indique laquelle est la sienne (« Vous »), ou aucune.
Tous ses liens personnels cessent alors de fonctionner : elle utilise
désormais son [compte](./account).

## Fiches publiques inactives

Une **fiche publique inactive** est une fiche d'origine « Publique », sans
compte, créée il y a plus de 12 mois. Elle n'a pas non plus d'activité
récente :

- aucune demande en attente ;
- aucune adhésion validée d'une saison terminée depuis moins de 12 mois ;
- aucune inscription confirmée à un événement terminé depuis moins de
  12 mois.

Les fiches importées ou saisies par un administrateur ne sont jamais
concernées.

- Le **tableau de bord** affiche leur nombre et mène au filtre.
- Dans *Membres*, le filtre **Origine › Inactives à supprimer** les liste
  (adresse `/admin/membres?origine=inactives`).
- Le bouton **Supprimer les N fiches** les efface après une confirmation qui
  en donne le nombre. Les suppressions se font par lots de **500 fiches au
  plus** : recommencez s'il en reste. Une fiche redevenue active entre-temps
  est conservée.
- La suppression efface aussi les adhésions et les inscriptions de la fiche,
  comme une [suppression ordinaire](./members#supprimer).

::: info Aucune suppression automatique
Frigg ne supprime jamais une fiche de lui-même : il les signale, et un
administrateur décide.
:::

## Protections

- **Confirmation d'adresse** : rien n'est enregistré avant que la personne ait
  cliqué sur le lien reçu par e-mail. Une demande non confirmée est effacée
  sous 7 jours.
- **Durée des liens** : le lien de confirmation est valable **48 heures** et
  ne sert qu'une fois. Le lien de suivi est valable **30 jours**, et tous les
  liens d'une adresse sont révoqués dès qu'un compte est créé.
- **Limites** : les envois sont limités par adresse (5 e-mails de
  confirmation ou de lien par heure) et par connexion (10 demandes par heure).
  Au-delà, un message neutre invite à réessayer plus tard.
- **Discrétion** : les réponses à l'écran ne révèlent jamais si une adresse a
  un compte, une fiche ou une demande.
- **Anti-robots** : un champ piège invisible écarte les robots, sans captcha.
- Les pages publiques ne sont pas indexées par les moteurs de recherche, et
  les journaux de l'application ne contiennent ni adresse, ni nom, ni lien.
