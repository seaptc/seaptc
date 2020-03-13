package conference

import (
	"crypto/md5"
	"fmt"
	"time"
)

// MaxEvalRating is the maximum value for an evaluation rating. The rating
// values are:
//  0 - not specified,
//  1 - minimum;
//  ...
//  MaxEvalRating - maximum
const MaxEvalRating = 4

type SessionEvaluation struct {
	// TODO: add instructor?
	Session            int       `json:"session"`
	ClassNumber        int       `json:"class"`
	KnowledgeRating    int       `json:"knowledge"`
	PresentationRating int       `json:"promotion"`
	UsefulnessRating   int       `json:"usefulness"`
	OverallRating      int       `json:"overall"`
	Comments           string    `json:"comments"`
	Source             string    `json:"source"`
	Updated            time.Time `json:"updated"`
}

type ConferenceEvaluation struct {
	ExperienceRating        int       `json:"experience"`
	PromotionRating         int       `json:"promotion"`
	RegistrationRating      int       `json:"registration"`
	CheckinRating           int       `json:"checkin"`
	MidwayRating            int       `json:"midway"`
	LunchRating             int       `json:"lunch"`
	FacilitiesRating        int       `json:"facilities"`
	WebsiteRating           int       `json:"website"`
	SignageWayfindingRating int       `json:"signageWayfinding"`
	LearnTopics             string    `json:"learnTopics"`
	TeachTopics             string    `json:"teachTopics"`
	Comments                string    `json:"comments"`
	Source                  string    `json:"source"`
	Updated                 time.Time `json:"updated"`
}

type EvaluationNote struct {
	Text   string `json:"note"`
	NoShow bool   `json:"noShow"`
}

type Evaluation struct {
	ParticipantID string
	Conference    *ConferenceEvaluation `json:"conference"`
	Sessions      []*SessionEvaluation  `json:"sessions"`
	Note          *EvaluationNote       `json:"note"`
}

func (e *Evaluation) SetSession(se *SessionEvaluation) {
	// Overwrite existing.
	for i := range e.Sessions {
		if e.Sessions[i].Session == se.Session {
			if se.ClassNumber != 0 {
				e.Sessions[i] = se
			} else {
				e.Sessions = append(e.Sessions[:i], e.Sessions[i+1:]...)
			}
			return
		}
	}
	e.Sessions = append(e.Sessions, se)
}

func (se *SessionEvaluation) Hash() string {
	buf := []byte{
		byte(se.Session),
		byte(se.ClassNumber),
		byte(se.ClassNumber >> 8),
		byte(se.KnowledgeRating),
		byte(se.PresentationRating),
		byte(se.UsefulnessRating),
		byte(se.OverallRating),
	}
	buf = append(buf, se.Comments...)
	sum := md5.Sum(buf)
	return fmt.Sprintf("%x", sum[:])
}

func (ce *ConferenceEvaluation) Hash() string {
	buf := []byte{
		byte(ce.ExperienceRating),
		byte(ce.PromotionRating),
		byte(ce.RegistrationRating),
		byte(ce.CheckinRating),
		byte(ce.MidwayRating),
		byte(ce.LunchRating),
		byte(ce.FacilitiesRating),
		byte(ce.WebsiteRating),
		byte(ce.SignageWayfindingRating),
	}
	buf = append(buf, ce.LearnTopics...)
	buf = append(buf, 0)
	buf = append(buf, ce.TeachTopics...)
	buf = append(buf, 0)
	buf = append(buf, ce.Comments...)
	sum := md5.Sum(buf)
	return fmt.Sprintf("%x", sum[:])
}

func (en *EvaluationNote) Hash() string {
	noShow := byte(0)
	if en.NoShow {
		noShow = 1
	}

	buf := []byte{noShow}
	buf = append(buf, en.Text...)
	sum := md5.Sum(buf)
	return fmt.Sprintf("%x", sum[:])
}

func (se *SessionEvaluation) Ratings() map[string]*int {
	return map[string]*int{
		"knowledge":    &se.KnowledgeRating,
		"presentation": &se.PresentationRating,
		"usefulness":   &se.UsefulnessRating,
		"overall":      &se.OverallRating,
	}
}

func (ce *ConferenceEvaluation) Ratings() map[string]*int {
	return map[string]*int{
		"experience":        &ce.ExperienceRating,
		"promotion":         &ce.PromotionRating,
		"registration":      &ce.RegistrationRating,
		"checkin":           &ce.CheckinRating,
		"midway":            &ce.MidwayRating,
		"lunch":             &ce.LunchRating,
		"facilities":        &ce.FacilitiesRating,
		"website":           &ce.WebsiteRating,
		"signageWayfinding": &ce.SignageWayfindingRating,
	}
}
