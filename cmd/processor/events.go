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
	Alias         string // optional rename, if set, use this name
	Slug          string
	HasParent     bool
	ProgramsByDay map[int][]*Program
	Children      []*Location

	TotalNrProgramsByDay map[int]int
	DataQualityIssues    DQI `json:",omitifempty"`
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
	Genres         []string

	FullStartTime      time.Time
	FullEndTime        time.Time
	CalculatedDuration time.Duration
	DataQualityIssues  DQI  `json:",omitifempty"`
	RolloverImplied    bool // When we already set the correct start and end time (true), do not correct for ROLLOVER_HOUR_FROM_START_OF_DAY. Nicely defaults to false with JSON Unmarshal.
	StartTimeEstimated bool // When the program was published, but did not have a StartTime yet, and we derived it from SortDate
	EndTimeEstimated   bool // When the program was published, but did not have a EndTime yet, and guessed the duration
}

func (p Program) String() string {
	dqi := " "
	dqiExpanded := ""
	if p.DataQualityIssues != 0 {
		dqi = " (!) "
		dqiExpanded = " (" + DQIToString(p.DataQualityIssues) + ")"
	}
	return fmt.Sprintf("[%v]%s%v - %v: %s%s",
		p.FullStartTime.Format("2006-01-02"),
		dqi,
		p.FullStartTime.Format("15:04"),
		p.FullEndTime.Format("15:04"),
		p.Title,
		dqiExpanded,
	)
}

// EventData provides an easy format for importing events from other locations.
//
// XXX: Program uses DayId and LocationId internally; it's not reasonable to think that external events use the same identifiers.
// Keep those fields out of the feed if processor needs to fill them in automatically
type EventData struct {
	LocationId       int // see enricher.go
	LocationTitle    string
	LocationSlug     string
	LocationOverride bool
	Programs         []*Program
}
