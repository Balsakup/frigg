# Quickstart : valider les paramètres généraux et l'identité visuelle

Guide de validation de bout en bout. Contrats : [api.md](./contracts/api.md),
[ui-routes.md](./contracts/ui-routes.md). Modèle : [data-model.md](./data-model.md).
Prérequis : ceux de [006 › quickstart.md](../006-memberships/quickstart.md)
(Mailpit, `create-admin`, `go tool task dev`), un compte membre invité, et
trois images de test : un PNG transparent de ~200 Ko, une photo JPEG avec
données EXIF de géolocalisation, un fichier SVG.

## Contrôles automatisés (constitution)

```sh
go tool task check
```

Attendu en plus de 009 :

- `CGO_ENABLED=0 go list -deps ./cmd/frigg` inclut `golang.org/x/image/webp`
  et `golang.org/x/image/draw`, et aucun paquet CGO ;
- `00010_association_settings.sql` s'applique sur une base vierge et sur une
  base `00009` peuplée : e-mails en attente conservés avec `wrap = 0`, nom de
  l'association conservé ;
- matrice d'autorisation : 10 opérations ajoutées (`get-identity` et
  `get-identity-logo` publiques et listées dans `allowedPublic` avec la
  référence à l'amendement 1.8.0 ; `get-association-contact` `member` ; sept
  opérations `admin`) ;
- tests en table : `internal/association` (RNA, SIRET dont La Poste,
  téléphone métropole et outre-mer, URL, date future), `internal/brand`
  (≥ 4,5:1 pour chaque couple de la palette sur un balayage de couleurs),
  traitement du logo (formats, bombe de pixels, EXIF retiré, variante e-mail
  ≤ 100 Ko) ;
- `internal/mail` : `Wrap` avec et sans logo, worker habillant les lignes
  `wrap = 1` avec l'identité courante ;
- Playwright + axe : `general-settings.spec.ts`, `appearance.spec.ts`, sans
  violation `serious` ni `critical`.

## Scénarios manuels

### 1. Nom et tagline (US1)

1. Administrateur → « Paramètres » : l'adresse devient
   `/admin/parametres/general`.
2. Changer le nom, saisir la tagline « Randonnées et sorties nature depuis
   1987 », enregistrer. **Attendu** : notification ; barre latérale, titre
   d'onglet et pied de page à jour sans rechargement ; tagline sous le nom.
3. Vider le nom, enregistrer. **Attendu** : `422`, message sous « Nom »,
   rien d'enregistré.
4. Se connecter en membre. **Attendu** : tagline dans la barre latérale et
   dans l'en-tête de l'accueil ; `/admin/parametres/general` refusé.

### 2. Identité légale et coordonnées (US2)

1. Saisir `w751234567`, SIRET `732 829 320 00074`, une date passée, un code
   postal, un téléphone `+33 1 23 45 67 89`. **Attendu** : enregistré et
   réaffiché en `W751234567`, `73282932000074`, `01 23 45 67 89`.
2. Saisir en même temps un SIRET à clé fausse, une date de demain et un site
   `www.exemple.fr`. **Attendu** : trois erreurs signalées en une fois ;
   recharger : aucune des valeurs du deuxième essai n'est enregistrée.

### 3. Apparence (US3)

1. « Apparence » : choisir la photo JPEG. **Attendu** : aperçu immédiat ;
   après enregistrement, logo dans la barre latérale. Télécharger
   `/api/identity/logo/<nom>` : plus de données EXIF (`exiftool`), côté le
   plus long ≤ 512 px.
2. Choisir le SVG. **Attendu** : refus sur le champ, logo actuel conservé.
3. Couleur principale `#fff3b0` (jaune pâle). **Attendu** : l'aperçu montre
   un texte de bouton foncé et des liens assombris ; le reste de l'interface
   ne change qu'après l'enregistrement ; axe sans violation de contraste.
4. Se déconnecter. **Attendu** : écran de connexion avec le logo, le nom et
   les couleurs de l'association, sans clignotement de l'identité de Frigg.
   `curl -s localhost:8080/api/identity` : uniquement `name`, `logoUrl`,
   `palette`. `curl` sur `/api/admin/settings/general` et
   `/api/association/contact` : `401`.
5. Demander une réinitialisation de mot de passe ; ouvrir le message dans
   Mailpit. **Attendu** : bandeau aux couleurs, logo embarqué (`cid:logo`,
   pièce `inline`), bouton aux couleurs, nom lisible avec les images bloquées.
6. Retirer le logo (confirmation) puis rétablir les couleurs (confirmation).
   **Attendu** : initiales et couleurs de Frigg partout ; l'ancienne adresse
   du logo répond `404` ; le répertoire `FRIGG_DATA_DIR/logo/` est vide.

### 4. Contact (US4)

1. Renseigner adresse de contact, téléphone, Instagram et Facebook.
   **Attendu** : encart « Contacter l'association » sur l'accueil membre ;
   `mailto:`, `tel:+33…`, liens externes en `noopener noreferrer`.
2. Vider tous ces champs. **Attendu** : encart absent.

### 5. E-mails en attente pendant un changement (R6)

1. Couper Mailpit, envoyer une communication à quelques adhérents, changer le
   nom et la couleur principale, relancer Mailpit. **Attendu** : les messages
   partent avec le nouveau nom en bandeau et les nouvelles couleurs.
