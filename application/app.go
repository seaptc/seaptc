package application

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/seaptc/seaptc/store"
)

type Service interface {
	Setup(app *Application) (name string, templates interface{}, err error)
	Execute(fn interface{}, w http.ResponseWriter, r *http.Request)
}

type Debug struct {
	LoginDate time.Time
}

type Application struct {
	Store     *store.Store
	Protocol  string
	AssetsDir string

	TimeOverride time.Duration // for debugging

	templateFuncs template.FuncMap
}

func New(ctx context.Context, projectID string, useEmulator bool, assetsDir string, devMode bool, timeOverride time.Duration,
	services []Service) (http.Handler, error) {
	app := &Application{
		Protocol:     "https",
		AssetsDir:    assetsDir,
		TimeOverride: timeOverride,
	}
	if devMode {
		app.Protocol = "http"
	}

	var err error
	app.Store, err = store.New(ctx, projectID, useEmulator)
	if err != nil {
		return nil, err
	}

	conf, _, err := app.Store.GetConference(ctx, false)
	if err != nil {
		return nil, err
	}

	if err := conf.Configuration.Validate(); err != nil {
		return nil, err
	}

	app.initFuncMap(assetsDir)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.Dir(assetsDir)))

	for _, s := range services {
		name, templates, err := s.Setup(app)
		if err != nil {
			return nil, err
		}

		if templates != nil {
			if err := app.parseTemplates(templates, name); err != nil {
				return nil, err
			}
		}

		if err := app.addHandlers(s, mux); err != nil {
			return nil, err
		}
	}
	return mux, nil
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (app *Application) addHandlers(service Service, mux *http.ServeMux) error {
	v := reflect.ValueOf(service)
	t := v.Type()
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if !strings.HasPrefix(m.Name, "Serve_") {
			continue
		}
		if m.Type.NumIn() != 2 || m.Type.NumOut() != 1 || m.Type.Out(0) != errorType {
			return fmt.Errorf("application: %T.Serve_%s should take one argument and return error", service, m.Name)
		}

		path := strings.ReplaceAll(strings.TrimPrefix(m.Name, "Serve"), "_", "/")
		mux.Handle(path, &handler{
			service: service,
			fn:      m.Func.Interface(),
		})
	}

	return nil
}

type handler struct {
	service Service
	fn      interface{}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.service.Execute(h.fn, w, r)
}
