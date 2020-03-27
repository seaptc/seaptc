package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/seaptc/seaptc/conference"
	"github.com/seaptc/seaptc/store"
)

func main() {
	ctx := context.Background()
	log.SetFlags(0)
	projectID := flag.String("p", store.DefaultProjectID(), "Project for Datastore")
	useEmulator := flag.Bool("e", true, "Use Datastore emulator")
	flag.Parse()
	s, err := store.New(ctx, *projectID, *useEmulator)
	if err != nil {
		log.Fatal(err)
	}

	if flag.Arg(0) == "help" {
		help()
		return
	}

	c := commands[flag.Arg(0)]
	if c == nil {
		var names []string
		for name := range commands {
			names = append(names, name)
		}
		sort.Strings(names)
		log.Fatalf("Unknown command %q, supported commands are %s", flag.Arg(0), strings.Join(names, ", "))
	}

	if err := c.fn(ctx, s); err != nil {
		log.Fatal(err)
	}
}

func help() {
	var names []string
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Println("help: print this help")
	for _, name := range names {
		fmt.Printf("%s: %s\n", name, commands[name].help)
	}
}

type command struct {
	fn   func(context.Context, *store.Store) error
	help string
}

var commands = map[string]*command{
	"delete": {
		help: "Delete named blob from store",
		fn: func(ctx context.Context, s *store.Store) error {
			return s.DeleteBlob(ctx, flag.Arg(1))
		}},
	"config-get": {
		help: "Read configuration from datastore and print as JSON.",
		fn: func(ctx context.Context, s *store.Store) error {
			conf, _, err := s.GetConference(ctx, false)
			if err != nil {
				return err
			}
			p, _ := json.MarshalIndent(conf.Configuration, "", "  ")
			os.Stdout.Write(append(p, '\n'))
			return nil
		}},
	"config-put": {
		help: "Read configuration from stdin as JSON and save to datastore.",
		fn: func(ctx context.Context, s *store.Store) error {
			var config conference.Configuration
			if err := json.NewDecoder(os.Stdin).Decode(&config); err != nil {
				return err
			}
			return s.PutConfiguration(ctx, &config)
		}},
	"eval-codes": {
		help: "Print dashboard and evaluation codes for planning speadsheet",
		fn:   evalCodes,
	},
	"classes-print": {
		help: "Print class listing as text.",
		fn: func(ctx context.Context, s *store.Store) error {
			conf, _, err := s.GetConference(ctx, false)
			if err != nil {
				return err
			}
			for _, c := range conf.Classes() {
				fmt.Printf("%d: %s\n", c.Number, c.Title)
			}
			return nil
		}},
}

func evalCodes(ctx context.Context, s *store.Store) error {
	conf, _, err := s.GetConference(ctx, false)
	if err != nil {
		return err
	}
	classes := conf.Classes()

	// Collect codes in use.
	evaluationCodes := make(map[string]int)
	accessTokens := make(map[string]int)
	for _, class := range classes {
		if class.AccessToken != "" {
			num, ok := accessTokens[class.AccessToken]
			if ok {
				return fmt.Errorf("Access token %s used in class %d and %d", class.AccessToken, num, class.Number)
			}
		}
		accessTokens[class.AccessToken] = class.Number

		for _, code := range class.EvaluationCodes {
			num, ok := evaluationCodes[code]
			if ok {
				return fmt.Errorf("Code %s used in class %d and %d", code, num, class.Number)
			}
			evaluationCodes[code] = class.Number
		}
	}

	// Don't assign these codes
	evaluationCodes["0000"] = 0
	evaluationCodes["1234"] = 0

	for _, class := range classes {
		if class.AccessToken == "" {
			for i := 0; i < 1000; i++ {
				r, err := randUint32()
				if err != nil {
					return err
				}
				token := fmt.Sprintf("%08x", r)
				if strings.HasPrefix(token, "0") {
					continue
				}
				_, ok := accessTokens[token]
				if !ok {
					accessTokens[token] = class.Number
					class.AccessToken = token
					break
				}
			}
		}
		codes := class.EvaluationCodes
		if len(codes) > class.Length() {
			// Remove extra codes.
			codes = codes[:class.Length()]
		} else {
			// Add codes to class as needed.
			for i := len(codes); i < class.Length(); i++ {
				for j := 0; j < 1000; j++ {
					r, err := randUint32()
					if err != nil {
						return err
					}
					code := fmt.Sprintf("%04d", r%10000)
					if strings.HasPrefix(code, "0") {
						continue
					}
					_, ok := evaluationCodes[code]
					if !ok {
						evaluationCodes[code] = class.Number
						codes = append(codes, code)
						break
					}
				}
			}
		}
		class.EvaluationCodes = codes
	}
	conference.SortClasses(classes, "")

	// Quote tokens and codes in output to prevent spreadsheet from
	// interpreting the values as numbers.
	fmt.Printf("\"class\",\"accessToken\",\"evaluationCodes\"\n")
	for _, class := range classes {
		fmt.Printf("\"%d\",\"=\"\"%s\"\"\",\"=\"\"%s\"\"\"\n", class.Number, class.AccessToken, strings.Join(class.EvaluationCodes, ", "))
	}
	return nil
}

// randUint32 returns a randum uint32
func randUint32() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}
