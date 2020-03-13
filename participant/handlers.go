package participant

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/conference"
	"github.com/seaptc/seaptc/log"
)

type templates struct {
	After,
	Before,
	Eval1,
	Eval2,
	Home,
	Login,
	Error *template.Template `template:".,root.html,../common.html"`
}

func (s *service) Serve_(rc *requestContext) error {
	switch {
	case rc.Request.URL.Path != "/":
		rc.Participant = nil
		return application.ErrNotFound
	case rc.Participant != nil:
		return s.serveHome(rc)
	case rc.state == stateLoginEdit:
		return rc.Respond(s.templates.Login, http.StatusOK, nil)
	case rc.state == stateBefore:
		return rc.Respond(s.templates.Before, http.StatusOK, nil)
	default:
		return rc.Respond(s.templates.After, http.StatusOK, nil)
	}
}

func (s *service) Serve_login(rc *requestContext) error {
	if rc.IsPost() {
		p := rc.Conference.ParticipantFromLoginCode(rc.FormValue("loginCode"))
		if p == nil {
			rc.MarkInputInvalid("loginCode")
		}
		if !rc.HasInvalidInput() {
			rc.setParticipantID(s.Protocol, p.ID)
			http.Redirect(rc.Response, rc.Request, "/", http.StatusSeeOther)
			return nil
		}
	}
	return rc.Respond(s.templates.Login, http.StatusOK, nil)
}

func (s *service) Serve_logout(rc *requestContext) error {
	rc.setParticipantID(s.Protocol, "")
	http.Redirect(rc.Response, rc.Request, "/", http.StatusSeeOther)
	return nil
}

func (s *service) serveHome(rc *requestContext) error {
	if rc.Participant == nil {
		http.Redirect(rc.Response, rc.Request, "/", http.StatusSeeOther)
		return nil
	}

	eval, err := s.Store.GetEvaluation(rc.Ctx, rc.Participant.ID)
	if err != nil {
		return err
	}

	var evaluatedClasses []*conference.SessionClass
	for _, se := range eval.Sessions {
		class := rc.Conference.Class(se.ClassNumber)
		if class == nil {
			log.Logf(rc.Ctx, log.Error, "class %d missing for evaluation %s", se.ClassNumber, rc.Participant.ID)
			continue
		}
		evaluatedClasses = append(evaluatedClasses, &conference.SessionClass{Class: class, Session: se.Session})
	}
	sort.Slice(evaluatedClasses, func(i, j int) bool { return evaluatedClasses[i].Session < evaluatedClasses[j].Session })

	data := struct {
		Schedule            []*conference.ScheduleItem
		EvaluatedClasses    []*conference.SessionClass
		EvaluatedConference bool
	}{
		Schedule:            rc.Conference.ParticipantSchedule(rc.Participant),
		EvaluatedClasses:    evaluatedClasses,
		EvaluatedConference: eval.Conference != nil,
	}

	return rc.Respond(s.templates.Home, http.StatusOK, &data)
}

func (s *service) Serve_eval(rc *requestContext) error {
	if rc.Participant == nil {
		http.Redirect(rc.Response, rc.Request, "/", http.StatusSeeOther)
		return nil
	}

	evalCode := rc.FormValue("evalCode")

	data := struct {
		SessionClass         *conference.SessionClass
		SessionEvaluation    *conference.SessionEvaluation
		ConferenceEvaluation *conference.ConferenceEvaluation
		EvaluateConference   bool
		EvaluateSession      bool
		IsInstructor         bool
	}{
		EvaluateConference:   evalCode == "conference",
		SessionEvaluation:    &conference.SessionEvaluation{},
		ConferenceEvaluation: &conference.ConferenceEvaluation{},
	}

	if !data.EvaluateConference {
		data.SessionClass = rc.Conference.SessionClassFromEvaluationCode(evalCode)
		if data.SessionClass == nil {
			if rc.FormValue("submit") != "" {
				rc.MarkInputInvalid("evalCode")
			}
			return rc.Respond(s.templates.Eval1, http.StatusOK, &data)
		}
		data.EvaluateSession = true
		data.EvaluateConference = data.SessionClass.Session == conference.NumSession-1

		sessionClasses := rc.Conference.ParticipantSessionClasses(rc.Participant)
		if sessionClasses[data.SessionClass.Session].Number == data.SessionClass.Number &&
			sessionClasses[data.SessionClass.Session].Instructor {
			data.IsInstructor = true
		}
	}

	if !rc.IsPost() {
		eval, err := s.Store.GetEvaluation(rc.Ctx, rc.Participant.ID)
		if err != nil {
			return err
		}

		if data.EvaluateSession {
			for _, se := range eval.Sessions {
				if se.Session == data.SessionClass.Session {
					data.SessionEvaluation = se
					break
				}
			}
		}
		if data.EvaluateConference && eval.Conference != nil {
			data.ConferenceEvaluation = eval.Conference
		}
		return rc.Respond(s.templates.Eval2, http.StatusOK, &data)
	}

	getRating := func(name string, required bool) int {
		n, _ := strconv.Atoi(rc.FormValue(name))
		if required && (n < 1 || n > 4) {
			rc.MarkInputInvalid(name)
		}
		return n
	}

	var eval conference.Evaluation
	if data.EvaluateSession {
		se := data.SessionEvaluation
		eval.SetSession(se)
		se.Session = data.SessionClass.Session
		se.ClassNumber = data.SessionClass.Number
		se.Updated = time.Now().In(conference.TimeLocation)
		se.Source = "participant"
		se.Comments = rc.FormValue("comments")
		if !data.IsInstructor {
			for name, pv := range se.Ratings() {
				*pv = getRating(name, true)
			}
		}
	}

	if data.EvaluateConference {
		ce := data.ConferenceEvaluation
		eval.Conference = ce
		ce.Updated = time.Now().In(conference.TimeLocation)
		ce.Source = "participant"
		for name, pv := range ce.Ratings() {
			*pv = getRating(name, false)
		}
		ce.LearnTopics = rc.FormValue("learnTopics")
		ce.TeachTopics = rc.FormValue("teachTopics")
		ce.Comments = rc.FormValue("confComments")
	}

	if rc.HasInvalidInput() {
		return rc.Respond(s.templates.Eval2, http.StatusOK, &data)
	}

	err := s.Store.SetEvaluation(rc.Ctx, rc.Participant.ID, &eval)
	if err != nil {
		return err
	}
	return rc.Redirect("/", application.FlashInfo, "Evaluation recorded.")
}
