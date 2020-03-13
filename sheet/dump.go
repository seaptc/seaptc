// +build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/seaptc/seaptc/sheet"
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	ctx := context.Background()
	classes, err := sheet.GetClasses(ctx, flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	p, err := json.MarshalIndent(classes, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(p)
}
