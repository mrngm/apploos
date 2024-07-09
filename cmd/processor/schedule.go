package main

import (
	//"bytes"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"
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

func appendEventTime(initialTime time.Time, eventTime string) time.Time {
	hours, minutes, ok := strings.Cut(eventTime, ":")
	if ok && len(hours) == 2 && len(minutes) == 2 {
		hrs, err := strconv.Atoi(hours)
		if err == nil {
			initialTime = initialTime.Add(time.Duration(hrs) * time.Hour)
		}
		mins, err := strconv.Atoi(minutes)
		if err == nil {
			initialTime = initialTime.Add(time.Duration(mins) * time.Minute)
		}
	}
	return initialTime
}

func SetupPrograms(everything VierdaagseOverview) (map[int]*VierdaagseProgram, map[int][]*VierdaagseProgram) {
	dayToPrograms := make(map[ /* dayId */ int][] /* sorted slice based on start_time full details */ *VierdaagseProgram)
	programs := make(map[int]*VierdaagseProgram)
	for _, prog := range everything.Programs {
		prog := prog
		// Calculate full start time and full end time. The start time is on the scheduled day. The end time might be on
		// the next day.
		prog.FullStartTime = appendEventTime(prog.Day.Date, prog.StartTime)
		prog.FullEndTime = appendEventTime(prog.Day.Date, prog.EndTime)
		if prog.FullStartTime.After(prog.FullEndTime) {
			// EndTime should be after StartTime
			prog.FullEndTime = prog.FullEndTime.AddDate(0, 0, 1)
		}
		prog.CalculatedDuration = prog.FullEndTime.Sub(prog.FullStartTime)

		if _, ok := programs[prog.IdWithTitle.Id]; !ok {
			programs[prog.IdWithTitle.Id] = &prog
		}
		if _, ok := dayToPrograms[prog.Day.Id]; !ok {
			dayToPrograms[prog.Day.Id] = make([]*VierdaagseProgram, 0)
		}
		dayToPrograms[prog.Day.Id] = append(dayToPrograms[prog.Day.Id], &prog)
	}
	for dayId := range dayToPrograms {
		slices.SortFunc(dayToPrograms[dayId], func(a, b *VierdaagseProgram) int {
			return a.FullStartTime.Compare(b.FullStartTime) // NB: sometimes the SortDate has typo's, so use the interpreted times
		})
	}

	return programs, dayToPrograms
}

func RenderSchedule(everything VierdaagseOverview) {
	days := SetupDays(everything)
	//locs := SetupLocations(everything)
	_, day2Program := SetupPrograms(everything)
	for _, day := range days {
		dayId := day.IdWithTitle.Id
		for _, program := range day2Program[dayId] {
			slog.Info("Program details", "day", day.IdWithTitle.Title, "eventTitle", program.IdWithTitle.Title,
				"startTime", program.FullStartTime,
				"endTime", program.FullEndTime,
				"duration", program.CalculatedDuration,
			)
		}
	}
}

// vim: cc=120:
