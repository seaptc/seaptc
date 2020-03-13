package main

import (
	"context"
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
	projectID := flag.String("p", "seaptc-ds", "Project for Datastore")
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
		help: "Read configuration from datastore and print.",
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
}
