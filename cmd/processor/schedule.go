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
	Id        int
	Title     string
	Slug      string
	HasParent bool
	Children  []*Location
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

func SetupLocations(everything VierdaagseOverview) (map[int]*Location, []string) {
	locations := make(map[int]*Location)
	parentLocations := 0
	for _, loc := range everything.Locations {
		if _, ok := locations[loc.Id]; !ok {
			locations[loc.Id] = &Location{
				Children: make([]*Location, 0),
			}
		}
		theLoc := locations[loc.Id]
		theLoc.Id = loc.Id
		theLoc.Title = loc.Title
		theLoc.Slug = loc.Slug
		if loc.Parent > 0 {
			// Sub locations, fill into parent location's Children in separate loop
			theLoc.HasParent = true
			continue
		}
		parentLocations++
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

	sortedParents := make([]string, 0, parentLocations)
	for _, theLoc := range locations {
		if len(theLoc.Children) > 0 || !theLoc.HasParent {
			sortedParents = append(sortedParents, theLoc.Title)
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
	slices.Sort(sortedParents)
	return locations, sortedParents
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
	// Day -> Location (parent) -> Lcations (child) -> Event
	days := SetupDays(everything)
	locs, sortedParents := SetupLocations(everything)
	_, day2Program := SetupPrograms(everything)

	slog.Info("sortedParents", "sortedParents", sortedParents)

	for n, day := range days {
		fmt.Printf(`<section class="bg-red"><h1 class="bg-red sticky-0">Dag %d, <time datetime="%s">%s</time></h1>`+"\n", n+1, day.Date.Format(time.RFC3339), day.IdWithTitle.Title)
		dayId := day.IdWithTitle.Id
		// Don't look down, really inefficient loops ahead
		for _, parentLoc := range sortedParents {
			var theLoc *Location
			for _, loc := range locs {
				if loc.Title == parentLoc {
					theLoc = loc
					break
				}
			}
			fmt.Printf(`  <section id="lokatie-%s"><h2 class="sticky-1 bg-blue">%s</h2>`+"\n", locs[theLoc.Id].Slug, theLoc.Title)
			for _, childLoc := range theLoc.Children {
				fmt.Printf(`    <h3 class="sticky-2" id="lokatie-%s-%s">%s</h3>`+"\n", locs[theLoc.Id].Slug, childLoc.Slug, childLoc.Title)
				for _, program := range day2Program[dayId] {
					if program.Location.Id != childLoc.Id {
						continue
					}
					fmt.Printf(`    <div class="event"><h4><time>%s</time> - <time>%s</time></h4><dd class="artist">%s</dd><dd class="summary">%s`+
						`<a id="meer" href="#meer" class="hide">(meer)</a> <a id="minder" href="#minder" class="show">(minder)</a></dd><dd class="description">%s</dd></div>`+"\n",
						program.StartTime, program.EndTime, program.Title, program.DescriptionShort, program.Description)
					/*
						slog.Info("Program details", "day", day.IdWithTitle.Title, "eventTitle", program.IdWithTitle.Title,
							"startTime", program.FullStartTime,
							"endTime", program.FullEndTime,
							"duration", program.CalculatedDuration,
						)
					*/
				}
			}
			fmt.Print(`  </section>` + "\n")
		}
		fmt.Print(`</section>` + "\n")
	}
}

// vim: cc=120:
