package application

import (
	"crypto/md5"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

func (s *Application) initFuncMap(assetsDir string) {
	var fileHashes sync.Map
	s.templateFuncs = template.FuncMap{
		"args": func(values ...interface{}) []interface{} {
			return values
		},
		// add adds integers.
		"add": func(values ...int) int {
			result := 0
			for _, v := range values {
				result += v
			}
			return result
		},
		// truncate truncates s to n runes.
		"truncate": func(s string, n int) string {
			i := 0
			for j := range s {
				i++
				if i > n {
					return s[:j] + "..."
				}
			}
			return s
		},
		// staticFile returns URL of static file w/ cache busting hash.
		"staticFile": func(s string) (string, error) {
			if u, ok := fileHashes.Load(s); ok {
				return u.(string), nil
			}
			p := filepath.Join(assetsDir, "static", s)
			f, err := os.Open(p)
			if err != nil {
				return "", err
			}
			defer f.Close()
			h := md5.New()
			io.Copy(h, f)
			u := fmt.Sprintf("%s?%x", path.Join("/static", s), h.Sum(nil))
			fileHashes.Store(s, u)
			return u, nil
		},
		"noHTMLEscape": func(s string) template.HTML {
			return template.HTML(s)
		},
		"separator": func(s string) *templateSeparator {
			return &templateSeparator{i: -1, s: s}
		},
		"join": func(slice interface{}, sep string) (string, error) {
			parts, ok := slice.([]string)
			if !ok {
				v := reflect.ValueOf(slice)
				if v.Kind() != reflect.Slice {
					return "", errors.New("join: first argument is not a slice")
				}
				for i := 0; i < v.Len(); i++ {
					parts = append(parts, fmt.Sprint(v.Index(i).Interface()))
				}
			}
			return strings.Join(parts, sep), nil
		},
	}
}

type templateSeparator struct {
	i int
	s string
}

func (ts *templateSeparator) String() string {
	ts.i++
	if ts.i == 0 {
		return ""
	}
	return ts.s
}

func (app *Application) parseTemplates(templates interface{}, subdir string) error {
	val := reflect.ValueOf(templates)

	const badArgMessage = "argument to TemplatesFromField must be a pointer to a struct"
	if val.Kind() != reflect.Ptr {
		panic(badArgMessage)
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		panic(badArgMessage)
	}

	templateType := reflect.TypeOf((**template.Template)(nil)).Elem()

	tbase := template.New("").Funcs(app.templateFuncs)
	cache := make(map[string]*template.Template)

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i)
		if ft.Type != templateType {
			continue
		}
		sources := ft.Tag.Get("template")
		if sources == "" {
			continue
		}
		if ft.PkgPath != "" {
			panic("fields with template tag must be exported")
		}

		var files []string
		for _, name := range strings.Split(sources, ",") {
			if name == "." {
				r, size := utf8.DecodeRuneInString(ft.Name)
				name = string(unicode.ToLower(r)) + ft.Name[size:]
			}
			if path.Ext(name) == "" {
				name += ".html"
			}
			files = append(files, filepath.Join(app.AssetsDir, "templates", subdir, name))
		}

		t := tbase
		for i := len(files) - 1; i >= 0; i-- {
			key := strings.Join(files[i:], "\n")
			tt, ok := cache[key]
			if !ok {
				tt = template.Must(t.Clone())
				var err error
				tt, err = tt.ParseFiles(files[i])
				if err != nil {
					return fmt.Errorf("%s: %w", files[i], err)
				}
				cache[key] = tt
			}
			t = tt
		}
		t = t.Lookup("ROOT")
		if t == nil {
			return fmt.Errorf("could not find template ROOT in %v", files)
		}
		val.Field(i).Set(reflect.ValueOf(t))
	}
	return nil
}
