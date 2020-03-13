package participant

import (
	"html/template"
	"net/http"
	"time"

	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/conference"
)

type service struct {
	*application.Application
	templates templates
}

const (
	// Before the conference, no login allowed.
	stateBefore = iota
	// Data entry and login allowed.
	stateLoginEdit
	// Data entry allowed, no login.
	stateEdit
	// After the conference, no data entry or login allowed.
	stateAfter
)

type requestContext struct {
	application.RequestContext
	Participant *conference.Participant
	state       int
}

func New() application.Service { return &service{} }

func (s *service) Setup(app *application.Application) (string, interface{}, error) {
	s.Application = app
	return "participant", &s.templates, nil
}

func (s *service) Execute(fn interface{}, w http.ResponseWriter, r *http.Request) {
	rc := &requestContext{}
	err := rc.Init(s.Application, w, r)
	if err != nil {
		s.handleError(rc, err)
		return
	}

	since := time.Since(rc.Conference.Date)
	if s.TimeOverride != 0 {
		since = s.TimeOverride
	}

	switch {
	case since < 0:
		rc.state = stateBefore
	case since < 24*time.Hour:
		rc.state = stateLoginEdit
	case since < 48*time.Hour:
		rc.state = stateEdit
	default:
		rc.state = stateAfter
	}

	if rc.state == stateLoginEdit || rc.state == stateEdit {
		if c, _ := rc.Request.Cookie("id"); c != nil {
			if s, ok := application.VerifySignature(rc.Conference.Configuration.CookieKey, c.Value); ok {
				parts, err := application.DecodeStringsFromCookie(s)
				if err == nil && len(parts) == 1 {
					id := parts[0]
					rc.Participant = rc.Conference.Participant(id)
				}
			}
		}
	}

	err = fn.(func(*service, *requestContext) error)(s, rc)
	if err != nil {
		s.handleError(rc, err)
		return
	}
}

func (s *service) handleError(rc *requestContext, err error) {
	e := rc.ConvertError(err)
	rc.HandleError(rc.Respond(s.templates.Error, e.Status, e))
}

func (rc *requestContext) Respond(t *template.Template, status int, v interface{}) error {
	return rc.RequestContext.Respond(t, status, struct {
		*requestContext
		Data interface{}
	}{rc, v})
}

func (rc *requestContext) setParticipantID(protocol string, id string) {
	if id == "" {
		http.SetCookie(rc.Response, &http.Cookie{
			Name:     "id",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			HttpOnly: true,
			Secure:   protocol == "https",
			SameSite: http.SameSiteStrictMode,
		})
		return
	}

	config := rc.Conference.Configuration
	const maxAge = 2 * 24 * 3600
	http.SetCookie(rc.Response, &http.Cookie{
		Name: "id",
		Value: application.SignValue(
			config.CookieKey,
			maxAge,
			application.EncodeStringsForCookie(id)),
		MaxAge:   maxAge,
		Path:     "/",
		HttpOnly: true,
		Secure:   protocol == "https",
		SameSite: http.SameSiteStrictMode,
	})
}
