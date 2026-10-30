# Contrat d'interface : routes et shell

Chemins en français (visibles par l'utilisateur), fichiers de route et
composants en anglais. Rendu de référence : l'artefact désigné par la
constitution (principe IV).

## Routes publiques (hors shell, gabarit « authentification »)

Mise en page centrée (`max-width: 30rem`), nom « Frigg » en tête (le nom de
l'association est une donnée de l'instance, non affichée hors connexion —
principe II), pas de barre latérale. Une personne déjà connectée qui ouvre `/connexion` est
redirigée vers son accueil.

| Chemin | Fichier de route | Contenu |
| --- | --- | --- |
| `/connexion?suite=` | `routes/auth/login.tsx` | Adresse, mot de passe, lien « Mot de passe oublié ? » ; message de session expirée si `?expiree=1` |
| `/mot-de-passe-oublie` | `routes/auth/forgot-password.tsx` | Adresse ; message de confirmation neutre |
| `/reinitialiser-mot-de-passe#<jeton>` | `routes/auth/reset-password.tsx` | Nouveau mot de passe (+ confirmation côté client), règle affichée |
| `/confirmer-adresse#<jeton>` | `routes/auth/confirm-email.tsx` | Confirmation automatique au chargement, puis lien vers l'accueil |
| `/invitation#<jeton>` | `routes/auth/accept-invitation.tsx` | Adresse invitée (lecture seule), choix du mot de passe |

La confirmation du mot de passe (« les deux saisies correspondent ») est une
vérification d'interface, pas une règle métier : elle ne part pas au serveur.

## Routes protégées (dans le shell)

Layout `routes/app/layout.tsx` : `clientLoader` → session ; 401 →
`/connexion?suite=…`.

| Chemin | Fichier de route | Accès | Contenu |
| --- | --- | --- | --- |
| `/` | `routes/app/member/home.tsx` | membre | Accueil de l'espace membre |
| `/mon-compte` | `routes/app/account.tsx` | membre | Identité, rôle, changement de mot de passe |
| `/admin` | `routes/app/admin/home.tsx` | admin | Accueil de l'espace administration |
| `/admin/comptes` | `routes/app/admin/accounts.tsx` | admin | Formulaire d'invitation + liste des invitations (renvoyer, annuler) |
| rubriques métier | `routes/app/coming-soon.tsx` | selon rubrique | Page « Bientôt disponible » (FR-025) |
| `*` | `routes/app/not-found.tsx` | membre | « Page introuvable » dans le shell |

Le layout `routes/app/admin/layout.tsx` affiche « Accès refusé » dans le shell
pour un membre (FR-023) ; le serveur refuse de toute façon les données.

## Navigation

| Espace | Navigation principale | Ressources (secondaire) |
| --- | --- | --- |
| Membre | Accueil `/` · Mes membres · Mes inscriptions · À régler · Événements à venir · Adhésions · Mes documents | Aide |
| Administration | Tableau de bord `/admin` · Membres · Adhésions · Événements · Inscriptions · Finances · Communication | Comptes `/admin/comptes` · Aide |

Les rubriques sans chemin ci-dessus pointent vers `coming-soon` sous
`/<rubrique>` ou `/admin/<rubrique>`. La liste est déclarée une seule fois
(`app/shell/navigation.ts`) avec, pour chaque entrée : identifiant, libellé,
icône lucide, chemin, espace.

## Shell (FR-018 à FR-024)

- **Barre latérale** : identité (monogramme + nom de l'association + « Espace
  membre / administration »), sélecteur d'espace (administrateurs seulement),
  navigation principale, ressources, menu du compte (adresse, rôle, « Mon
  compte », « Se déconnecter »). Repliable en icônes ≥ 1024 px (état mémorisé),
  en surimpression en dessous (fermeture : bouton, Échap, clic extérieur).
- **En-tête** : bouton de la barre latérale uniquement (FR-024).
- **Contenu** : alerte « Confirmez votre adresse électronique » si
  `emailVerified = false` (bouton « Renvoyer le lien »), fil d'Ariane, en-tête de
  page (titre `h1` focusable, introduction, actions), zone de page.
- **Pied de page** : « © {année} {association} · Propulsé par Frigg v{version} ».
- **Accessibilité** : liens d'évitement (contenu, navigation), focus déplacé sur
  le `h1` à chaque navigation, `aria-current="page"` sur l'élément actif,
  annonce du titre de page aux lecteurs d'écran.
