package dashboard

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/seaptc/seaptc/application"
)

type service struct {
	*application.Application
	templates templates
}

type requestContext struct {
	application.RequestContext
	StaffID string
}

func New() application.Service { return &service{} }

func (s *service) Setup(app *application.Application) (string, interface{}, error) {
	s.Application = app
	return "dashboard", &s.templates, nil
}

func (s *service) Execute(fn interface{}, w http.ResponseWriter, r *http.Request) {
	rc := &requestContext{}
	err := rc.Init(s.Application, w, r)
	if err != nil {
		s.handleError(rc, err)
		return
	}

	if c, _ := rc.Request.Cookie("staff"); c != nil {
		if s, ok := application.VerifySignature(rc.Conference.Configuration.CookieKey, c.Value); ok {
			parts, err := application.DecodeStringsFromCookie(s)
			if err == nil && len(parts) == 1 {
				rc.StaffID = strings.ToLower(parts[0])
			}
		}
	}

	if rc.ConfFromCache && rc.Conference.IsStaff(rc.StaffID) {
		var err error
		rc.Conference, _, err = s.Store.GetConference(rc.Ctx, true)
		if err != nil {
			s.handleError(rc, err)
			return
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

func (rc *requestContext) IsStaff() bool {
	return rc.Conference.IsStaff(rc.StaffID)
}

func (rc *requestContext) IsAdmin() bool {
	return rc.Conference.IsAdmin(rc.StaffID)
}
