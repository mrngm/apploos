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
		// the next day. Thanks to @yorickvP, we use ROLLOVER_HOUR_FROM_START_OF_DAY to determine if the event should be
		// shifted to the next day
		prog.FullStartTime = appendEventTime(prog.Day.Date, prog.StartTime)
		prog.FullEndTime = appendEventTime(prog.Day.Date, prog.EndTime)
		if prog.FullStartTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
			prog.FullStartTime = prog.FullStartTime.AddDate(0, 0, 1)
		}
		if prog.FullEndTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
			prog.FullEndTime = prog.FullEndTime.AddDate(0, 0, 1)
		}
		if prog.FullStartTime.After(prog.FullEndTime) {
			// EndTime should be after StartTime
			prog.DataQualityIssues |= DQIEndTimeBeforeStart
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

var htmlPrefix = `<!DOCTYPE html>
<html>
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta name="viewport" content="width=device-width" />
    <title>${LOCATION} - Vierdaagsefeesten ${YEAR}</title>
    <link rel="stylesheet" type="text/css" href="style.css" />
    </style>
  </head>
  <body>
    <a name="top"></a>
    <div id="main" class="container">
`
var htmlSuffix = `
    </div>
  </body>
</html>
`

var testingBanner = `
<div id="testing-banner">TESTOMGEVING, <a href="https://apploos.nl/4df/">klik hier</a> om naar de live website te gaan.</div>
`

func RenderSchedule(everything VierdaagseOverview) {
	// Day -> Location (parent) -> Lcations (child) -> Event
	days := SetupDays(everything)
	locs, sortedParents := SetupLocations(everything)
	_, day2Program := SetupPrograms(everything)

	slog.Info("sortedParents", "sortedParents", sortedParents)

	eventIssues := make([]string, 0)

	fmt.Print(htmlPrefix)
	for n, day := range days {
		roze := onRozeWoensdagFromTime(day.Date.Add(1 * time.Second + time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY) * time.Hour))
		dayClass := "bg-red"
		locationClass := "bg-blue"
		subLocationClass := "bg-white"
		dayPrefix := ""
		if roze {
			dayClass = "bg-roze"
			dayPrefix = "Roze "
			locationClass = "bg-main-roze"
			subLocationClass = "bg-sub-roze"
		}
		fmt.Printf(`<section class="bg-red"><h1 class="%s sticky-0">Dag %d, %s<time datetime="%s">%s</time></h1>`+"\n",
			dayClass, n+1, dayPrefix, day.Date.Format(time.RFC3339), day.IdWithTitle.Title)
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
			renderedParentEvents := make([]string, 0)
			renderedEvents := make([]string, 0)

			// Render programs on the parent location
			even := false
			for _, program := range day2Program[dayId] {
				if program.Location.Id != theLoc.Id {
					continue
				}
				renderedParentEvents = append(renderedParentEvents, renderEvent(program, even))
				even = !even
				if eventIssue := DQIToString(program.DataQualityIssues); len(eventIssue) > 0 {
					eventIssues = append(eventIssues, formatProgramSlug(program)+": "+eventIssue)
				}
			}

			haveRenderedParent := false
			haveRenderedEvents := false
			if len(renderedParentEvents) > 0 {
				if !haveRenderedParent {
					fmt.Printf(`  <section id="lokatie-%s"><h2 class="sticky-1 %s">%s</h2>`+"\n", locs[theLoc.Id].Slug, locationClass, theLoc.Title)
					haveRenderedParent = true
				}
				for _, event := range renderedParentEvents {
					fmt.Print(event)
				}
				haveRenderedEvents = true
			}

			// Render programs on the child location
			for _, childLoc := range theLoc.Children {
				for _, program := range day2Program[dayId] {
					if program.Location.Id != childLoc.Id {
						continue
					}
					renderedEvents = append(renderedEvents, renderEvent(program, even))
					even = !even
					if eventIssue := DQIToString(program.DataQualityIssues); len(eventIssue) > 0 {
						eventIssues = append(eventIssues, formatProgramSlug(program)+": "+eventIssue)
					}
				}
				if len(renderedEvents) > 0 {
					if !haveRenderedParent {
						fmt.Printf(`  <section id="lokatie-%s"><h2 class="sticky-1 %s">%s</h2>`+"\n", locs[theLoc.Id].Slug, locationClass, theLoc.Title)
						haveRenderedParent = true
					}
					fmt.Printf(`    <h3 class="sticky-2 %s" id="lokatie-%s-%s">%s</h3>`+"\n", subLocationClass, locs[theLoc.Id].Slug, childLoc.Slug, childLoc.Title)
					for _, event := range renderedEvents {
						fmt.Print(event)
					}
					haveRenderedEvents = true
					renderedEvents = make([]string, 0)
				}
			}
			if haveRenderedEvents {
				fmt.Print(`  </section> <!-- ` + theLoc.Title + ` -->` + "\n")
			}
		}
		fmt.Print(`</section>` + "\n")
	}
	if !*prod {
		fmt.Print(testingBanner + "\n")
		fmt.Print(`<!-- summarized event issues` + "\n")
		slices.Sort(eventIssues)
		for _, eventIssue := range eventIssues {
			fmt.Print(`    ` + eventIssue + "\n")
		}
		fmt.Print(`end summarized event issues -->` + "\n")
	}
	fmt.Print(htmlSuffix)
}

func cleanDescription(in string) string {
	removals := []string{`<p>`, `</p>`, `<br>`}
	for _, remove := range removals {
		in = strings.ReplaceAll(in, remove, "")
	}
	return strings.TrimSpace(in)
}

func renderEvent(program *VierdaagseProgram, isEven bool) string {
	evenClass := "bg-even"
	if onRozeWoensdag(program) {
		evenClass = "bg-even-roze"
	}
	if !isEven {
		evenClass = "bg-odd"
		if onRozeWoensdag(program) {
			evenClass = "bg-odd-roze"
		}
	}

	programSummary := program.DescriptionShort
	programDetails := cleanDescription(program.Description)

	if len(programDetails) < 3 {
		program.DataQualityIssues |= DQIDescriptionEmptyish
		slog.Info("Removed programDetails after cleaning, length less than 3", "program.Description", program.Description, "cleaned_programDetails", programDetails)
		programDetails = ""
	}

	if len(programSummary) == 0 && len(programDetails) > 0 {
		program.DataQualityIssues |= DQISummaryEmptyish
		firstSentence, theRest, ok := strings.Cut(programDetails, ".")
		if !ok {
			// Swap summary and details
			program.DataQualityIssues |= DQINeededSummaryDescriptionSwap
			programSummary, programDetails = programDetails, programSummary
		} else {
			program.DataQualityIssues |= DQISummaryFromDescription
			programSummary = firstSentence + "."
			programDetails = theRest
		}
	}

	if len(programDetails) == 0 || program.Title == programDetails {
		program.DataQualityIssues |= DQIOnlySummary
		return fmt.Sprintf(`    <div class="event %s"><h4 id="%s"><time datetime="%s">%s</time> - <time datetime="%s">%s</time> %s</h4><dd class="summary">%s</dd></div>`+"\n",
			evenClass, formatProgramSlug(program), program.FullStartTime.Format(time.RFC3339), program.StartTime,
			program.FullEndTime.Format(time.RFC3339), program.EndTime, program.Title, programSummary)
	}

	return fmt.Sprintf(`    <div class="event %s"><h4 id="%s"><time datetime="%s">%s</time> - <time datetime="%s">%s</time> %s</h4>`+
		`<input type="checkbox" class="meer-toggle" id="meer-%d" /><dd class="summary">%s `+
		`<label for="meer-%d" class="hide"></label></dd><dd class="description">%s</dd></div>`+"\n",
		evenClass, formatProgramSlug(program), program.FullStartTime.Format(time.RFC3339), program.StartTime,
		program.FullEndTime.Format(time.RFC3339), program.EndTime, program.Title, program.IdWithTitle.Id, programSummary,
		program.IdWithTitle.Id,
		programDetails)
}

func logProgramDetailsWithDay(day VierdaagseDay, program *VierdaagseProgram) {
	slog.Info("Program details", "day", day.IdWithTitle.Title, "eventTitle", program.IdWithTitle.Title,
		"startTime", program.FullStartTime,
		"endTime", program.FullEndTime,
		"duration", program.CalculatedDuration,
	)
}

func formatProgramSlug(program *VierdaagseProgram) string {
	if program.Slug == "" {
		return fmt.Sprintf("unknown-slug-%d", program.IdWithTitle.Id)
	}
	return fmt.Sprintf("%s-%d", program.Slug, program.IdWithTitle.Id)
}

var (
	RozeWoensdagStart = time.Date(2024, 7, 17, ROLLOVER_HOUR_FROM_START_OF_DAY, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	RozeWoensdagEnd   = RozeWoensdagStart.AddDate(0, 0, 1).Add(-1 * time.Nanosecond)
)

func onRozeWoensdag(program *VierdaagseProgram) bool {
	return onRozeWoensdagFromTime(program.FullStartTime) && onRozeWoensdagFromTime(program.FullEndTime)
}
func onRozeWoensdagFromTime(cmp time.Time) bool {
	return cmp.After(RozeWoensdagStart) && cmp.Before(RozeWoensdagEnd)
}

// vim: cc=120:
