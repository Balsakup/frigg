package api

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/season"
)

const (
	tagMyMembers = "Mes membres"
	tagMembers   = "Membres"

	MsgMemberGone           = "Ce membre n'existe plus."
	MsgFirstNameRequired    = "Saisissez le prénom."
	MsgLastNameRequired     = "Saisissez le nom."
	MsgBirthFuture          = "La date de naissance ne peut pas être dans le futur."
	MsgBirthTooOld          = "Vérifiez la date de naissance : elle remonte à plus de 120 ans."
	MsgEmailFormat          = "Saisissez une adresse électronique valide (par exemple nom@exemple.fr)."
	MsgPhoneFormat          = "Saisissez un numéro à 10 chiffres (06 12 34 56 78) ou au format international (+32 …)."
	MsgAddressIncomplete    = "Complétez l'adresse : ligne d'adresse, code postal et commune."
	MsgAddressLine2Alone    = "Saisissez d'abord la ligne d'adresse."
	MsgPostalCode           = "Saisissez un code postal à 5 chiffres."
	MsgSelfTaken            = "Une autre fiche vous désigne déjà."
	MsgDuplicate            = "Un membre portant ces nom, prénom et date de naissance existe déjà. Confirmez pour enregistrer quand même."
	MsgMyMembersFull        = "Vous gérez déjà 20 membres. Contactez le bureau pour en ajouter."
	MsgAccountMembersFull   = "Ce compte gère déjà 20 membres."
	MsgMemberAccountChanged = "Le rattachement de ce membre a changé entre-temps. Vérifiez la fiche avant d'enregistrer."
)

// PostalAddress est l'adresse postale d'un membre, complète ou absente.
type PostalAddress struct {
	Line1      string  `json:"line1"`
	Line2      *string `json:"line2"`
	PostalCode string  `json:"postalCode"`
	City       string  `json:"city"`
}

// Member est une fiche membre (research.md 005 › R1).
type Member struct {
	ID        int64          `json:"id"`
	FirstName string         `json:"firstName"`
	LastName  string         `json:"lastName"`
	BirthDate string         `json:"birthDate" format:"date"`
	Age       int            `json:"age" doc:"Âge en années révolues à la date du jour (Paris)"`
	Email     *string        `json:"email"`
	Phone     *string        `json:"phone" doc:"Normalisé : 0612345678 ou +32470123456"`
	Address   *PostalAddress `json:"address"`
	IsSelf    bool           `json:"isSelf" doc:"Fiche du titulaire du compte (« Vous »)"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// MemberAccount est le compte qui gère une fiche.
type MemberAccount struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`
}

// AdminMember est une fiche vue par l'administration, avec son rattachement.
type AdminMember struct {
	Member
	Account *MemberAccount `json:"account" doc:"Compte qui gère la fiche ; null : sans compte"`
}

// MemberBody regroupe les champs saisis dans les deux espaces.
type MemberBody struct {
	FirstName        string `json:"firstName" maxLength:"80"`
	LastName         string `json:"lastName" maxLength:"80"`
	BirthDate        string `json:"birthDate" format:"date"`
	Email            string `json:"email,omitempty" required:"false" maxLength:"254" doc:"Absente ou vide : aucune"`
	Phone            string `json:"phone,omitempty" required:"false" maxLength:"30" doc:"Absent ou vide : aucun"`
	AddressLine1     string `json:"addressLine1,omitempty" required:"false" maxLength:"100"`
	AddressLine2     string `json:"addressLine2,omitempty" required:"false" maxLength:"100"`
	PostalCode       string `json:"postalCode,omitempty" required:"false" maxLength:"10"`
	City             string `json:"city,omitempty" required:"false" maxLength:"100"`
	ConfirmDuplicate bool   `json:"confirmDuplicate,omitempty" required:"false" doc:"Vrai : enregistrer malgré un doublon probable"`
}

// memberRecord est une fiche lue en base, avec son compte éventuel.
type memberRecord struct {
	sqlc.Member
	AccountEmail      sql.NullString
	AccountDisabledAt sql.NullString
}

func fromListRow(r sqlc.ListMembersRow) memberRecord {
	return memberRecord{
		Member: sqlc.Member{
			ID: r.ID, AccountID: r.AccountID, IsSelf: r.IsSelf, FirstName: r.FirstName, LastName: r.LastName,
			BirthDate: r.BirthDate, Email: r.Email, Phone: r.Phone, AddressLine1: r.AddressLine1,
			AddressLine2: r.AddressLine2, PostalCode: r.PostalCode, City: r.City,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		},
		AccountEmail:      r.AccountEmail,
		AccountDisabledAt: r.AccountDisabledAt,
	}
}

func fromGetRow(r sqlc.GetMemberRow) memberRecord {
	return fromListRow(sqlc.ListMembersRow(r))
}

func optionalValue(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func toMember(r sqlc.Member, today season.Date) Member {
	created, _ := time.Parse(timeFormat, r.CreatedAt)
	updated, _ := time.Parse(timeFormat, r.UpdatedAt)
	m := Member{
		ID:        r.ID,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		BirthDate: r.BirthDate,
		Email:     optionalValue(r.Email),
		Phone:     optionalValue(r.Phone),
		IsSelf:    r.IsSelf == 1,
		CreatedAt: created,
		UpdatedAt: updated,
	}
	if birth, err := season.ParseDate(r.BirthDate); err == nil {
		m.Age = member.Age(birth, today)
	}
	if r.AddressLine1.Valid {
		m.Address = &PostalAddress{
			Line1:      r.AddressLine1.String,
			Line2:      optionalValue(r.AddressLine2),
			PostalCode: r.PostalCode.String,
			City:       r.City.String,
		}
	}
	return m
}

func toAdminMember(r memberRecord, today season.Date) AdminMember {
	out := AdminMember{Member: toMember(r.Member, today)}
	if r.AccountID.Valid {
		out.Account = &MemberAccount{ID: r.AccountID.Int64, Email: r.AccountEmail.String, Disabled: r.AccountDisabledAt.Valid}
	}
	return out
}

func sortable(r sqlc.Member) member.Sortable {
	return member.Sortable{ID: r.ID, FirstName: r.FirstName, LastName: r.LastName, IsSelf: r.IsSelf == 1}
}

var memberMessages = map[error]map[string]string{
	member.ErrRequired:          {"firstName": MsgFirstNameRequired, "lastName": MsgLastNameRequired},
	member.ErrBirthFuture:       {"": MsgBirthFuture},
	member.ErrBirthTooOld:       {"": MsgBirthTooOld},
	member.ErrEmailFormat:       {"": MsgEmailFormat},
	member.ErrPhoneFormat:       {"": MsgPhoneFormat},
	member.ErrAddressIncomplete: {"": MsgAddressIncomplete},
	member.ErrAddressLine2Alone: {"": MsgAddressLine2Alone},
	member.ErrPostalCode:        {"": MsgPostalCode},
}

// validateMember analyse et valide la saisie : toutes les erreurs de champ
// dans un seul 422 (FR-015).
func validateMember(body MemberBody, today season.Date) (member.Clean, error) {
	birth, err := season.ParseDate(body.BirthDate)
	if err != nil {
		return member.Clean{}, FieldErrors("body.birthDate", MsgInvalidDate)
	}
	c, err := member.Validate(member.Input{
		FirstName:    body.FirstName,
		LastName:     body.LastName,
		BirthDate:    birth,
		Email:        body.Email,
		Phone:        body.Phone,
		AddressLine1: body.AddressLine1,
		AddressLine2: body.AddressLine2,
		PostalCode:   body.PostalCode,
		City:         body.City,
	}, today)
	var fes member.FieldErrors
	if errors.As(err, &fes) {
		details := make([]*huma.ErrorDetail, len(fes))
		for i, fe := range fes {
			msgs := memberMessages[fe.Err]
			msg, ok := msgs[fe.Field]
			if !ok {
				msg = msgs[""]
			}
			details[i] = &huma.ErrorDetail{Location: "body." + fe.Field, Message: msg}
		}
		return member.Clean{}, FieldErrorDetails(details...)
	}
	return c, err
}

// memberFields sont les colonnes de données d'une fiche nettoyée.
type memberFields struct {
	FirstName, LastName, BirthDate                   string
	Email, Phone, AddressLine1, AddressLine2, Postal sql.NullString
	City                                             sql.NullString
}

func fieldsOf(c member.Clean) memberFields {
	return memberFields{
		FirstName:    c.FirstName,
		LastName:     c.LastName,
		BirthDate:    c.BirthDate.String(),
		Email:        nullString(c.Email),
		Phone:        nullString(c.Phone),
		AddressLine1: nullString(c.AddressLine1),
		AddressLine2: nullString(c.AddressLine2),
		Postal:       nullString(c.PostalCode),
		City:         nullString(c.City),
	}
}

// identityChanged indique si prénom, nom ou date de naissance diffèrent de la
// fiche enregistrée : seul ce cas relance le contrôle de doublon (R8).
func identityChanged(c member.Clean, r sqlc.Member) bool {
	return member.IdentityKey(c.FirstName, c.LastName) != member.IdentityKey(r.FirstName, r.LastName) ||
		c.BirthDate.String() != r.BirthDate
}

// checkDuplicate refuse un doublon probable non confirmé, sans révéler à quel
// compte appartient la fiche existante (FR-005, R8).
func checkDuplicate(ctx context.Context, q *sqlc.Queries, c member.Clean, excludeID int64, confirmed bool) error {
	if confirmed {
		return nil
	}
	rows, err := q.ListMembersByBirthDate(ctx, c.BirthDate.String())
	if err != nil {
		return err
	}
	key := member.IdentityKey(c.FirstName, c.LastName)
	for _, r := range rows {
		if r.ID != excludeID && member.IdentityKey(r.FirstName, r.LastName) == key {
			return FieldErrors("body.confirmDuplicate", MsgDuplicate)
		}
	}
	return nil
}

// checkAccountCapacity refuse un 21e rattachement au compte (FR-004).
func checkAccountCapacity(ctx context.Context, q *sqlc.Queries, accountID int64, msg string) error {
	n, err := q.CountMembersOfAccount(ctx, sql.NullInt64{Int64: accountID, Valid: true})
	if err != nil {
		return err
	}
	if n >= member.MaxPerAccount {
		return huma.Error409Conflict(msg)
	}
	return nil
}

// selfTaken traduit la violation de l'index members_one_self (concurrence).
func selfTaken(err error) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: members.account_id") {
		return FieldErrors("body.isSelf", MsgSelfTaken)
	}
	return err
}

// logMemberAction trace une écriture réussie sur une fiche, sans aucune de ses
// données personnelles (FR-017, R13).
func (s *server) logMemberAction(ctx context.Context, action string, memberID int64, accountID sql.NullInt64, space string) {
	attrs := []any{"action", action, "actor_id", CurrentAccount(ctx).ID, "member_id", memberID}
	if accountID.Valid {
		attrs = append(attrs, "account_id", accountID.Int64)
	}
	attrs = append(attrs, "space", space)
	s.Logger.InfoContext(ctx, "écriture sur une fiche membre", attrs...)
}

func (s *server) registerMembers(api huma.API) {
	s.registerMyMembers(api)
	s.registerAdminMembers(api)
}
