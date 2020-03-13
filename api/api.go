package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/conference"
)

type service struct {
	*application.Application
}

type requestContext struct {
	application.RequestContext
}

func New() application.Service { return &service{} }

func (s *service) Setup(app *application.Application) (string, interface{}, error) {
	s.Application = app
	return "dashboard", nil, nil
}

func (s *service) Execute(fn interface{}, w http.ResponseWriter, r *http.Request) {
	rc := &requestContext{}
	err := rc.Init(s.Application, w, r)
	if err != nil {
		s.handleError(rc, err)
		return
	}
	err = fn.(func(*service, *requestContext) error)(s, rc)
	if err != nil {
		s.handleError(rc, err)
		return
	}
}

func (s *service) handleError(rc *requestContext, err error) {
	e := rc.ConvertError(err)
	rc.HandleError(rc.Respond(e.Status, map[string]interface{}{"error": e.Message}))
}

func (rc *requestContext) Respond(status int, data interface{}) error {
	data = map[string]interface{}{"result": data}
	p, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return err
	}
	rc.Response.Header().Set("Content-Type", "application/json")
	rc.Response.Header().Set("Content-Length", strconv.Itoa(len(p)))
	rc.Response.Write(p)
	return nil
}

type sessionEvent struct {
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	New          string   `json:"titleNew"` // rename to avoid js reserved word
	TitleNote    string   `json:"titleNote"`
	Description  string   `json:"description"`
	StartSession int      `json:"startSession"`
	EndSession   int      `json:"endSession"`
	StartTime    []int    `json:"startTime"` // year, month, day, hour, minute
	EndTime      []int    `json:"endTime"`   // year, month, day, hour, minute
	Capacity     int      `json:"capacity"`  // 0: no limit, -1 no space
	Programs     []string `json:"programs"`
}

func (rc *requestContext) createSessionEvent(class *conference.Class) *sessionEvent {
	start := conference.SessionTimes[class.Start].Start
	end := conference.SessionTimes[class.End].End

	var programs []string
	if class.Programs != (1<<conference.NumPrograms)-1 {
		for _, pd := range class.ProgramDescriptions(false) {
			programs = append(programs, pd.Name)
		}
	}

	year, month, day := rc.Conference.Date.Date()

	return &sessionEvent{
		Number:       class.Number,
		New:          class.New,
		Title:        class.Title,
		TitleNote:    class.TitleNote,
		Description:  class.Description,
		StartSession: class.Start + 1,
		EndSession:   class.End + 1,
		StartTime:    []int{year, int(month), day, int(start / time.Hour), int((start % time.Hour) / time.Minute)},
		EndTime:      []int{year, int(month), day, int(end / time.Hour), int((end % time.Hour) / time.Minute)},
		Capacity:     class.Capacity,
		Programs:     programs,
	}
}

func (rc *requestContext) createSpecialSessionEvent(number int,
	title string, description string,
	start, end time.Duration) *sessionEvent {

	year, month, day := rc.Conference.Date.Date()
	return &sessionEvent{
		Number:      number,
		Title:       title,
		Description: description,
		StartTime:   []int{year, int(month), day, int(start / time.Hour), int((start % time.Hour) / time.Minute)},
		EndTime:     []int{year, int(month), day, int(end / time.Hour), int((end % time.Hour) / time.Minute)},
	}
}

func (s *service) Serve_api_sessionEvents_(rc *requestContext) error {
	numString := strings.TrimPrefix(rc.Request.URL.Path, "/api/sessionEvents/")
	number, err := strconv.Atoi(numString)
	if err != nil {
		return &application.HTTPError{Status: http.StatusNotFound, Message: fmt.Sprintf("Class %q not found.", numString)}
	}

	var se *sessionEvent
	switch number {
	case conference.NoClassClassNumber:
		se = rc.createSpecialSessionEvent(number,
			"No classes (select if not taking classes at the conference)",
			"Select this activity to indicate that you are not taking classes at the conference.",
			conference.SessionTimes[0].Start, conference.SessionTimes[conference.NumSession-1].End)
	default:
		class := rc.Conference.Class(number)
		if class == nil {
			return &application.HTTPError{
				Status:  http.StatusNotFound,
				Message: fmt.Sprintf("Class %q not found.", numString),
			}
		}
		se = rc.createSessionEvent(class)
	}

	return rc.Respond(http.StatusOK, se)
}
