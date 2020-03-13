// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/seaptc/seaptc/dk"
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	participants, err := dk.ParseCSV(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	p, _ := json.MarshalIndent(participants, "", "  ")
	os.Stdout.Write(p)
}
