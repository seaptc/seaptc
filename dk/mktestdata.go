// +build ignore

package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/seaptc/server/model"
	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(0)
	store.SetupFlags()
	n := flag.Int("n", 800, "number of participants")
	flag.Parse()
	ctx := context.Background()

	st, err := store.NewFromFlags(ctx)
	if err != nil {
		log.Fatal(err)
	}
	config, err := st.GetAppConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	classes, err := sheet.GetClasses(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}

	names, err = readDictFile("/usr/share/dict/propernames")
	if err != nil {
		log.Fatal(err)
	}

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	record := make([]string, len(columns))
	for i, c := range columns {
		record[i] = c.name
	}
	w.Write(record)
	for i := 0; i < *n; i++ {
		for j, c := range columns {
			record[j] = c.value()
		}
		w.Write(record)
		for _, c := range selectClasses(classes) {
			record[0] = fmt.Sprintf("%d: %s", c.Number, c.Title)
			w.Write(record)
		}
		if rand.Intn(20) == 0 {
			record[0] = fmt.Sprintf("700: Banquet")
			w.Write(record)
		}
	}
}

var columns = []struct {
	name  string
	value func() string
}{
	{"Event Name", stringValue("2019 Program and Training Conference")},
	{"Registration Number", registrationNumber()},
	{"Registered By First Name", stringValue("John")},
	{"Registered By Last Name", stringValue("Smith")},
	{"Registered By Email", stringValue("john@example.com")},
	{"Registered By Phone", stringValue("425-882-8080")},
	{"Registration Date/Time", stringValue("6/4/2019 1:19:54 PM")},
	{"First Name", name},
	{"Last Name", name},
	{"Suffix", stringValue("")},
	{"Type", weightedStringPicker("0 Adult\n 3 Instructors & Staff\n 2 Youth")},
	{"Telephone", stringValue("425-882-8080")},
	{"Email", email},
	{"Address", stringValue("1 Main Street")},
	{"City", stringValue("Redmond")},
	{"State", stringValue("WA")},
	{"Postal Code", stringValue("98052")},
	{"Generic 1", stringValue("")}, // BSA number
	{"Council", council},
	{"District", pickDistrict},
	{"Unit Type", unitType},
	{"Unit Number", unitNumber},
	{"Do you have any meal requirements?:Gluten Free", pickAorB(10, "", 1, "Gluent Free")},
	{"Do you have any meal requirements?:Vegan", pickAorB(10, "", 1, "Vegan")},
	{"Do you have any meal requirements?:Vegetarian", pickAorB(10, "", 1, "Vegetarian")},
	{"Nickname for PTC name badge", pickAorB(1, "Nicky", 10, "")},
	{"Print QR code on PTC name badge?", pickAorB(1, "No", 10, "Yes")},
	{"How did you hear about the PTC?:Roundtable/District", pickAorB(10, "", 1, "Roundtable/District")},
	{"How did you hear about the PTC?:eTotem", pickAorB(10, "", 1, "eTotem")},
	{"How did you hear about the PTC?:Council website", pickAorB(10, "", 1, "Council website")},
	{"How did you hear about the PTC?:Attended before", pickAorB(10, "", 1, "Attended before")},
	{"How did you hear about the PTC?:Unit", pickAorB(10, "", 1, "Unit")},
	{"How did you hear about the PTC?:Wood Badge", pickAorB(10, "", 1, "Wood Badge")},
	{"What other ways did you hear about the PTC?", stringValue("")},
	{"How many years have you been in scouting?", stringValue("")},
	{"Staff role", staffRole},
	{"Which classes are you teaching?", stringValue("")},
	{"Which organization are you representing on the midway?", stringValue("")},
}

var council = weightedStringPicker(`
    100 Chief Seattle
    5 Mount Baker
    5 Other`)

var pickDistrict = stringPicker(`
    Don't know or not in Chief Seattle Council
    Council
    Alpine (Cougar Mountain, Fall City, Issaquah, North Bend, Sammamish Plateau, Snoqualmie, Renton Highlands)
    Aquila (Burien, Des Moines, Normandy Park, Sea Tac, Tukwila, Vashon Island, West Seattle, White Center)
    Aurora (Lake Forest Park, North Seattle, Shoreline)
    Foothills (Auburn, Black Diamond, Covington, Maple Valley, Pacific)
    Green River (Kent, Newcastle, Renton, Skyway)
    Mt. Olympus (Clallam and Jefferson counties)
    New District North (Bothell, Carnation, Duvall, Kenmore, Redmond, Woodinville)
    New District South (Bellevue, Kirkland, Mercer Island, North Renton)
    Orca (Central Kitsap, North Kitsap and Bainbridge Island)
    Sinclair (Belfair, Bremerton, Port Orchard and surrounding cities) (Beacon Hill, Capitol Hill, Central Seattle, South Seattle, Rainier Valley)
    Thunderbird (Beacon Hill, Capitol Hill, Central Seattle, South Seattle, Rainier Valley)
    `)

var unitType = weightedStringPicker(`
    10 Cub Pack
    10 Scout Troop
    10 Venturing Crew
    10 Sea Scout Ship
    3 District
    3 Council
    3 Other
    `)

var staffRole = stringPicker(`
    Instructor
    Support (working for West Niver)
    Midway (working for Nick Heaton)
    `)

func stringValue(s string) func() string {
	return func() string { return s }
}

func pickAorB(wa int, a string, wb int, b string) func() string {
	return func() string {
		i := rand.Intn(wa + wb)
		if i < wa {
			return a
		}
		return b
	}
}

var names []string

func name() string {
	return names[rand.Intn(len(names))]
}

func email() string {
	return name() + "@example.com"
}

var weightedPat = regexp.MustCompile(`(?m)^\s*(\d+)\s+(.*)$`)

func weightedStringPicker(choices string) func() string {
	type value struct {
		w int
		s string
	}

	var n int
	var values []value
	for _, m := range weightedPat.FindAllStringSubmatch(choices, -1) {
		w, _ := strconv.Atoi(m[1])
		n += 1
		values = append(values, value{w: w, s: m[2]})
	}

	return func() string {
		i := rand.Intn(n)
		for _, v := range values {
			i -= v.w
			if i < 0 {
				return v.s
			}
		}
		panic("ouch")
	}
}

func stringPicker(choices string) func() string {
	var values []string
	for _, s := range strings.Split(choices, "\n") {
		s = strings.TrimSpace(s)
		if s != "" {
			values = append(values, s)
		}
	}
	return func() string {
		return values[rand.Intn(len(values))]
	}
}

func unitNumber() string {
	return strconv.Itoa(100 + rand.Intn(20))
}

func registrationNumber() func() string {
	var registrationNumber = 1000
	return func() string {
		registrationNumber += 127
		return strconv.Itoa(registrationNumber)
	}
}

func readDictFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	var values []string
	for s.Scan() {
		values = append(values, s.Text())
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func selectClasses(classes []*model.Class) []*model.Class {
	var result []*model.Class

nextClass:
	for i := 0; i < 50; i++ {
		proposed := classes[rand.Intn(len(classes))]
		ps, pe := proposed.StartEnd()
		for _, selected := range result {
			ss, se := selected.StartEnd()
			if pe >= ss && ps <= se {
				continue nextClass
			}
		}
		result = append(result, proposed)
	}
	return result
}
