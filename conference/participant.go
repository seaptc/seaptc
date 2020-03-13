package conference

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	StaffRoleInstructor = "Instructor"
	StaffRoleMidway     = "Midway"
	StaffRoleSupport    = "Support"
)

type Participant struct {
	ID                 string    `json:"id"`
	RegistrationNumber string    `json:"registrationNumber"`
	RegisteredByName   string    `json:"registeredByName"`
	RegisteredByEmail  string    `json:"registeredByEmail"`
	RegisteredByPhone  string    `json:"registeredByPhone"`
	RegistrationTime   time.Time `json:"registrationTime"`
	FirstName          string    `json:"firstName"`
	LastName           string    `json:"lastName"`
	Nickname           string    `json:"nickname"`
	Suffix             string    `json:"suffix"`
	NameExtra          string    `json:"nameExtra"`
	Staff              bool      `json:"staff"`
	Youth              bool      `json:"youth"`
	Phone              string    `json:"phone"`
	Email              string    `json:"email"`
	Address            string    `json:"address"`
	City               string    `json:"city"`
	State              string    `json:"state"`
	Zip                string    `json:"zip"`
	StaffRole          string    `json:"staffRole"` // Instructor, Support, Midway
	Council            string    `json:"council"`
	District           string    `json:"district"`
	UnitType           string    `json:"unitType"`
	UnitNumber         string    `json:"unitNumber"`
	LunchOption        string    `json:"lunchOption"`
	Marketing          string    `json:"marketing"`
	ScoutingYears      string    `json:"scoutingYears"`
	ShowQRCode         bool      `json:"showQRCode"`
	BSANumber          string    `json:"bsaNumber"`
	Classes            []int     `json:"classes"`
	StaffDescription   string    `json:"staffDescription"`

	LoginCode string `json:"loginCode"`
	sortName  string
}

// ParticipantID returns a hash of unique participant fields.
func ParticipantID(p *Participant) string {
	var buf []byte
	buf = append(buf, p.LastName...)
	buf = append(buf, 0)
	buf = append(buf, p.FirstName...)
	buf = append(buf, 0)
	buf = append(buf, p.Suffix...)
	buf = append(buf, 0)
	if p.Nickname != "" {
		buf = append(buf, strings.ToLower(p.Nickname)...)
		buf = append(buf, 0)
	}
	buf = append(buf, p.RegistrationNumber...)
	if p.Youth {
		buf = append(buf, 0, 1, 0)
	}
	buf = append(buf, p.NameExtra...)
	sum := md5.Sum(bytes.ToLower(buf))
	return hex.EncodeToString(sum[:])
}

// Type returns a short description of the participant's registration type.
func (p *Participant) Type() string {
	switch {
	case p.Staff:
		return "Staff"
	case p.Youth:
		return "Youth"
	default:
		return "Adult"
	}
}

func (p *Participant) Unit() string {
	if p.UnitNumber == "" {
		return p.UnitType
	}
	return p.UnitType + " " + p.UnitNumber
}

func (p *Participant) Name() string {
	if p.Suffix != "" {
		return p.FirstName + " " + p.LastName + " " + p.Suffix
	}
	return p.FirstName + " " + p.LastName
}

func (p *Participant) NicknameOrFirstName() string {
	if p.Nickname != "" {
		return p.Nickname
	}
	return p.FirstName
}

// Firsts returns Name's or Nickname's.
func (p *Participant) Firsts() string {
	n := p.NicknameOrFirstName()
	if n == "" {
		return ""
	}
	if strings.HasSuffix(n, "s") {
		return n + "'"
	}
	return n + "'s"
}

func (p *Participant) Emails() []string {
	if !p.Youth || p.Email == p.RegisteredByEmail {
		return []string{p.Email}
	}
	return []string{p.RegisteredByEmail, p.Email}
}

// init initializes derived fields.
func (p *Participant) init() {
	p.sortName = strings.ToLower(fmt.Sprintf("%s\n%s\n%s", p.LastName, p.FirstName, p.Suffix))
}

func DefaultParticipantLess(a, b *Participant) bool {
	return a.sortName < b.sortName
}

func SortParticipants(participants []*Participant, key string) {
	key, reverse := SortKeyReverse(key)
	switch key {
	case "type":
		sort.Slice(participants, reverse(func(i, j int) bool {
			switch {
			case participants[i].Youth != participants[j].Youth:
				return participants[i].Youth
			case participants[i].Staff != participants[j].Staff:
				return !participants[i].Staff
			case participants[i].StaffRole != participants[j].StaffRole:
				return participants[i].StaffRole < participants[j].StaffRole
			default:
				return participants[i].sortName < participants[j].sortName
			}
		}))
	case "unit", "district", "council":
		sort.Slice(participants, reverse(func(i, j int) bool {
			switch {
			case participants[i].Council != participants[j].Council:
				return participants[i].Council < participants[j].Council
			case participants[i].District != participants[j].District:
				return participants[i].District < participants[j].District
			case participants[i].UnitNumber != participants[j].UnitNumber:
				return participants[i].UnitNumber < participants[j].UnitNumber
			case participants[i].UnitType != participants[j].UnitType:
				return participants[i].UnitType < participants[j].UnitType
			default:
				return participants[i].sortName < participants[j].sortName
			}
		}))
	default:
		sort.Slice(participants, reverse(func(i, j int) bool { return participants[i].sortName < participants[j].sortName }))
	}
}

// FilterParticipants filters the slice in place.
func FilterParticipants(participants []*Participant, fn func(*Participant) bool) []*Participant {
	i := 0
	for _, p := range participants {
		if fn(p) {
			participants[i] = p
			i++
		}
	}
	return participants[:i]
}
