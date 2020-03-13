package conference

import "errors"

type Lunch struct {
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	Location  string `json:"location"`

	// 1: first, 2: second
	Seating int `json:"seating"`

	// If participant is taking one of these classes then
	//  pick up lunch here
	// else if participant is in one of these unit types then
	//  pick up lunch here
	// else
	//  pick up lunch at general
	//
	// Unit types are from registration: Pack, Troop, Crew, Ship
	//
	Classes   []int    `json:"classes"`
	UnitTypes []string `json:"unitTypes"`
}

type Configuration struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`

	// Google Open ID for login
	LoginClient struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	} `json:"loginClient"`

	// Planning spreadsheet
	ClassesSheetURL string `json:"classesSheetURL"`

	// First lunch is default
	Lunches []*Lunch `json:"lunches"`

	RegistrationURL string `json:"registrationURL"`

	// Use this message to announce when registration will open or that the
	// current catalog is for the previous event.
	CatalogStatusMessage string `json:"catalogStatusMessage"`

	StaffIDs  []string `json:"staffIDs"`
	AdminIDs  []string `json:"adminIDs"`
	CookieKey string   `json:"cookieKey"` // HMAC key for signed cookies

	// URL of Doubleknot Export page
	DoubleknotExportPageURL string `json:"doubleknotExportPageURL"`
}

func newConfiguration() *Configuration {
	return &Configuration{
		StaffIDs: []string{},
		AdminIDs: []string{},
		Lunches:  []*Lunch{tbdLunch},
	}
}

func (config *Configuration) Validate() error {
	if config.CookieKey == "" {
		return errors.New("config: CookieKey not set")
	}
	return nil
}
