package dashboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/conference"
	"github.com/seaptc/seaptc/dk"
	"github.com/seaptc/seaptc/log"
	"github.com/seaptc/seaptc/sheet"
)

type templates struct {
	Admin,
	Class,
	Classes,
	Configuration,
	EvalCode,
	Evaluation,
	Index,
	LunchCount,
	LunchList,
	Participant,
	Participants,
	Error *template.Template `template:".,root.html,../common.html"`

	Form          *template.Template `template:".,../common.html"`
	LunchStickers *template.Template `template:"."`
	Classrooms    *template.Template `template:"."`
}

func (s *service) Serve_dashboard_(rc *requestContext) error {
	return application.ErrNotFound
}

func (s *service) Serve_dashboard(rc *requestContext) error {
	data := struct {
		Councils  map[string]int
		Districts map[string]map[string]int
		Types     map[string]int
		Total     int
	}{
		make(map[string]int),
		make(map[string]map[string]int),
		make(map[string]int),
		0,
	}

	for _, p := range rc.Conference.Participants() {
		data.Total++
		data.Councils[p.Council]++
		data.Types[p.Type()]++
		if p.Council == "Chief Seattle" {
			d := data.Districts[p.District]
			if d == nil {
				d = make(map[string]int)
				data.Districts[p.District] = d
			}
			unitName := p.Unit()
			d[unitName]++
			d[""]++
		}
	}

	return rc.Respond(s.templates.Index, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_admin(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}
	return rc.Respond(s.templates.Admin, http.StatusOK, nil)
}

func (s *service) Serve_dashboard_refreshClasses(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}
	if !rc.IsPost() {
		return application.ErrBadRequest
	}
	classes, err := sheet.GetClasses(rc.Ctx, rc.Conference.Configuration.ClassesSheetURL)
	if err != nil {
		return err
	}
	if err := s.Store.PutClasses(rc.Ctx, classes); err != nil {
		return err
	}
	return rc.Redirect("/dashboard/classes", application.FlashInfo, "%d classes updated", len(classes))
}

func (s *service) Serve_dashboard_uploadRegistrations(rc *requestContext) error {
	if !rc.IsAdmin() {
		return application.ErrForbidden
	}
	if !rc.IsPost() {
		return application.ErrBadRequest
	}

	f, _, err := rc.Request.FormFile("file")
	if err == http.ErrMissingFile {
		return &application.HTTPError{Status: http.StatusBadRequest, Message: "File is required."}
	} else if err != nil {
		return err
	}
	defer f.Close()
	return s.importRegistrations(rc, f)
}

func (s *service) importRegistrations(rc *requestContext, data io.Reader) error {
	participants, err := dk.ParseCSV(data)
	if err != nil {
		return err
	}

	err = s.Store.PutParticipants(rc.Ctx, participants)
	if err != nil {
		return err
	}

	return rc.Redirect("/dashboard/admin", "info", "Import %d participants", len(participants))
}

func (s *service) Serve_dashboard_classes(rc *requestContext) error {
	registered := make(map[int]int)
	for _, p := range rc.Conference.Participants() {
		for _, n := range p.Classes {
			registered[n]++
		}
	}

	classes := rc.Conference.Classes()

	switch sortKey, reverse := conference.SortKeyReverse(rc.FormValue("sort")); sortKey {
	case "registered":
		sort.SliceStable(classes, reverse(func(i, j int) bool {
			return registered[classes[i].Number] < registered[classes[j].Number]
		}))
	case "available":
		sort.SliceStable(classes, reverse(func(i, j int) bool {
			m := classes[i].Capacity - registered[classes[i].Number]
			if classes[i].Capacity == 0 {
				m = 9999
			}
			n := classes[j].Capacity - registered[classes[j].Number]
			if classes[j].Capacity == 0 {
				n = 9999
			}
			return m < n
		}))
	case "default":
		conference.SortClasses(classes, rc.FormValue("sort"))
	}

	var data = struct {
		Classes    []*conference.Class
		Lunch      interface{}
		Registered interface{}
		Available  interface{}
	}{
		Classes: classes,
		Lunch:   rc.Conference.ClassLunch,
		Registered: func(c *conference.Class) string {
			n := registered[c.Number]
			if n == 0 {
				return ""
			}
			return strconv.Itoa(n)
		},
		Available: func(c *conference.Class) string {
			if c.Capacity == 0 {
				return ""
			}
			return strconv.Itoa(c.Capacity - registered[c.Number])
		},
	}
	return rc.Respond(s.templates.Classes, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_classes_(rc *requestContext) error {
	n, _ := strconv.Atoi(strings.TrimPrefix(rc.Request.URL.Path, "/dashboard/classes/"))
	class := rc.Conference.Class(n)
	if class == nil {
		return application.ErrNotFound
	}

	var data = struct {
		InstructorView    bool
		Class             *conference.Class
		Participants      []*conference.Participant
		ParticipantEmails []string
		InstructorURL     string
		Lunch             *conference.Lunch
	}{
		Class:          class,
		InstructorView: rc.IsStaff(),
		Lunch:          rc.Conference.ClassLunch(class),
	}

	if len(data.Class.AccessToken) >= 4 {
		data.InstructorURL = fmt.Sprintf("%s://%s/dashboard/classes/%d?t=%s", s.Protocol, rc.Request.Host, class.Number, data.Class.AccessToken)
		if rc.Request.FormValue("t") == data.Class.AccessToken {
			data.InstructorView = true
		}
	}

	if data.InstructorView {
		data.Participants = rc.Conference.ClassParticipants(class)
		conference.SortParticipants(data.Participants, rc.Request.FormValue("sort"))
		for _, p := range data.Participants {
			data.ParticipantEmails = append(data.ParticipantEmails, p.Emails()...)
		}
		sort.Strings(data.ParticipantEmails)
		// Deduplicate
		i := 0
		prev := ""
		for _, e := range data.ParticipantEmails {
			if e != prev {
				prev = e
				data.ParticipantEmails[i] = e
				i++
			}
		}
		data.ParticipantEmails = data.ParticipantEmails[:i]
	}

	return rc.Respond(s.templates.Class, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_participants(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}

	participants := rc.Conference.Participants()
	conference.SortParticipants(participants, rc.FormValue("sort"))

	var data = struct {
		Participants   []*conference.Participant
		SessionClasses interface{}
	}{
		participants,
		rc.Conference.ParticipantSessionClasses,
	}
	return rc.Respond(s.templates.Participants, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_participants_(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}

	id := strings.TrimPrefix(rc.Request.URL.Path, "/dashboard/participants/")
	participant := rc.Conference.Participant(id)
	if participant == nil {
		return application.ErrNotFound
	}

	var data = struct {
		Participant       *conference.Participant
		Schedule          []*conference.ScheduleItem
		InstructorClasses []int
	}{
		Participant:       participant,
		Schedule:          rc.Conference.ParticipantSchedule(participant),
		InstructorClasses: rc.Conference.ParticipantInstructorClasses(participant),
	}
	return rc.Respond(s.templates.Participant, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_setInstructorClasses(rc *requestContext) error {
	if !rc.IsAdmin() {
		return application.ErrForbidden
	}

	id := rc.FormValue("id")
	modifications := make(map[int]int)
	for i := 0; i < conference.NumSession; i++ {
		n, _ := strconv.Atoi(rc.FormValue(fmt.Sprintf("class%d", i)))
		modifications[i] = n
	}

	if err := s.Store.ModifyInstructorClasses(rc.Ctx, id, modifications); err != nil {
		return err
	}

	return rc.Redirect(fmt.Sprintf("/dashboard/participants/%s", id), application.FlashInfo, "Instructor classes updated")
}

func (s *service) Serve_dashboard_configuration(rc *requestContext) error {
	if !rc.IsAdmin() {
		return application.ErrForbidden
	}

	var data struct {
		Error  string
		Config string
	}

	if rc.IsPost() {
		data.Config = rc.FormValue("config")
		var config conference.Configuration
		d := json.NewDecoder(strings.NewReader(data.Config))
		d.DisallowUnknownFields()
		err := d.Decode(&config)
		var se *json.SyntaxError
		if errors.As(err, &se) {
			offset := int(se.Offset)
			data.Error = fmt.Sprintf("%d: %v", strings.Count(data.Config[:offset+1], "\n")+1, err)
		} else if err != nil {
			data.Error = err.Error()
		} else {
			err = s.Store.PutConfiguration(rc.Ctx, &config)
			if err != nil {
				return err
			}
			return rc.Redirect("/dashboard/admin", application.FlashInfo, "Configuration updated.")
		}
	} else {
		p, _ := json.MarshalIndent(rc.Conference.Configuration, "", "  ")
		data.Config = string(p)
	}

	return rc.Respond(s.templates.Configuration, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_lunchCount(rc *requestContext) error {
	data := struct {
		Lunch  map[*conference.Lunch]int
		Option map[string]int
		Count  map[string]int
		Total  int
	}{
		make(map[*conference.Lunch]int),
		make(map[string]int),
		make(map[string]int),
		0,
	}
	for _, p := range rc.Conference.Participants() {
		_, lunch := rc.Conference.ParticipantSessionClassesAndLunch(p)
		data.Lunch[lunch]++
		data.Option[p.LunchOption]++
		data.Count[fmt.Sprintf("%s:%s", lunch.Name, p.LunchOption)]++
		data.Total++
	}
	return rc.Respond(s.templates.LunchCount, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_lunchList(rc *requestContext) error {
	if !rc.IsAdmin() {
		return application.ErrForbidden
	}

	participants := rc.Conference.Participants()
	participants = conference.FilterParticipants(participants,
		func(p *conference.Participant) bool { return p.LunchOption != "" })
	conference.SortParticipants(participants, "")

	type lunchInfo struct {
		Participants []*conference.Participant
		Counts       map[string]int
	}

	data := make(map[*conference.Lunch]*lunchInfo)

	for _, p := range participants {
		lunch := rc.Conference.ParticipantLunch(p)
		li := data[lunch]
		if li == nil {
			li = &lunchInfo{Counts: make(map[string]int)}
			data[lunch] = li
		}
		li.Participants = append(li.Participants, p)
		li.Counts[p.LunchOption]++
	}
	return rc.Respond(s.templates.LunchList, http.StatusOK, data)
}

func (s *service) Serve_dashboard_lunchStickers(rc *requestContext) error {
	if !rc.IsAdmin() {
		return application.ErrForbidden
	}

	participants := rc.Conference.Participants()
	participants = conference.FilterParticipants(participants,
		func(p *conference.Participant) bool { return p.LunchOption != "" })
	sort.Slice(participants, func(i, j int) bool {
		a := participants[i]
		b := participants[j]
		alunch := rc.Conference.ParticipantLunch(a)
		blunch := rc.Conference.ParticipantLunch(b)
		switch {
		case alunch.Name < blunch.Name:
			return true
		case alunch.Name > blunch.Name:
			return false
		case a.LunchOption < b.LunchOption:
			return true
		case a.LunchOption > b.LunchOption:
			return false
		default:
			return conference.DefaultParticipantLess(a, b)
		}
	})

	iv := func(name string, def int) int {
		v, _ := strconv.Atoi(rc.FormValue(name))
		if v <= 0 {
			return def
		}
		return v
	}
	sv := func(name string, def string) string {
		v := rc.FormValue(name)
		if v == "" {
			return def
		}
		return v
	}

	var data = struct {
		Rows    int
		Columns int
		Top     string
		Left    string
		Width   string
		Height  string
		Gutter  string
		Font    string
		Pages   [][][]*conference.Participant
		Lunch   interface{}
	}{
		iv("rows", 7),
		iv("columns", 2),
		sv("top", "0.8in"),
		sv("left", "0in"),
		sv("width", "4.25in"),
		sv("height", "1.325in"),
		sv("gutter", "0.in"),
		sv("font", "16pt"),
		nil,
		rc.Conference.ParticipantLunch,
	}
	for len(participants) > 0 {
		var page [][]*conference.Participant
		for i := 0; i < data.Rows && len(participants) > 0; i++ {
			n := len(participants)
			if n > data.Columns {
				n = data.Columns
			}
			page = append(page, participants[:n])
			participants = participants[n:]
		}
		data.Pages = append(data.Pages, page)
	}
	return rc.Respond(s.templates.LunchStickers, http.StatusOK, &data)
}

func (s *service) Serve_dashboard_classrooms(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}

	type activity struct {
		Time *conference.ScheduleTime
		Name string
	}

	data := struct {
		Sessions  [][]*conference.SessionClass
		Locations map[string][]activity
	}{
		Sessions:  rc.Conference.Sessions(),
		Locations: make(map[string][]activity),
	}

	ignoreLunchLocation := rc.Conference.GeneralLunch().Location

	for _, l := range rc.Conference.Configuration.Lunches {
		if l.Location == ignoreLunchLocation {
			continue
		}
		t := conference.Seating1LunchTime
		if l.Seating != 1 {
			t = conference.Seating2LunchTime
		}
		data.Locations[l.Location] = append(data.Locations[l.Location],
			activity{Time: t, Name: fmt.Sprintf("%s Lunch", l.Name)})
	}

	for _, sessionClasses := range data.Sessions {
		for _, sc := range sessionClasses {
			t := conference.SessionTimes[sc.Session]
			if sc.Session == conference.LunchSession {
				if rc.Conference.ClassLunch(sc.Class).Seating == 1 {
					t = conference.Seating1ClassTime
				} else {
					t = conference.Seating2ClassTime
				}
			}
			data.Locations[sc.Location] = append(data.Locations[sc.Location],
				activity{Time: t, Name: fmt.Sprintf("%d: %s%s", sc.Number, sc.ShortTitle(), sc.IofN())})
		}
	}

	for _, activities := range data.Locations {
		sort.Slice(activities, func(i, j int) bool {
			return activities[i].Time.Start < activities[j].Time.Start
		})
	}

	return rc.Respond(s.templates.Classrooms, http.StatusOK, &data)
}

func evalUpdateString(source string, t time.Time) string {
	if source == "" || t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s @ %s", source, t.In(conference.TimeLocation).Format("1/2/2006 3:04PM"))
}

func (s *service) Serve_dashboard_evaluations_(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}

	participant := rc.Conference.Participant(strings.TrimPrefix(rc.Request.URL.Path, "/dashboard/evaluations/"))
	if participant == nil {
		return application.ErrNotFound
	}

	data := struct {
		Participant          *conference.Participant
		Sessions             [][]*conference.SessionClass
		Classes              []*conference.SessionClass
		SessionEvaluations   []*conference.SessionEvaluation
		ConferenceEvaluation *conference.ConferenceEvaluation
		EvaluationNote       *conference.EvaluationNote
		Redirect             string
	}{
		Participant: participant,
		Sessions:    rc.Conference.Sessions(),
		Classes:     rc.Conference.ParticipantSessionClasses(participant),
		Redirect:    "/dashboard/evalCodd",
	}

	if rc.FormValue("ref") == "p" {
		data.Redirect = fmt.Sprintf("/dashboard/participants/%s", participant.ID)
	}

	if !rc.IsPost() {
		set := rc.Request.Form.Set
		setRating := func(key string, rating int) {
			value := ""
			if rating != 0 {
				value = strconv.Itoa(rating)
			}
			rc.Request.Form.Set(key, value)
		}

		eval, err := s.Store.GetEvaluation(rc.Ctx, participant.ID)
		if err != nil {
			return err
		}

		data.ConferenceEvaluation = eval.Conference
		if data.ConferenceEvaluation == nil {
			data.ConferenceEvaluation = &conference.ConferenceEvaluation{}
		}
		ce := data.ConferenceEvaluation
		set("hashc", ce.Hash())
		set("updatec", evalUpdateString(ce.Source, ce.Updated))
		for name, pv := range ce.Ratings() {
			setRating(name, *pv)
		}

		data.EvaluationNote = eval.Note
		if data.EvaluationNote == nil {
			data.EvaluationNote = &conference.EvaluationNote{}
		}
		set("hashn", data.EvaluationNote.Hash())

		data.SessionEvaluations = make([]*conference.SessionEvaluation, conference.NumSession)
		for _, se := range eval.Sessions {
			if se.Session < 0 || se.Session >= conference.NumSession {
				log.Logf(rc.Ctx, log.Error, "bad class eval, participant=%s, session=%d", participant.ID, se.Session)
				continue
			}
			data.SessionEvaluations[se.Session] = se
		}

		for i, se := range data.SessionEvaluations {
			if se == nil {
				se = &conference.SessionEvaluation{Session: i}
				data.SessionEvaluations[i] = se
			}
			session := strconv.Itoa(i)
			set("hash"+session, se.Hash())
			set("update"+session, evalUpdateString(se.Source, se.Updated))
			for name, pv := range se.Ratings() {
				setRating(name+session, *pv)
			}
		}
		return rc.Respond(s.templates.Evaluation, http.StatusOK, &data)
	}

	getRating := func(name string) int {
		s := rc.FormValue(name)
		if s == "" {
			return 0
		}
		n, _ := strconv.Atoi(s)
		if n < 1 || n > 4 {
			rc.MarkInputInvalid(name)
		}
		return n
	}

	var modifiedEval conference.Evaluation
	var changes []string

	for i := 0; i < conference.NumSession; i++ {
		session := strconv.Itoa(i)
		classNumber, _ := strconv.Atoi(strings.TrimSpace(rc.FormValue("class" + session)))

		se := conference.SessionEvaluation{
			Session:     i,
			ClassNumber: classNumber,
			Comments:    rc.FormValue("comments" + session),
		}
		hasRating := false
		for name, pv := range se.Ratings() {
			*pv = getRating(name + session)
			if *pv != 0 {
				hasRating = true
			}
		}
		data.SessionEvaluations = append(data.SessionEvaluations, &se)

		if se.ClassNumber == 0 && (hasRating || se.Comments != "") {
			rc.MarkInputInvalid("class" + session)
		}

		if se.Hash() != rc.FormValue("hash"+session) {
			se.Source = "staff"
			se.Updated = time.Now()
			modifiedEval.Sessions = append(modifiedEval.Sessions, &se)
			changes = append(changes, fmt.Sprintf("session %d", se.Session+1))
		}
	}

	ce := &conference.ConferenceEvaluation{}
	data.ConferenceEvaluation = ce
	for name, pv := range ce.Ratings() {
		*pv = getRating(name)
	}
	ce.LearnTopics = rc.FormValue("learnTopics")
	ce.TeachTopics = rc.FormValue("teachTopics")
	ce.Comments = rc.FormValue("confComments")

	if ce.Hash() != rc.FormValue("hashc") {
		ce.Source = "staff"
		ce.Updated = time.Now()
		modifiedEval.Conference = ce
		changes = append(changes, "conference")
	}

	en := &conference.EvaluationNote{}
	data.EvaluationNote = en
	en.NoShow = rc.FormValue("noShow") != ""
	en.Text = rc.FormValue("note")
	if en.Hash() != rc.FormValue("hashn") {
		modifiedEval.Note = en
		changes = append(changes, "staff notes")
	}

	if rc.HasInvalidInput() {
		return rc.Respond(s.templates.Evaluation, http.StatusOK, &data)
	}

	if len(changes) == 0 {
		return rc.Redirect(data.Redirect, "info", "Updated evaluation for %s: no changes", data.Participant.Name())
	}

	if err := s.Store.SetEvaluation(rc.Ctx, participant.ID, &modifiedEval); err != nil {
		return err
	}

	return rc.Redirect(data.Redirect, "info", "Updated evaluation for %s: %s", data.Participant.Name(), strings.Join(changes, "; "))
}

func (s *service) Serve_dashboard_evalCode(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}

	loginCode := rc.FormValue("loginCode")
	participant := rc.Conference.ParticipantFromLoginCode(loginCode)
	if participant != nil {
		http.Redirect(rc.Response, rc.Request,
			fmt.Sprintf("/dashboard/evaluations/%s", participant.ID),
			http.StatusSeeOther)
		return nil
	}

	if _, ok := rc.Request.Form["loginCode"]; ok {
		rc.MarkInputInvalid("loginCode")
	}
	return rc.Respond(s.templates.EvalCode, http.StatusOK, nil)
}

/*
func (s *service) Serve_dashboard_report(rc *requestContext) error {
	if !rc.IsStaff() {
		return application.ErrForbidden
	}

	type instructorKey struct {
		participantID string
		classNumber   int
	}

	type comment struct {
		IsInstructor bool
		Text         string
	}

	type reportSession struct {
		Knowledge       ratings
		Presentation    ratings
		Usefulness      ratings
		Overall         ratings
		EvaluationCount int
		Comments        []comment
	}

	type reportClass struct {
		*conference.Class
		Registered int
		Sessions   []*reportSession
	}

	type countItem struct {
		Count int
		Text  string
	}

	var data struct {
		Classes           []*reportClass
		Experience        ratings
		Promotion         ratings
		Registration      ratings
		Checkin           ratings
		Midway            ratings
		Lunch             ratings
		Facilities        ratings
		Website           ratings
		SignageWayfinding ratings
		Comments          []string
		LearnTopics       []string
		TeachTopics       []string
		EvaluationCount   int
		Marketing         []countItem
		ScoutingYears     []countItem
		Nxx               map[int]int
	}

    conf := rc.Conference

	reportClasses := make(map[int]*reportClass)
	prevStart := -1
	data.Nxx = make(map[int]int)
	for _, c := range conf.Classes() {
		if start := c.Start(); start != prevStart {
			data.Nxx[start] = c.Number
			prevStart = start
		}
		sessions := make([]*reportSession, c.Length)
		for i := range sessions {
			sessions[i] = &reportSession{}
		}
		class := &reportClass{Class: c, Sessions: sessions}
		reportClasses[c.Number] = class
		data.Classes = append(data.Classes, class)
	}

	for _, p := range conf.Participants() {
		for _, n := range p.Classes {
			if c := reportClasses[n]; c != nil {
				c.Registered++
			}
		}

		for _, n := range p.InstructorClasses {
			instructors[instructorKey{participantID: p.ID, classNumber: n}] = true
		}
	}

	for _, e := range sessionEvaluations {
		if e.ClassNumber == 0 {
			// No class
			continue
		}
		c := reportClasses[e.ClassNumber]
		if c == nil {
			return fmt.Errorf("evaluation for participant %s in session %d has invalid class %d", e.ParticipantID, e.Session, e.ClassNumber)
		}
		i := e.Session - c.Start()
		if i < 0 || i >= len(c.Sessions) {
			return fmt.Errorf("evaluation for participant %s in class %d has invalid session %d", e.ParticipantID, e.ClassNumber, e.Session)
		}

		session := c.Sessions[i]
		isInstructor := instructors[instructorKey{participantID: e.ParticipantID, classNumber: e.ClassNumber}]

		if s := strings.TrimSpace(e.Comments); s != "" {
			session.Comments = append(session.Comments, comment{Text: s, IsInstructor: isInstructor})
		}

		set := func(value int, what string, r *ratings) error {
			if value < 0 || value > model.MaxEvalRating {
				return fmt.Errorf("evaluation for participant %s in in session %d has invalid %s: %d", e.ParticipantID, e.Session, what, value)
			}
			r[value]++
			return nil
		}

		if !isInstructor {
			if err := set(e.KnowledgeRating, "knowledge", &session.Knowledge); err != nil {
				return err
			}
			if err := set(e.PresentationRating, "presentation", &session.Presentation); err != nil {
				return err
			}
			if err := set(e.UsefulnessRating, "usefulness", &session.Usefulness); err != nil {
				return err
			}
			if err := set(e.OverallRating, "overall", &session.Overall); err != nil {
				return err
			}
			session.EvaluationCount++
		}
	}

	data.ScoutingYears = []countItem{{Text: "< 1"}, {Text: "1"}, {Text: "2"}, {Text: "3"}, {Text: "4"}, {Text: "5"}, {Text: "6 - 9"}, {Text: "10 - 19"}, {Text: ">= 20 "}}
	marketing := make(map[string]int)
	for _, p := range participants {
		for _, s := range strings.Split(p.Marketing, ";") {
			marketing[strings.TrimSpace(s)]++
		}
		if f, err := strconv.ParseFloat(p.ScoutingYears, 64); err == nil {
			switch {
			case f < 1:
				data.ScoutingYears[0].Count++
			case f < 2:
				data.ScoutingYears[1].Count++
			case f < 3:
				data.ScoutingYears[2].Count++
			case f < 4:
				data.ScoutingYears[3].Count++
			case f < 5:
				data.ScoutingYears[4].Count++
			case f < 6:
				data.ScoutingYears[5].Count++
			case f < 10:
				data.ScoutingYears[6].Count++
			case f < 20:
				data.ScoutingYears[7].Count++
			default:
				data.ScoutingYears[8].Count++
			}
		}
	}

	for t, c := range marketing {
		data.Marketing = append(data.Marketing, countItem{Count: c, Text: t})
	}
	sort.Slice(data.Marketing, func(i, j int) bool { return data.Marketing[i].Count > data.Marketing[j].Count })

	for _, e := range conferenceEvaluations {
		set := func(value int, what string, r *ratings) error {
			if value < 0 || value > model.MaxEvalRating {
				return fmt.Errorf("evaluation for participant %d has invalid %s: %d", e.ParticipantID, what, value)
			}
			r[value]++
			return nil
		}
		if err := set(e.ExperienceRating, "experience", &data.Experience); err != nil {
			return err
		}
		if err := set(e.PromotionRating, "promotion", &data.Promotion); err != nil {
			return err
		}
		if err := set(e.RegistrationRating, "registration", &data.Registration); err != nil {
			return err
		}
		if err := set(e.CheckinRating, "checkin", &data.Checkin); err != nil {
			return err
		}
		if err := set(e.MidwayRating, "midway", &data.Midway); err != nil {
			return err
		}
		if err := set(e.LunchRating, "lunch", &data.Lunch); err != nil {
			return err
		}
		if err := set(e.FacilitiesRating, "Facilities", &data.Facilities); err != nil {
			return err
		}
		if err := set(e.WebsiteRating, "website", &data.Website); err != nil {
			return err
		}
		if err := set(e.SignageWayfindingRating, "signageWayFinding", &data.SignageWayfinding); err != nil {
			return err
		}
		if s := strings.TrimSpace(e.Comments); s != "" {
			data.Comments = append(data.Comments, s)
		}
		if s := strings.TrimSpace(e.LearnTopics); s != "" && !ignoreTopics[strings.ToLower(s)] {
			data.LearnTopics = append(data.LearnTopics, s)
		}
		if s := strings.TrimSpace(e.TeachTopics); s != "" && !ignoreTopics[strings.ToLower(s)] {
			data.TeachTopics = append(data.TeachTopics, s)
		}
		data.EvaluationCount++
	}

	return rc.respond(svc.templates.Report, http.StatusOK, &data)
}
*/
