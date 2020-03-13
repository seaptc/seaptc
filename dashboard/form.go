package dashboard

import (
	"net/http"
	"sort"
	"strings"

	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/conference"
	"rsc.io/qr"
)

var formOptions = map[string]*struct {
	filter bool
	auto   int
	limit  int
	sort   func([]*conference.Participant)
}{
	"auto": {
		filter: true,
		auto:   60,
		limit:  50,
		sort: func(participants []*conference.Participant) {
			sort.Slice(participants, func(i, j int) bool {
				return conference.DefaultParticipantLess(participants[j], participants[i])
			})
		}},
	"batch": {
		filter: true,
		limit:  50,
		sort: func(participants []*conference.Participant) {
			// Sort by staff role and name
			sort.Slice(participants, func(i, j int) bool {
				irole := participants[i].StaffRole
				jrole := participants[j].StaffRole
				switch {
				case irole != jrole:
					return jrole < irole
				default:
					return conference.DefaultParticipantLess(participants[j], participants[i])
				}
			})
		}},
	"first": {
		sort: func(participants []*conference.Participant) {
			// Descending by length of first name.
			sort.Slice(participants, func(i, j int) bool {
				a := len(participants[j].FirstName)
				b := len(participants[i].FirstName)
				switch {
				case a != b:
					return a < b
				default:
					return conference.DefaultParticipantLess(participants[i], participants[j])
				}
			})
		}},
	"last": {
		sort: func(participants []*conference.Participant) {
			// Descending by length of last name.
			sort.Slice(participants, func(i, j int) bool {
				a := len(participants[j].Name()) - len(participants[j].FirstName)
				b := len(participants[i].Name()) - len(participants[i].FirstName)
				switch {
				case a != b:
					return a < b
				default:
					return conference.DefaultParticipantLess(participants[i], participants[j])
				}
			})
		}},
}

func (s *service) Serve_dashboard_forms(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}

	if rc.IsPost() {
		printSignatures := make(map[string]string)
		for _, idsig := range rc.Request.Form["idsig"] {
			if i := strings.Index(idsig, "/"); i > 0 {
				printSignatures[idsig[:i]] = idsig[i+1:]
			}
		}
		if err := s.Store.SetPrintSignatures(rc.Ctx, printSignatures); err != nil {
			return err
		}
	}

	options := formOptions[rc.Request.FormValue("options")]
	if options == nil {
		options = formOptions["batch"]
	}
	auto := options.auto

	participants := rc.Conference.Participants()
	options.sort(participants)

	if options.filter {
		printSignatures, err := s.Store.GetPrintSignatures(rc.Ctx)
		if err != nil {
			return err
		}
		participants = conference.FilterParticipants(
			participants,
			func(p *conference.Participant) bool {
				return rc.Conference.PrintSignature(p) != printSignatures[p.ID]
			})
	}

	if options.limit > 0 && len(participants) > options.limit {
		participants = participants[:options.limit]
		// Swtich to manual mode to avoid getting stuck in print-refresh loop.
		auto = 0
	}

	return s.renderForms(rc, auto, !options.filter, participants)
}

func (s *service) Serve_dashboard_forms_(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}
	p := rc.Conference.Participant(strings.TrimPrefix(rc.Request.URL.Path, "/dashboard/forms/"))
	if p == nil {
		return application.ErrNotFound
	}

	return s.renderForms(rc, 0, true, []*conference.Participant{p})
}

func (s *service) renderForms(rc *requestContext, auto int, preview bool, participants []*conference.Participant) error {
	var data = struct {
		Participants   []*conference.Participant
		Lunch          interface{}
		SessionClasses interface{}
		Auto           int
		Preview        bool
	}{
		Auto:         auto,
		Preview:      preview,
		Participants: participants,
	}
	return rc.Respond(s.templates.Form, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_blankForm(rc *requestContext) error {

	var data = struct {
		Participants   []*conference.Participant
		Lunch          interface{}
		SessionClasses interface{}
		Auto           int
		Preview        bool
	}{
		Participants: []*conference.Participant{&conference.Participant{}},
		Lunch:        func(p *conference.Participant) interface{} { return nil },
		SessionClasses: func(p *conference.Participant) []*conference.SessionClass {
			var result []*conference.SessionClass
			for i := 0; i < conference.NumSession; i++ {
				result = append(result, &conference.SessionClass{Session: i, Class: &conference.Class{}})
			}
			return result
		},
		Auto:    0,
		Preview: false,
	}
	return rc.Respond(s.templates.Form, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_vcard(rc *requestContext) error {
	vcard := []byte("BEGIN:VCARD\r\nVERSION:4.0\r\n")
	for name, values := range rc.Request.Form {
		value := strings.TrimSpace(values[0])
		if value == "" {
			continue
		}
		vcard = append(vcard, name...)
		vcard = append(vcard, ':')
		for i := range value {
			b := value[i]
			switch b {
			case '\\':
				vcard = append(vcard, `\\`...)
			case '\n':
				vcard = append(vcard, `\n`...)
			case '\r':
				vcard = append(vcard, `\r`...)
			case ',':
				vcard = append(vcard, `\,`...)
			case ':':
				vcard = append(vcard, `\:`...)
			case ';':
				vcard = append(vcard, `\;`...)
			default:
				vcard = append(vcard, b)
			}
		}
		vcard = append(vcard, "\r\n"...)
	}
	vcard = append(vcard, "END:VCARD\r\n"...)
	code, err := qr.Encode(string(vcard), qr.L)
	if err != nil {
		return err
	}
	rc.Response.Header().Set("Content-Type", "image/png")
	rc.Response.Write(code.PNG())
	return nil
}
