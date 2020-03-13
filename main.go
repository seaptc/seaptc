package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/seaptc/seaptc/api"
	"github.com/seaptc/seaptc/application"
	"github.com/seaptc/seaptc/catalog"
	"github.com/seaptc/seaptc/dashboard"
	"github.com/seaptc/seaptc/participant"
)

func main() {
	if os.Getenv("GAE_INSTANCE") != "" {
		log.SetFlags(0)
	}

	defaultAddr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		defaultAddr = ":" + port
	}

	devMode := os.Getenv("GAE_INSTANCE") == ""

	var (
		addr         = flag.String("addr", defaultAddr, "Listen on this address")
		projectID    = flag.String("p", "seaptc-ds", "Project id")
		assetsDir    = flag.String("d", "assets", "Direcory containing assets")
		useEmulator  = flag.Bool("e", devMode, "Use Datastore emulator")
		timeOverride = flag.Duration("t", 0, "Use current time as conference date plus this duration")
	)
	flag.Parse()
	ctx := context.Background()

	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.Dir(*assetsDir)))

	h, err := application.New(ctx,
		*projectID, *useEmulator, *assetsDir, devMode, *timeOverride,
		[]application.Service{
			dashboard.New(),
			catalog.New(),
			participant.New(),
			api.New(),
		})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on addr %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, h))
}
