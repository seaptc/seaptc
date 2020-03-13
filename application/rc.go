package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/seaptc/seaptc/conference"
	"github.com/seaptc/seaptc/log"
)

type RequestContext struct {
	Request  *http.Request
	Response http.ResponseWriter
	Ctx      context.Context
	invalid  map[string]bool

	Conference    *conference.Conference
	ConfFromCache bool
}

func (rc *RequestContext) Init(app *Application, w http.ResponseWriter, r *http.Request) error {
	rc.Request = r
	rc.Response = w
	rc.Ctx = log.ContextWithTraceID(r)

	var err error
	rc.Conference, rc.ConfFromCache, err = app.Store.GetConference(rc.Ctx, false)
	if err != nil {
		return err
	}

	err = r.ParseForm()
	return err
}

func (rc *RequestContext) ConvertError(err error) *HTTPError {
	e, ok := err.(*HTTPError)
	if ok {
		if e.Message == "" {
			e = &HTTPError{Status: e.Status, Message: http.StatusText(e.Status), Err: e.Err}
		}
	} else {
		e = &HTTPError{Status: http.StatusInternalServerError, Message: "Internal server error.", Err: err}
	}
	if e.Err != nil {
		log.Logf(rc.Ctx, log.Error, "status=%d, err=%v", e.Status, e.Err)
	}
	return e
}

func (rc *RequestContext) Logf(fmt string, args ...interface{}) {
	log.Logf(rc.Ctx, log.Info, fmt, args...)
}

func (rc *RequestContext) HandleError(err error) {
	if err == nil {
		return
	}
	log.Logf(rc.Ctx, log.Error, "execute error: %v", err)
	http.Error(rc.Response, "Internal server error.", http.StatusInternalServerError)
}

func (rc *RequestContext) Respond(t *template.Template, status int, v interface{}) error {
	var buf bytes.Buffer
	err := t.Execute(&buf, v)
	if err != nil {
		return err
	}
	rc.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	rc.Response.Header().Set("Cache-Control", "max-age=0")
	rc.Response.WriteHeader(status)
	rc.Response.Write(buf.Bytes())
	return nil
}

func (rc *RequestContext) IsPost() bool {
	return rc.Request.Method == "POST"
}

func (rc *RequestContext) FormValue(name string) string {
	return strings.TrimSpace(rc.Request.Form.Get(name))
}

func (rc *RequestContext) RFormValue(name string) (string, error) {
	if _, ok := rc.Request.Form[name]; !ok {
		return "", fmt.Errorf("form value %q not found", name)
	}
	return rc.FormValue(name), nil
}

func (rc *RequestContext) HasInvalidInput() bool {
	return len(rc.invalid) > 0
}

func (rc *RequestContext) MarkInputInvalid(name string) {
	if rc.invalid == nil {
		rc.invalid = make(map[string]bool)
	}
	rc.invalid[name] = true
}

func (rc *RequestContext) InvalidClass(name string) string {
	if rc.invalid[name] {
		// Boostrap CSS class for invalid input.
		return "is-invalid"
	}
	return ""
}

// return template.HTMLAttr(fmt.Sprintf(` class="%s" id="%s" name="%s" value="%s" `, cssClass, name, name, template.HTMLEscapeString(rc.FormValue(name))))

type FlashSeverity string

const (
	FlashError FlashSeverity = "danger"
	FlashInfo  FlashSeverity = "info"
)

func (rc *RequestContext) Redirect(path string, severity FlashSeverity, flashFormat string, flashArgs ...interface{}) error {
	rc.SetFlashMessage(severity, flashFormat, flashArgs...)
	http.Redirect(rc.Response, rc.Request, path, http.StatusSeeOther)
	return nil
}

func (rc *RequestContext) SetFlashMessage(severity FlashSeverity, format string, args ...interface{}) {
	http.SetCookie(rc.Response, &http.Cookie{
		Name:  "flash",
		Value: EncodeStringsForCookie(string(severity), fmt.Sprintf(format, args...)),
		Path:  "/",
	})
}

func (rc *RequestContext) FlashMessage() interface{} {
	http.SetCookie(rc.Response, &http.Cookie{
		Name:   "flash",
		Path:   "/",
		MaxAge: -1,
	})

	c, err := rc.Request.Cookie("flash")
	if err != nil {
		return nil
	}

	parts, err := DecodeStringsFromCookie(c.Value)
	if err != nil || len(parts) != 2 {
		return nil
	}
	return &struct {
		Kind, Message string
	}{
		parts[0],
		parts[1],
	}
}

func (rc *RequestContext) Sort(text string, key string) (template.HTML, error) {
	if key == "" {
		return "", errors.New("sort key cannot be empty string")
	}
	var isDefault bool
	if key[0] == '!' {
		isDefault = true
		key = key[1:]
	}

	qp := rc.Request.URL.Query()
	sort := qp.Get("sort")
	reverse := "-"
	if isDefault && sort == "" {
		sort = key
	} else if len(sort) > 0 && sort[0] == '-' {
		reverse = ""
		sort = sort[1:]
	}

	if sort == key {
		sort = reverse + key
	} else {
		sort = key
	}

	if isDefault && sort == key {
		qp.Del("sort")
	} else {
		qp.Set("sort", sort)
	}

	ucopy := *rc.Request.URL
	ucopy.RawQuery = qp.Encode()
	return template.HTML(`<a href="` + ucopy.RequestURI() + `">` + template.HTMLEscapeString(text) + `</a>`), nil
}
