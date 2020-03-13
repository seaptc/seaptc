package catalog

import (
	"html/template"
	"net/http"
	"strings"
	"sync"

	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/conference"
)

type templates struct {
	Admin,
	Classes,
	Configuration,
	Index,
	Error *template.Template `template:".,root.html,../common.html"`
}

type service struct {
	*application.Application

	templates struct {
		Index,
		Program,
		All,
		New,
		Error *template.Template `template:".,root.html"`
	}

	mu    sync.RWMutex
	conf  *conference.Conference
	pages map[string][]byte
}

type requestContext struct {
	application.RequestContext
}

func New() application.Service { return &service{} }

func (s *service) Setup(app *application.Application) (string, interface{}, error) {
	s.Application = app
	return "catalog", &s.templates, nil
}

func (s *service) Execute(fn interface{}, w http.ResponseWriter, r *http.Request) {
	var rc requestContext
	err := rc.Init(s.Application, w, r)
	if err != nil {
		s.handleError(&rc, err)
		return
	}
	err = fn.(func(*service, *requestContext) error)(s, &rc)
	if err != nil {
		s.handleError(&rc, err)
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

func (s *service) Serve_register(rc *requestContext) error {
	u := rc.Conference.Configuration.RegistrationURL
	if u == "" {
		u = "https://seatlebsa.org/PTC"
	}
	http.Redirect(rc.Response, rc.Request, u, http.StatusSeeOther)
	return nil
}

func (s *service) Serve_directions(rc *requestContext) error {
	http.Redirect(rc.Response, rc.Request, "https://seattlebsa.org/ptc-documents/712-ptc-map/file", http.StatusSeeOther)
	return nil
}

func (s *service) Serve_catalog_(rc *requestContext) error {
	return s.serve_page(rc, s.templates.All, -1, true)
}

func (s *service) Serve_catalog_new(rc *requestContext) error {
	return s.serve_page(rc, s.templates.New, -1, false)
}

func (s *service) Serve_catalog_cub(rc *requestContext) error {
	return s.serve_page(rc, s.templates.Program, conference.CubScoutProgram, false)
}

func (s *service) Serve_catalog_bsa(rc *requestContext) error {
	return s.serve_page(rc, s.templates.Program, conference.ScoutsBSAProgram, false)
}

func (s *service) Serve_catalog_ven(rc *requestContext) error {
	return s.serve_page(rc, s.templates.Program, conference.VenturingProgram, false)
}

func (s *service) Serve_catalog_sea(rc *requestContext) error {
	return s.serve_page(rc, s.templates.Program, conference.SeaScoutProgram, false)
}

func (s *service) Serve_catalog_com(rc *requestContext) error {
	return s.serve_page(rc, s.templates.Program, conference.CommissionerProgram, false)
}

func (s *service) Serve_catalog_you(rc *requestContext) error {
	return s.serve_page(rc, s.templates.Program, conference.YouthProgram, false)
}

func (s *service) serve_page(rc *requestContext, t *template.Template, program int, grid bool) error {
	classes := rc.Conference.Classes()

	// Ingore classes with negative requested capacity.
	i := 0
	for _, c := range classes {
		if c.Capacity < 0 {
			continue
		}
		classes[i] = c
		i++
	}
	classes = classes[:i]

	conference.SortClasses(classes, "number")

	var (
		morningGrid   [][]*catalogClass
		afternoonGrid [][]*catalogClass
	)
	if grid {
		morningGrid = createCatalogGrid(classes, true)
		afternoonGrid = createCatalogGrid(classes, false)
	}

	var data = struct {
		Morning            [][]*catalogClass
		Afternoon          [][]*catalogClass
		Classes            []*conference.Class
		Key                []*conference.ProgramDescription
		Program            *conference.ProgramDescription
		Title              string
		SuggestedSchedules []*catalogSuggestedSchedule
	}{
		Morning:   morningGrid,
		Afternoon: afternoonGrid,
		Key:       conference.ProgramDescriptions,
		Classes:   classes,
	}

	if program >= 0 {
		data.Program = conference.ProgramDescriptions[program]
		data.Title = strings.Title(data.Program.Name)
		data.Classes = nil
		mask := 1 << uint(program)
		for _, c := range classes {
			if c.Programs&mask != 0 {
				data.Classes = append(data.Classes, c)
			}
		}
	}
	return rc.Respond(t, http.StatusOK, &data)
}

type catalogClass struct {
	Number    int
	Length    int
	Title     string
	TitleNote string
	Flag      bool
}

type catalogSuggestedSchedule struct {
	Name    string
	Classes []*catalogClass
}

func createCatalogGrid(classes []*conference.Class, morning bool) [][]*catalogClass {
	// Separate classes into rows.

	rows := make([][]*catalogClass, 100)
	for _, c := range classes {
		start, end := c.Start, c.End
		if start < 0 || end >= conference.NumSession {
			continue
		}

		cc := &catalogClass{
			Number:    c.Number,
			Length:    c.Length(),
			Title:     c.Title,
			TitleNote: c.TitleNote,
		}

		i := c.Number % 100
		row := rows[i]
		if row == nil {
			row = make([]*catalogClass, 3)
			rows[i] = row
		}

		if morning {
			if start > 2 {
				continue
			}
			if end > 2 {
				cc.Length = 3 - start
				cc.Flag = true
			}
			row[start] = cc
		} else {
			if end < 3 {
				continue
			}
			if start < 3 {
				cc.Length = end - 2
				cc.Flag = true
				start = 3
			}
			row[start-3] = cc
		}
	}

	// Remove unused rows, add dummy classes

	noclass := &catalogClass{Length: 1}
	i := 0
	for _, row := range rows {
		if row == nil {
			continue
		}
		rows[i] = row
		i++

		for j := 0; j < len(row); {
			if row[j] != nil {
				j += row[j].Length
				continue
			}
			row[j] = noclass
			j++
		}
	}
	return rows[:i]
}
