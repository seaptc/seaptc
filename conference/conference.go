package conference

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	NumSession         = 6
	LunchSession       = 2
	NoClassClassNumber = 999
)

var TimeLocation = func() *time.Location {
	l, err := time.LoadLocation("US/Pacific")
	if err != nil {
		log.Fatal(err)
	}
	return l
}()

func noReverse(fn func(i, j int) bool) func(i, j int) bool {
	return fn
}

func reverse(fn func(i, j int) bool) func(i, j int) bool {
	return func(i, j int) bool {
		return fn(j, i)
	}
}

func SortKeyReverse(key string) (string, func(func(int, int) bool) func(int, int) bool) {
	switch {
	case key == "":
		return "", noReverse
	case key[0] == '-':
		return key[1:], reverse
	default:
		return key, noReverse
	}
}

// Conference holds most of the data associated with the conference.
type Conference struct {
	classes                 []*Class
	classesByNumber         map[int]*Class
	participants            []*Participant
	instructorClasses       map[string][]int
	participantsByID        map[string]*Participant
	participantsByLoginCode map[string]*Participant

	Configuration *Configuration
	Date          time.Time

	evalCode struct {
		once  sync.Once
		value map[string]*SessionClass
	}

	sessions struct {
		once  sync.Once
		value [][]*SessionClass
	}

	ids struct {
		once  sync.Once
		staff map[string]bool
		admin map[string]bool
	}

	lunch struct {
		once       sync.Once
		def        *Lunch
		byClass    map[int]*Lunch
		byUnitType map[string]*Lunch
	}
}

func New() *Conference {
	return &Conference{
		Configuration: newConfiguration(),
	}
}

// copy creates a shallow copy of the conference.
func (conf *Conference) copy() *Conference {
	var newConf Conference
	newConf.classes = conf.classes
	newConf.classesByNumber = conf.classesByNumber
	newConf.participants = conf.participants
	newConf.participantsByID = conf.participantsByID
	newConf.participantsByLoginCode = conf.participantsByLoginCode
	newConf.instructorClasses = conf.instructorClasses
	newConf.Configuration = conf.Configuration
	newConf.Date = conf.Date
	return &newConf
}

func (conf *Conference) UpdateConfiguration(config *Configuration) *Conference {
	newConf := conf.copy()
	newConf.Configuration = config
	newConf.Date = time.Date(config.Year, time.Month(config.Month), config.Day, 0, 0, 0, 0, TimeLocation)
	return newConf
}

func (conf *Conference) UpdateParticipants(participants []*Participant) *Conference {
	newConf := conf.copy()
	newConf.participants = participants
	newConf.participantsByID = make(map[string]*Participant, len(participants))
	newConf.participantsByLoginCode = make(map[string]*Participant, len(participants))
	for _, p := range participants {
		p.init()
		newConf.participantsByID[p.ID] = p
		newConf.participantsByLoginCode[p.LoginCode] = p
	}
	return newConf
}

func (conf *Conference) UpdateInstructorClasses(instructorClasses map[string][]int) *Conference {
	newConf := conf.copy()
	newConf.instructorClasses = instructorClasses
	return newConf
}

func (conf *Conference) UpdateClasses(classes []*Class) *Conference {
	newC := conf.copy()
	newC.classes = classes
	newC.classesByNumber = map[int]*Class{}
	for _, c := range classes {
		newC.classesByNumber[c.Number] = c
	}
	return newC
}

func (conf *Conference) Classes() []*Class {
	// Clone so that caller can sort and filter.
	return append(([]*Class)(nil), conf.classes...)
}

func (conf *Conference) Participants() []*Participant {
	// Clone so that caller can sort and filter.
	return append(([]*Participant)(nil), conf.participants...)
}

func (conf *Conference) Class(number int) *Class {
	return conf.classesByNumber[number]
}

func (conf *Conference) ParticipantInstructorClasses(p *Participant) []int {
	classNumbers := conf.instructorClasses[p.ID]
	if classNumbers == nil {
		classNumbers = make([]int, NumSession)
	}
	return classNumbers
}

func (conf *Conference) PrintSignature(p *Participant) string {
	var buf []byte
	for i, n := range p.Classes {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendInt(buf, int64(n), 36)
	}
	buf = append(buf, '|')
	for i, n := range conf.ParticipantInstructorClasses(p) {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendInt(buf, int64(n), 36)
	}
	return string(buf)
}

func (conf *Conference) Participant(id string) *Participant {
	return conf.participantsByID[id]
}

func (conf *Conference) ParticipantFromLoginCode(loginCode string) *Participant {
	return conf.participantsByLoginCode[loginCode]
}

func (conf *Conference) setupIDs() {
	conf.ids.once.Do(func() {
		conf.ids.staff = make(map[string]bool)
		conf.ids.admin = make(map[string]bool)
		for _, id := range conf.Configuration.StaffIDs {
			conf.ids.staff[strings.ToLower(id)] = true
		}
		for _, id := range conf.Configuration.AdminIDs {
			id = strings.ToLower(id)
			conf.ids.staff[id] = true
			conf.ids.admin[id] = true
		}
	})
}

func (conf *Conference) IsStaff(id string) bool {
	if id == "" {
		return false
	}
	conf.setupIDs()
	return conf.ids.staff[id]
}

func (conf *Conference) IsAdmin(id string) bool {
	if id == "" {
		return false
	}
	conf.setupIDs()
	return conf.ids.admin[id]
}

func (conf *Conference) ClassParticipants(c *Class) []*Participant {
	var result []*Participant
	for _, p := range conf.participants {
		for _, n := range p.Classes {
			if n == c.Number {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

func (conf *Conference) SessionClassFromEvaluationCode(evaluationCode string) *SessionClass {
	conf.evalCode.once.Do(func() {
		conf.evalCode.value = make(map[string]*SessionClass)
		for _, c := range conf.classes {
			for i, code := range c.EvaluationCodes {
				conf.evalCode.value[code] = &SessionClass{Class: c, Session: c.Start + i}
			}
		}
	})
	return conf.evalCode.value[evaluationCode]
}

func (conf *Conference) Sessions() [][]*SessionClass {
	conf.sessions.once.Do(func() {
		conf.sessions.value = make([][]*SessionClass, NumSession)
		for _, c := range conf.classes {
			for i := c.Start; i <= c.End; i++ {
				if i >= NumSession {
					continue
				}
				conf.sessions.value[i] = append(conf.sessions.value[i], &SessionClass{Class: c, Session: i})
			}
		}
	})
	return conf.sessions.value
}

func (conf *Conference) setupLunch() {
	conf.lunch.once.Do(func() {
		conf.lunch.byClass = make(map[int]*Lunch)
		conf.lunch.byUnitType = make(map[string]*Lunch)
		for _, l := range conf.Configuration.Lunches {
			for _, n := range l.Classes {
				conf.lunch.byClass[n] = l
			}
			for _, unitType := range l.UnitTypes {
				conf.lunch.byUnitType[unitType] = l
			}
		}
	})
}

var (
	tbdLunch     = &Lunch{Seating: 2, Name: "TBD", ShortName: "TBD", Location: "TBD"}
	programLunch = &Lunch{Seating: 2, Name: "Lunch location depends on participant unit type", ShortName: "*"}
)

func (conf *Conference) ClassLunch(c *Class) *Lunch {
	if c.End < LunchSession || c.Start > LunchSession {
		return nil
	}
	conf.setupLunch()
	l := conf.lunch.byClass[c.Number]
	if l == nil {
		l = programLunch
	}
	return l
}

func (conf *Conference) GeneralLunch() *Lunch {
	if len(conf.Configuration.Lunches) == 0 {
		return tbdLunch
	}
	return conf.Configuration.Lunches[0]
	return nil
}

func (conf *Conference) resolveClassNumbers(classNumbers []int) []*Class {
	if len(classNumbers) == 0 {
		return nil
	}
	result := make([]*Class, 0, len(classNumbers))
	for _, n := range classNumbers {
		if c := conf.Class(n); c != nil {
			result = append(result, c)
		}
	}
	return result
}

/*
func (conf *Conference) RelatedClasses(p *Participant) []map[int][]string {

	addRelatedClass := func(c *model.Class, description string) {
		start, end := c.StartEnd()
		for i := start; i <= end; i++ {
			m := data.RelatedClasses[i]
			if m == nil {
				m = make(map[int][]string)
				data.RelatedClasses[i] = m
			}
			m[c.Number] = append(m[c.Number], description)
		}
	}

	for _, classNumber := range participant.Classes {
		c := conf.Class(classNumber)
		if c != nil {
			addRelatedClass(c, "class")
		}
	}

	for _, classNumber := range participant.InstructorClasses {
		c := conf.Class(classNumber)
		if c != nil {
			addRelatedClass(c, "inst")
		}
	}

	for _, classNumber := range participant.RejectedInstructorClasses {
		c := conf.Class(classNumber)
		if c != nil {
			addRelatedClass(c, "rejected")
		}
	}

	if participant.StaffRole == model.StaffRoleInstructor {
		for _, n := range model.ClassNumberPat.FindAllString(participant.StaffDescription, -1) {
			classNumber, _ := strconv.Atoi(n)
			c := conf.Class(classNumber)
			if c != nil {
				addRelatedClass(c, "staff desc")
			}
		}
	}

	participantName := participant.Name()
	for _, c := range conf.Classes() {
		for _, email := range c.InstructorEmails {
			if email == participant.Email {
				addRelatedClass(c, "sheet email")
			}
		}
		for _, name := range c.InstructorNames {
			if name == participantName {
				addRelatedClass(c, "sheet name")
			}
		}
	}

*/
