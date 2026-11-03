# Contrat d'API : communications

Mêmes conventions que [006 › api.md](../../006-memberships/contracts/api.md) :
préfixe `/api`, camelCase, `problem+json` en français avec `errors[].location`
sur les erreurs `422`, `401` sans session, `403` rôle insuffisant. Le schéma
OpenAPI produit par `frigg openapi` fait foi. **Toutes les opérations sont
`admin`** (FR-021) ; tag OpenAPI `communications`.

## Types

```text
Situation = "up_to_date" | "announced" | "due"
TargetPreset = "all" | "up_to_date" | "unpaid" | "custom"     // calculé, lecture seule

Target {
  seasonId: int,
  situations: Situation[],          // 1..3, sans doublon
  feeIds: int[]                     // 0..50 ; [] = tous les tarifs de la saison
}

AudienceMember {
  memberId: int, firstName: string, lastName: string,
  situation: Situation, feeLabel: string,
  contactEmail: string|null         // null = non joignable
}

Audience {
  target: Target, preset: TargetPreset, targetLabel: string,
  membersCount: int,                // adhérents ciblés, non joignables compris
  unreachableCount: int,
  recipientsCount: int,             // adresses distinctes
  members: AudienceMember[],        // joignables, tri nom puis prénom
  unreachable: AudienceMember[]
}

Attachment {
  id: int, name: string, contentType: string, sizeBytes: int
}

CommunicationSummary {
  id: int, subject: string, targetLabel: string, preset: TargetPreset,
  author: AccountRef|null,          // null = compte supprimé
  recipientsCount: int, attachmentsCount: int,
  delivery: { pending: int, sent: int, failed: int },
  createdAt: datetime
}

Communication = CommunicationSummary + {
  body: string,
  season: { id: int, label: string },
  target: Target,
  membersCount: int, unreachableCount: int,
  attachments: Attachment[],
  recipients: Recipient[]           // tri : failed, pending, sent puis adresse
}

Recipient {
  email: string|null,               // null = adresse effacée (compte supprimé)
  membersCount: int,
  status: "pending" | "sent" | "failed",
  sentAt: datetime|null
}

Preview { subject: string, text: string, html: string }
```

## Opérations

| Opération | Méthode et chemin | Succès |
| --- | --- | --- |
| `get-communication-audience` | `GET /admin/communications/audience` | `200 Audience` |
| `upload-communication-attachment` | `POST /admin/communications/attachments` | `201 Attachment` |
| `delete-communication-attachment` | `DELETE /admin/communications/attachments/{id}` | `204` |
| `preview-communication` | `POST /admin/communications/preview` | `200 Preview` |
| `send-communication` | `POST /admin/communications` | `201 Communication` (`200` si déjà envoyée) |
| `list-communications` | `GET /admin/communications` | `200 { items: CommunicationSummary[], total: int }` |
| `get-communication` | `GET /admin/communications/{id}` | `200 Communication` |
| `download-communication-attachment` | `GET /admin/communications/{id}/attachments/{attachmentId}` | `200` fichier |

### `get-communication-audience`

Query : `seasonId` (requis), `situations` (requis, répété :
`situations=announced&situations=due`), `feeIds` (facultatif, répété).

- `422` : saison inexistante ou en brouillon (`location: query.seasonId`),
  situation inconnue, tarif n'appartenant pas à la saison
  (`location: query.feeIds`).
- Une cible sans adhérent renvoie `200` avec des compteurs à 0.

### `upload-communication-attachment`

`multipart/form-data`, champs :

- `draftKey` : UUID v4 (`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`) ;
- `file` : un fichier.

`MaxBodyBytes` = 5 Mio + 64 Kio.

- `413` « Le fichier dépasse 5 Mo. » (corps trop volumineux) ;
- `422` sur `form.file` avec l'un des messages (huma place les erreurs d'un
  formulaire multipart sous `form.*`) :
  - « Ce type de fichier n'est pas accepté. Formats acceptés : PDF, JPEG, PNG,
    Word (.docx), Excel (.xlsx), OpenDocument (.odt, .ods). »
  - « Le fichier dépasse 5 Mo. »
  - « Le fichier est vide. »
  - « Une communication accepte au plus 5 pièces jointes. »
  - « Les pièces jointes dépassent 10 Mo au total (7,2 Mo déjà joints). »
- `422` sur `form.draftKey` : clé mal formée, ou clé déjà utilisée par une
  communication envoyée (« Cette communication a déjà été envoyée. »).

### `delete-communication-attachment`

`404` si la pièce n'existe pas, appartient à un autre administrateur ou est
rattachée à une communication envoyée (réponse identique, pas d'information
sur l'existence).

### `preview-communication`

Corps : `{ seasonId: int, subject: string(1..150), body: string(1..10000) }`.
Renvoie le message rendu (objet, version texte, version HTML) avec le nom de
l'association et la mention de saison. Aucune écriture.

### `send-communication`

Corps :

```text
{
  draftKey: string,                 // UUID v4, voir ci-dessus
  target: Target,
  subject: string(1..150),          // espaces de bord retirés avant contrôle
  body: string(1..10000),
  expectedRecipients: int(>= 1)     // décompte présenté à l'administrateur
}
```

- `201 Communication` : communication enregistrée, e-mails en file.
- `200 Communication` : `draftKey` déjà envoyée ; renvoie la communication
  existante, sans nouvel envoi (FR-014), quel que soit le reste du corps.
- `409` `{ detail: "Le nombre de destinataires a changé…", audience: Audience }`
  : décompte recalculé différent de `expectedRecipients` (FR-013). Le corps
  `problem+json` porte le champ d'extension `audience`.
- `422` : champ invalide ; cible sans destinataire (`location: body.target`,
  « Aucun destinataire ne correspond à cette cible. ») — en pratique une cible
  devenue vide renvoie `409`, `expectedRecipients` valant au moins 1 ; pièces de la
  préparation hors limites ; pièce introuvable ou altérée sur le disque
  (« Une pièce jointe est illisible ; retirez-la et ajoutez-la de nouveau. »).

### `list-communications`

Query : `page` (≥ 1, défaut 1), `perPage` (1..100, défaut 25). Tri : plus
récente d'abord.

### `get-communication`

`404` si inexistante.

### `download-communication-attachment`

En-têtes : `Content-Type` (type détecté), `Content-Length`,
`Content-Disposition: attachment; filename="<ascii>"; filename*=UTF-8''<encodé>`,
`X-Content-Type-Options: nosniff`, `Cache-Control: private, no-store`.
`404` si la pièce n'appartient pas à la communication `{id}` ou si elle n'est
pas encore rattachée à une communication envoyée. `500` générique si le fichier
manque sur le disque (journalisé).

## Opérations existantes modifiées

| Opération | Changement |
| --- | --- |
| `delete-account` (002) | Efface aussi l'adresse du compte de `communication_recipients` (research R12) ; les e-mails en attente supprimés passent leur destinataire en « en échec » (déclencheur) |

Aucune autre opération n'est modifiée. Aucune opération publique n'est ajoutée.
