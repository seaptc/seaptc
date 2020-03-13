package conference

import (
	"regexp"
	"sort"
	"strings"
)

// Class represents a PTC class. All data is loaded from the spreadsheet.
type Class struct {
	Number           int      `json:"number"`
	Start            int      `json:"start"`
	End              int      `json:"end"`
	Responsibility   string   `json:"responsibility"`
	New              string   `json:"new"`
	Title            string   `json:"title"`
	TitleNote        string   `json:"titleNote"`
	Description      string   `json:"description"`
	Programs         int      `json:"programs"`
	Capacity         int      `json:"capacity"`
	Location         string   `json:"location"`
	AccessToken      string   `json:"accessToken"`
	InstructorNames  []string `json:"instructorNames"`
	InstructorEmails []string `json:"instructorEmails"`
	EvaluationCodes  []string `json:"evaluationCodes"`
}

// Length returns length of class in sessions.
func (c *Class) Length() int { return c.End - c.Start + 1 }

func (c *Class) ProgramDescriptions(reverse bool) []*ProgramDescription {
	return programDescriptionsForMask(c.Programs, reverse)
}

func (c *Class) ShortTitle() string {
	if i := strings.Index(c.Title, " - "); i > 0 {
		return c.Title[:i]
	}
	if strings.HasSuffix(c.Title, ")") {
		if i := strings.Index(c.Title, " ("); i > 0 {
			return c.Title[:i]
		}
	}
	return c.Title
}

var ClassNumberPat = regexp.MustCompile(`[1-6]\d\d`)

func IsValidClassNumber(number int) bool {
	return 100 <= number && number < (NumSession+1)*100
}

func SortClasses(classes []*Class, key string) {
	key, reverse := SortKeyReverse(key)
	switch key {
	case "location":
		sort.Slice(classes, reverse(func(i, j int) bool {
			switch {
			case classes[i].Location < classes[j].Location:
				return true
			case classes[i].Location > classes[j].Location:
				return false
			default:
				return classes[i].Number < classes[j].Number
			}
		}))
	case "responsibility":
		sort.Slice(classes, reverse(func(i, j int) bool {
			switch {
			case classes[i].Responsibility < classes[j].Responsibility:
				return true
			case classes[i].Responsibility > classes[j].Responsibility:
				return false
			default:
				return classes[i].Number < classes[j].Number
			}
		}))
	case "capacity":
		sort.Slice(classes, reverse(func(i, j int) bool {
			switch {
			case classes[i].Capacity < classes[j].Capacity:
				return true
			case classes[i].Capacity > classes[j].Capacity:
				return false
			default:
				return classes[i].Number < classes[j].Number
			}
		}))
	default:
		sort.Slice(classes, reverse(func(i, j int) bool { return classes[i].Number < classes[j].Number }))
	}
}

const (
	CubScoutProgram = iota
	ScoutsBSAProgram
	VenturingProgram
	SeaScoutProgram
	CommissionerProgram
	YouthProgram
	NumPrograms
)

type ProgramDescription struct {
	Code string
	Name string
}

func (pd *ProgramDescription) TitleName() string {
	return strings.Title(pd.Name)
}

var ProgramDescriptions = []*ProgramDescription{
	// Must match order in xxxPorgram constants above.
	{"cub", "Cub Pack adults"},
	{"bsa", "Scout Troop adults"},
	{"ven", "Venturing Crew adults"},
	{"sea", "Sea Scout adults"},
	{"com", "Commissioners"},
	{"you", "youth"},

	// AllProgram must be last in slice for programDescriptionsForMask()
	{"all", "everyone"},
}

func programDescriptionsForMask(mask int, reverse bool) []*ProgramDescription {
	if (1<<NumPrograms)-1 == mask {
		return ProgramDescriptions[NumPrograms:]
	}

	var result []*ProgramDescription
	for i := 0; i < NumPrograms; i++ {
		if ((1 << uint(i)) & mask) != 0 {
			result = append(result, ProgramDescriptions[i])
		}
	}

	if reverse {
		i := 0
		j := len(result) - 1
		for i < j {
			result[i], result[j] = result[j], result[i]
			i++
			j--
		}
	}

	return result
}
