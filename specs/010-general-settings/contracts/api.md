# Contrat d'API : paramètres généraux et identité visuelle

Mêmes conventions que [006 › api.md](../../006-memberships/contracts/api.md) :
préfixe `/api`, camelCase, `problem+json` en français avec `errors[].location`
sur les `422`, `401` sans session, `403` rôle insuffisant. Le schéma OpenAPI
produit par `frigg openapi` fait foi. Tag OpenAPI `settings` pour les
opérations d'administration, `association` pour les autres.

## Types

```text
Hex = string                      // ^#[0-9a-f]{6}$ (majuscules acceptées en entrée, renvoyé en minuscules)

Palette {
  brand: Hex, brandInk: Hex, onBrand: Hex,
  accent: Hex, accentInk: Hex, onAccent: Hex
}

Identity {                        // exactement ces trois clés (principe II v1.8.0)
  name: string,                   // « Mon association » si jamais défini
  logoUrl: string|null,           // "/api/identity/logo/<nom généré>"
  palette: Palette
}

Address {
  line1: string|null, line2: string|null,   // ≤ 100
  postalCode: string|null,                  // ^[0-9]{5}$
  city: string|null                         // ≤ 100
}

SocialLinks {                     // chaque valeur : URL http(s) ≤ 300 ou null
  facebook, instagram, linkedin, youtube, mastodon, bluesky
}

GeneralSettings {
  name: string,                   // 1..120 après suppression des espaces
  tagline: string|null,           // ≤ 150
  shortName: string|null,         // ≤ 20
  purpose: string|null,           // ≤ 2000, plusieurs lignes
  rna: string|null,               // W + 9 chiffres
  siret: string|null,             // 14 chiffres, clé valide ; espaces acceptés en entrée
  foundedOn: date|null,           // AAAA-MM-JJ, ≤ aujourd'hui (Paris)
  address: Address,
  contactEmail: string|null,      // format email, ≤ 254
  phone: string|null,             // français ; renvoyé en forme nationale "0123456789"
  website: string|null,           // URL http(s) ≤ 300
  social: SocialLinks
}

Appearance {
  primaryColor: Hex|null,         // null = couleur par défaut de Frigg
  accentColor: Hex|null,
  palette: Palette,               // dérivée (research R3)
  logo: Logo|null
}

Logo {
  url: string,                    // variante interface
  originalName: string,           // pour l'administrateur seulement
  width: int, height: int,
  uploadedAt: datetime
}

AssociationContact {              // aucun champ d'identité légale
  address: Address, contactEmail: string|null, phone: string|null,
  website: string|null, social: SocialLinks
}

Association (dans Session) {
  name: string, tagline: string|null, logoUrl: string|null
}
```

Une chaîne vide ou faite d'espaces en entrée vaut `null` (FR-014), sauf pour
`name`, où elle produit un `422` sur `body.name`.

## Opérations publiques (principe II v1.8.0)

Ajoutées à `allowedPublic` dans `internal/api/access_test.go`.

| OperationID | Méthode et chemin | Réponse |
| --- | --- | --- |
| `get-identity` | `GET /identity` | `200 Identity` ; `Cache-Control: no-cache` |
| `get-identity-logo` | `GET /identity/logo/{name}` | `200` octets de la variante interface, `Content-Type` détecté, `Content-Disposition: inline`, `X-Content-Type-Options: nosniff`, `Cache-Control: public, max-age=31536000, immutable` ; `404` si `name` n'est pas le nom courant ou ne respecte pas `^[0-9a-f]{32}$` |

Aucune autre donnée de l'association, aucune version ni configuration n'y
figure ; un test vérifie la liste exacte des clés de `Identity`.

## Opérations membre

| OperationID | Méthode et chemin | Rôle | Réponse |
| --- | --- | --- | --- |
| `get-session` (modifiée) | `GET /session` | `member` | `association` passe à `{name, tagline, logoUrl}` |
| `get-association-contact` | `GET /association/contact` | `member` | `200 AssociationContact` |

## Opérations d'administration (`admin`)

| OperationID | Méthode et chemin | Corps | Réponse |
| --- | --- | --- | --- |
| `get-general-settings` | `GET /admin/settings/general` | — | `200 GeneralSettings` |
| `update-general-settings` | `PUT /admin/settings/general` | `GeneralSettings` | `200 GeneralSettings` (normalisés) ; `422` groupé, rien n'est enregistré |
| `get-appearance` | `GET /admin/settings/appearance` | — | `200 Appearance` |
| `update-appearance` | `PUT /admin/settings/appearance` | `{primaryColor: Hex|null, accentColor: Hex|null}` | `200 Appearance` ; `422` sur un code invalide |
| `preview-palette` | `GET /admin/settings/appearance/palette?primary=&accent=` | — | `200 Palette` ; paramètre absent = couleur par défaut ; rien n'est enregistré |
| `upload-logo` | `PUT /admin/settings/appearance/logo` | multipart, champ `file` (≤ 1 Mio ; corps ≤ 1 Mio + 16 Kio) | `200 Appearance` ; `422 form.file` : format refusé, image illisible ou trop grande en pixels ; `413` : corps trop gros |
| `delete-logo` | `DELETE /admin/settings/appearance/logo` | — | `204` ; idempotent (`204` sans logo) |

### Messages d'erreur (`422`)

| Champ | Message |
| --- | --- |
| `name` | « Indiquez le nom de l'association. » / « Le nom ne doit pas dépasser 120 caractères. » |
| `rna` | « Le numéro RNA commence par W suivi de 9 chiffres (W123456789). » |
| `siret` | « Le numéro SIRET comporte 14 chiffres. » / « Ce numéro SIRET n'est pas valide. Vérifiez-le sur l'avis de situation Sirene. » |
| `foundedOn` | « La date de création ne peut pas être dans le futur. » |
| `address.postalCode` | « Le code postal comporte 5 chiffres. » |
| `contactEmail` | « Saisissez une adresse électronique valide. » |
| `phone` | « Saisissez un numéro de téléphone français (01 23 45 67 89). » |
| `website`, `social.*` | « Saisissez une adresse web complète, commençant par https://. » |
| `primaryColor`, `accentColor` | « Saisissez une couleur au format #1f5278. » |
| `file` | « Déposez une image PNG, JPEG ou WebP de 1 Mo au plus. » (format refusé ou fichier de plus de 1 048 576 octets ; même message affiché sur le champ pour un `413`) / « Cette image est trop grande : 8 000 pixels de côté au plus. » |
| (longueurs) | message existant `MsgTextTooLong` |

## Effets de bord

- Toute écriture réussie met à jour `association.updated_at` et journalise
  l'action ([research.md › R9](../research.md#r9-journalisation)).
- `upload-logo` et `delete-logo` suppriment les fichiers précédents après la
  validation de la transaction.
- Les e-mails mis en file après cette fonctionnalité portent `wrap = 1` :
  nom, couleurs et logo sont appliqués au moment de l'envoi.
