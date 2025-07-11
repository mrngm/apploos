package main

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type Schedule struct {
	Days      []*Day
	Locations []*Location
}

type Day struct {
	Id    int
	Title string
	Date  time.Time
}

type Location struct {
	Id            int
	Title         string
	Slug          string
	HasParent     bool
	ProgramsByDay map[int][]*Program
	Children      []*Location

	TotalNrProgramsByDay map[int]int
}

func (l *Location) String() string {
	return fmt.Sprintf("[%d] %s nChildren: %d", l.Id, l.Title, len(l.Children))
}

type Program struct {
	Id             int
	DayId          int
	LocationId     int
	Title          string
	Slug           string
	Summary        string
	Details        string
	TicketLink     string
	TicketPrice    decimal.Decimal
	TicketsSoldOut bool

	FullStartTime      time.Time
	FullEndTime        time.Time
	CalculatedDuration time.Duration
	DataQualityIssues  DQI  `json:",omitifempty"`
	RolloverImplied    bool // When we already set the correct start and end time (true), do not correct for ROLLOVER_HOUR_FROM_START_OF_DAY. Nicely defaults to false with JSON Unmarshal.
	StartTimeEstimated bool // When the program was published, but did not have a StartTime yet, and we derived it from SortDate
	EndTimeEstimated   bool // When the program was published, but did not have a EndTime yet, and guessed the duration
}

// EventData provides an easy format for importing events from other locations.
//
// XXX: Program uses DayId and LocationId internally; it's not reasonable to think that external events use the same identifiers.
// Keep those fields out of the feed if processor needs to fill them in automatically
type EventData struct {
	LocationId    int // see enricher.go
	LocationTitle string
	LocationSlug  string
	Programs      []*Program
}
