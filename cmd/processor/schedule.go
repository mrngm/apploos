package main

import (
	//"bytes"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

type Location struct {
	Id       int
	Title    string
	Children []*Location
}

func (l *Location) String() string {
	return fmt.Sprintf("[%d] %s nChildren: %d", l.Id, l.Title, len(l.Children))
}

func SetupDays(everything VierdaagseOverview) []VierdaagseDay {
	days := make([]VierdaagseDay, len(everything.Days))
	copy(days, everything.Days)
	slices.SortFunc(days, func(a, b VierdaagseDay) int {
		return a.Date.Compare(b.Date)
	})
	return days
}

func SetupLocations(everything VierdaagseOverview) map[int]*Location {
	locations := make(map[int]*Location)
	for _, loc := range everything.Locations {
		if _, ok := locations[loc.Id]; !ok {
			locations[loc.Id] = &Location{
				Children: make([]*Location, 0),
			}
		}
		theLoc := locations[loc.Id]
		theLoc.Id = loc.Id
		theLoc.Title = loc.Title
		if loc.Parent > 0 {
			// Sub locations, fill in separate loop
			continue
		}
	}
	for _, loc := range everything.Locations {
		if loc.Parent > 0 {
			theParentLoc, ok := locations[loc.Parent]
			if !ok {
				panic("location parent should exist")
			}
			theChildLoc, ok := locations[loc.Id]
			if !ok {
				panic("location child should exist")
			}
			theChildLoc.Title = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(theChildLoc.Title, theParentLoc.Title), "- "))
			theParentLoc.Children = append(theParentLoc.Children, theChildLoc)
		}
	}

	for _, theLoc := range locations {
		if len(theLoc.Children) > 0 {
			// Sort the children based on name
			slices.SortFunc(theLoc.Children, func(a, b *Location) int {
				return strings.Compare(a.Title, b.Title)
			})
			for _, loc := range theLoc.Children {
				if len(loc.Children) > 0 {
					slog.Error("child location has more locations", "len", len(loc.Children))
				}
			}
		}
	}
	return locations
}

func SetupPrograms(everything VierdaagseOverview) map[int]VierdaagseProgram {
	programs := make(map[int]VierdaagseProgram)
	for _, prog := range everything.Programs {
		if _, ok := programs[prog.IdWithTitle.Id]; !ok {
			programs[prog.IdWithTitle.Id] = prog
		}
	}
	return programs
}

func RenderSchedule(everything VierdaagseOverview) {
	days := SetupDays(everything)
	//locs := SetupLocations(everything)
	_ = SetupLocations(everything)
	//progs := SetupPrograms(everything)
	_ = SetupPrograms(everything)
	for _, day := range days {
		//dayId := day.IdWithTitle.Id
		_ = day.IdWithTitle.Id
	}
}
