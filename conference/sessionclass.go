package conference

import (
	"fmt"
	"log"
)

type SessionClass struct {
	*Class
	Session    int
	Instructor bool
}

func (sc *SessionClass) NumberDotPart() string {
	if sc.Start >= sc.End {
		return fmt.Sprintf("%d", sc.Number)
	}
	return fmt.Sprintf("%d.%d", sc.Number, sc.part())
}

func (sc *SessionClass) IofN() string {
	if sc.Start >= sc.End {
		return ""
	}
	return fmt.Sprintf(" (%d of %d)", sc.part(), sc.Length())
}

func (sc *SessionClass) part() int {
	return sc.Session - sc.Start + 1
}

func (sc *SessionClass) EvaluationCode() string {
	i := sc.Session - sc.Start
	if 0 <= i && i < len(sc.EvaluationCodes) {
		return sc.EvaluationCodes[i]
	}
	return ""
}

var noClass = &Class{Title: "No Class", Start: 0, End: 0}

func (conf *Conference) ParticipantSessionClassesAndLunch(p *Participant) ([]*SessionClass, *Lunch) {
	sessionClasses := make([]*SessionClass, NumSession)
	for i := range sessionClasses {
		sessionClasses[i] = &SessionClass{Session: i, Class: noClass}
	}

	for _, n := range p.Classes {
		c := conf.Class(n)
		if c == nil {
			log.Printf("unknown class %d for participant %v", n, p.ID)
			continue
		}
		for i := c.Start; i <= c.End; i++ {
			sc := sessionClasses[i]
			sc.Class = c
		}
	}

	for i, n := range conf.instructorClasses[p.ID] {
		if n <= 0 {
			continue
		}
		c := conf.Class(n)
		if c == nil {
			log.Printf("unknown instructor class %d for participant %v", n, p.ID)
			continue
		}
		sc := sessionClasses[i]
		sc.Class = c
		sc.Instructor = true
	}

	c := sessionClasses[LunchSession]

	conf.setupLunch()
	lunch := conf.lunch.byClass[c.Number]
	if lunch == nil {
		lunch = conf.lunch.byUnitType[p.UnitType]
	}
	if lunch == nil {
		lunch = conf.GeneralLunch()
	}

	return sessionClasses, lunch
}

func (conf *Conference) ParticipantSessionClasses(p *Participant) []*SessionClass {
	scs, _ := conf.ParticipantSessionClassesAndLunch(p)
	return scs
}

func (conf *Conference) ParticipantLunch(p *Participant) *Lunch {
	_, lunch := conf.ParticipantSessionClassesAndLunch(p)
	return lunch
}
