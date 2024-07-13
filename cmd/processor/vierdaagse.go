package main

import (
	"strings"
	"time"
)

const (
	ROLLOVER_HOUR_FROM_START_OF_DAY = 7 // h/t @yorickvP
)

// type DQI stands for DataQualityIssue
type DQI int

const (
	DQINone DQI = 1 << iota
	DQIEndTimeBeforeStart
	DQISummaryEmptyish
	DQIDescriptionEmptyish
	DQINeededSummaryDescriptionSwap
	DQISummaryFromDescription
	DQIOnlySummary
)

// DQIToString formats the encapsulated quality issues into a slug-ish string, separated with spaces
func DQIToString(issues DQI) string {
	ret := make([]string, 0, 16)
	if issues&DQIEndTimeBeforeStart != 0 {
		ret = append(ret, "end-time-before-start")
	}
	if issues&DQISummaryEmptyish != 0 {
		ret = append(ret, "summary-emptyish")
	}
	if issues&DQIDescriptionEmptyish != 0 {
		ret = append(ret, "description-emptyish")
	}
	if issues&DQINeededSummaryDescriptionSwap != 0 {
		ret = append(ret, "swapped-summary-description")
	}
	if issues&DQISummaryFromDescription != 0 {
		ret = append(ret, "created-summary-from-description")
	}
	if issues&DQIOnlySummary != 0 {
		ret = append(ret, "only-summary")
	}
	return strings.Join(ret, ", ")
}

type VierdaagseOverview struct {
	General       VierdaagseGeneral
	Days          []VierdaagseDay
	Locations     []VierdaagseLocation
	Genres        []VierdaagseGenre
	Themes        []VierdaagseTheme
	Programs      []VierdaagseProgram
	PartnerTypes  []IdWithTitle
	Partners      []VierdaagsePartner
	FAQ           []VierdaagseFAQ
	FAQCategories []VierdaagseFAQCategory
	Updates       []VierdaagseUpdate
	//	Content       VierdaagseContent // TODO
	POI           []VierdaagsePOI
	POICategories []VierdaagsePOICategory
	FoodThemes    []VierdaagseFoodTheme
	FoodKitchen   []VierdaagseFoodKitchen
	Food          []VierdaagseFood
	// Onboarding    VierdaagseOnboarding // TODO
	// Ads           VierdaagseAds // TODO
	// Coupons       VierdaagseCoupons // TODO

	DirModTime  time.Time
	FileModTime time.Time
}

type VierdaagseGeneral struct {
	Socials []URLAndType
}

type VierdaagseDay struct {
	IdWithTitle
	Date time.Time
}

type VierdaagseLocation struct {
	IdWithTitle
	DescriptionShort string `json:"description_short"`
	Description      string
	Marker           LatLong
	Logo             string
	Images           []Image
	URL              string
	Slug             string
	Type             string
	MapboxImage      string
	// CustomData
	Parent       int
	HasProgramOn []string
	PostDate     time.Time
	DateUpdated  time.Time
}

type VierdaagseGenre struct {
	IdWithTitle
	DescriptionShort string `json:"description_short"`
	URL              string

	DatePostedAndUpdated
}

type VierdaagseTheme struct {
	IdWithTitle
	DescriptionShort string `json:"description_short"`
	Description      string
	Logo             string
	Images           []Image
	URL              string
	Slug             string
	MediaPartnerLogo Image
	MediaPartnerURL  string
	Color            string

	DatePostedAndUpdated
}

type VierdaagseProgram struct {
	IdWithTitle
	ActId         int `json:"act_id"`
	Day           DayWithId
	DayPart       string `json:"day_part"`
	SortDate      string
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	Location      SingularId
	Genres        []SingularId
	Theme         SingularId
	IsHighlight   bool `json:"is_highlight"`
	OriginCountry bool
	// Sort
	// CustomData
	// AgeWarnings
	Website          string
	DescriptionShort string `json:"description_short"`
	Description      string
	Images           []Image
	// Videolink
	TicketsPrice   float64 `json:"tickets_price"`
	TicketsLink    string  `json:"tickets_link"`
	TicketsSoldOut bool    `json:"tickets_soldout"`
	URL            string
	Socials        []URLAndType
	Related        []string // []int disguised as []string
	// Partners
	Slug       string
	ShareTitle string
	ShareText  string
	// SearchWords

	DatePostedAndUpdated

	FullStartTime      time.Time
	FullEndTime        time.Time
	CalculatedDuration time.Duration
	DataQualityIssues  DQI
}

type VierdaagsePartner struct {
	IdWithTitle
	Logo        string
	URL         struct{ LinkedURL string }
	PartnerType int
}

type VierdaagseFAQ struct {
	Id          int
	Question    string
	Answer      string
	FAQCategory int
	ShowOnHome  bool
}

type VierdaagseFAQCategory struct {
	IdWithTitle
	// Ads
}

type VierdaagseUpdate struct {
	IdWithTitle
	Category string
	Image    Image
	// Themes
	// Partners
	// Content
	Slug string
	URL  string
}

type VierdaagseContent struct {
	// TODO
}

type VierdaagsePOI struct {
	IdWithTitle
	Marker   LatLong
	Category SingularId
	Logo     string
}

type VierdaagsePOICategory struct {
	IdWithTitle
	EnabledByDefault bool
	Logo             string
}

type VierdaagseFoodTheme struct {
	IdWithTitle
	DatePostedAndUpdated
}

type VierdaagseFoodKitchen struct {
	IdWithTitle
	DatePostedAndUpdated
}

type VierdaagseFood struct {
	IdWithTitle
	// OpeningTimes
	// OpeningHours
	Location         SingularId
	Marker           LatLong
	DescriptionShort string `json:"description_short"`
	Description      string
	Images           []Image
	// Videolink
	Theme       []SingularId
	Kitchen     []SingularId
	IsHighlight bool `json:"is_highlight"`
	// Partners
	Related []int
	// Food
	// CustomData
	URL  string
	Slug string

	DatePostedAndUpdated
}

type VierdaagseOnboarding struct {
	// TODO
}

type VierdaagseAds struct {
	// TODO
}

type VierdaagseCoupons struct {
	// TODO
}

type URLAndType struct {
	Type string
	URL  string
}

type LatLong struct {
	Lat float64
	Lng float64
}

type Image struct {
	URL         string
	Width       int
	Height      int
	Orientation string
	// FocalPoint
}

type DayWithId struct {
	Id   int
	Date time.Time
}

type SingularId struct {
	Id int
}

type DatePostedAndUpdated struct {
	PostDate    time.Time
	DateUpdated time.Time
}

type IdWithTitle struct {
	Id    int
	Title string
}
