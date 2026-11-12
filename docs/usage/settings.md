# Paramètres de l'association

**Rôle requis** : Administrateur

*Configuration › Paramètres* regroupe les réglages de l'association en quatre
sections : **Général**, **Apparence**, **Champs personnalisés** (voir
[Champs personnalisés](./custom-fields)) et **Paiements** (voir
[Paiement par carte](./card-payment)).

Quitter une section avec des modifications non enregistrées demande
confirmation.

## Général

| Champ | Règle |
| --- | --- |
| Nom | Obligatoire, 120 caractères au plus |
| Tagline | Facultative, 150 caractères au plus. Affichée sous le nom dans la barre latérale et sur l'accueil des membres |
| Nom court | Facultatif, 20 caractères au plus |
| Objet | Facultatif, 2 000 caractères au plus |
| Numéro RNA | `W` suivi de 9 chiffres |
| SIRET | 14 chiffres et clé de contrôle valide |
| Date de création | Ne peut pas être dans le futur |
| Adresse du siège | Ligne d'adresse, complément, code postal à 5 chiffres, ville |
| Adresse électronique, téléphone, site web | Vérifiés s'ils sont renseignés. Le téléphone est un numéro français, enregistré au format `01 23 45 67 89` |
| Réseaux sociaux | Facebook, Instagram, LinkedIn, YouTube, Mastodon, Bluesky : adresses web complètes |

Tous les champs sauf le nom sont facultatifs. L'enregistrement est tout ou
rien : si un champ est invalide, rien n'est enregistré et le champ en erreur
est signalé.

Les coordonnées et les réseaux sociaux sont présentés aux membres dans un
encart « Contacter l'association », sur leur accueil. L'identité légale (RNA,
SIRET, date de création) reste réservée aux administrateurs.

## Apparence

- **Logo** : image PNG, JPEG ou WebP de 1 Mo au plus (SVG refusé). Il est
  réencodé à l'enregistrement, ce qui retire les métadonnées de la photo
  (EXIF, position GPS). Sans logo, les initiales de l'association le
  remplacent.
- **Couleur principale** et **couleur d'accent** : choisies au sélecteur ou
  par un code hexadécimal. Un aperçu montre le résultat avant
  l'enregistrement. Les textes posés sur ces couleurs, ou écrits dans ces
  teintes, sont assombris automatiquement pour rester lisibles (contraste
  d'au moins 4,5:1).
- **Rétablir les couleurs par défaut** de Frigg, après confirmation.

Les couleurs de succès, d'avertissement et d'erreur ne changent pas.

Le nom, le logo et les couleurs habillent toute l'application, l'écran de
connexion compris : ce sont les seules informations de l'association visibles
sans être connecté. Ils habillent aussi les e-mails, au moment de leur envoi.
