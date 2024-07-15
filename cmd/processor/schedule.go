package main

import (
	"bytes"
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
		if prog.FullStartTime.IsZero() {
			prog.FullStartTime = appendEventTime(prog.Day.Date, prog.StartTime)
		}
		if prog.FullEndTime.IsZero() {
			prog.FullEndTime = appendEventTime(prog.Day.Date, prog.EndTime)
		}
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
<html lang="nl">
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta name="viewport" content="width=device-width" />
    <title>Vierdaagsefeesten 2024</title>
    <link rel="stylesheet" type="text/css" href="style.css" />
  </head>
  <body>
    <a name="top"></a>
    <div id="main" class="container">
`
var htmlSuffix = `
    </div>
    <script type="text/javascript">
    function highlightNow() {
      let now = new Date()
	    document.querySelectorAll(".event").forEach(x => {
 	     const [start, end] = Array.from(x.querySelectorAll("time")).map(y => new Date(y.getAttribute("datetime")))
 	     if (start <= now && now <= end) {
         x.classList.add("now")
       } else if (end <= now) {
         x.classList.add("past")
       }
	    })
      now = new Date();
      const nextMinute = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours(), now.getMinutes() + 1, 0, 0);
      setTimeout(highlightNow, nextMinute - now);
    }
    highlightNow()
    </script>
  </body>
</html>
`

var testingBanner = `
<div id="testing-banner">TESTOMGEVING, <a href="https://apploos.nl/4df/">klik hier</a> om naar de live website te gaan.</div>
`

func RenderSchedule(everything VierdaagseOverview) ([]byte, error) {
	buf := new(bytes.Buffer)
	var err error

	// Day -> Location (parent) -> Lcations (child) -> Event
	days := SetupDays(everything)
	locs, sortedParents := SetupLocations(everything)
	_, day2Program := SetupPrograms(everything)

	slog.Info("sortedParents", "sortedParents", sortedParents)

	eventIssues := make([]string, 0)

	_, err = fmt.Fprint(buf, htmlPrefix)
	if err != nil {
		return nil, err
	}
	if !*prod {
		_, err = fmt.Fprint(buf, testingBanner+"\n")
		if err != nil {
			return nil, err
		}
	}
	for n, day := range days {
		roze := onRozeWoensdagFromTime(day.Date.Add(1*time.Second + time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour))
		dayPrefix := ""
		daySectionClass := ""
		if roze {
			dayPrefix = "Roze "
			daySectionClass = "roze"
		}
		_, err = fmt.Fprintf(buf, `<section class="%s day" id="day-%d"><h1 class="sticky-0"><a href="#day-%d">Dag %d, %s<time datetime="%s">%s</time></a></h1>`+"\n",
			daySectionClass, n+1, n+1, n+1, dayPrefix, day.Date.Format(time.RFC3339), day.IdWithTitle.Title)
		if err != nil {
			return nil, err
		}
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
					_, err = fmt.Fprintf(buf, `  <section id="day-%d-lokatie-%s"><h2 class="sticky-1">%s</h2>`+"\n", n+1, locs[theLoc.Id].Slug, theLoc.Title)
					if err != nil {
						return nil, err
					}
					haveRenderedParent = true
				}
				for _, event := range renderedParentEvents {
					_, err = fmt.Fprint(buf, event)
					if err != nil {
						return nil, err
					}
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
						_, err = fmt.Fprintf(buf, `  <section id="day-%d-lokatie-%s"><h2 class="sticky-1">%s</h2>`+"\n", n+1, locs[theLoc.Id].Slug, theLoc.Title)
						if err != nil {
							return nil, err
						}
						haveRenderedParent = true
					}
					_, err = fmt.Fprintf(buf, `    <h3 class="sticky-2" id="day-%d-lokatie-%s-%s">%s</h3>`+"\n", n+1, locs[theLoc.Id].Slug, childLoc.Slug, childLoc.Title)
					if err != nil {
						return nil, err
					}
					for _, event := range renderedEvents {
						_, err = fmt.Fprint(buf, event)
						if err != nil {
							return nil, err
						}
					}
					haveRenderedEvents = true
					renderedEvents = make([]string, 0)
				}
			}
			if haveRenderedEvents {
				_, err = fmt.Fprint(buf, `  </section> <!-- `+theLoc.Title+` -->`+"\n")
				if err != nil {
					return nil, err
				}
			}
		}
		_, err = fmt.Fprint(buf, `</section>`+"\n")
		if err != nil {
			return nil, err
		}
	}
	if !*prod {
		_, err = fmt.Fprint(buf, `<!-- summarized event issues`+"\n")
		if err != nil {
			return nil, err
		}
		slices.Sort(eventIssues)
		for _, eventIssue := range eventIssues {
			_, err = fmt.Fprint(buf, `    `+eventIssue+"\n")
			if err != nil {
				return nil, err
			}
		}
		_, err = fmt.Fprint(buf, `end summarized event issues -->`+"\n")
		if err != nil {
			return nil, err
		}
	}
	if !everything.DirModTime.IsZero() && !everything.FileModTime.IsZero() {
		_, err = fmt.Fprintf(buf, `<!-- dir: %s, file: %s -->`+"\n", everything.DirModTime.Format(time.RFC3339), everything.FileModTime.Format(time.RFC3339))
		if err != nil {
			return nil, err
		}
	}
	_, err = fmt.Fprint(buf, htmlSuffix)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cleanDescription(in string) string {
	removals := []string{`<p>`, `</p>`, `<br>`}
	for _, remove := range removals {
		in = strings.ReplaceAll(in, remove, "")
	}
	return strings.TrimSpace(in)
}

func renderEvent(program *VierdaagseProgram, isEven bool) string {
	programSummary := program.DescriptionShort
	programDetails := cleanDescription(program.Description)

	if len(programDetails) < 3 {
		program.DataQualityIssues |= DQIDescriptionEmptyish
		slog.Info("Removed programDetails after cleaning, length less than 3", "program.Description", program.Description, "cleaned_programDetails", programDetails)
		programDetails = ""
	}

	if len(programSummary) == 0 && len(programDetails) > 0 {
		program.DataQualityIssues |= DQISummaryEmptyish
		lowestIndex := len(programDetails)
		lowestSeparator := "."
		for _, sep := range []string{".", "!", "?"} {
			if idx := strings.Index(programDetails, sep+" "); idx > -1 && idx < lowestIndex {
				lowestIndex = idx
				lowestSeparator = sep
			}
		}
		firstSentence, theRest, ok := strings.Cut(programDetails, lowestSeparator+" ")
		if !ok {
			// Swap summary and details
			program.DataQualityIssues |= DQINeededSummaryDescriptionSwap
			programSummary, programDetails = programDetails, programSummary
		} else {
			program.DataQualityIssues |= DQISummaryFromDescription
			programSummary = firstSentence + lowestSeparator
			programDetails = theRest
		}
	}

	ticketAddition := ""
	if program.TicketsPrice > 0 {
		if len(program.TicketsLink) > 0 {
			ticketAddition = ` (<a target="_blank" href="` + program.TicketsLink + `" title="Ticket kopen voor ` + program.Title + `">€</a>)`
		} else {
			ticketAddition = ` (€)`
		}
		if program.TicketsSoldOut {
			ticketAddition = ticketAddition + ` (uitverkocht)`
		}
	}
	if len(programDetails) == 0 || program.Title == programDetails {
		program.DataQualityIssues |= DQIOnlySummary
		return fmt.Sprintf(`    <div class="event"><h4 id="%s"><time datetime="%s">%s</time> - <time datetime="%s">%s</time> %s%s</h4><dd class="summary">%s</dd></div>`+"\n",
			formatProgramSlug(program), program.FullStartTime.Format(time.RFC3339), program.FullStartTime.Format("15:04"),
			program.FullEndTime.Format(time.RFC3339), program.FullEndTime.Format("15:04"), program.Title, ticketAddition, programSummary)
	}

	return fmt.Sprintf(`    <div class="event"><h4 id="%s"><time datetime="%s">%s</time> - <time datetime="%s">%s</time> %s%s</h4>`+
		`<input type="checkbox" class="meer-toggle" id="meer-%d" /><dd class="summary">%s `+
		`<label for="meer-%d" class="hide"></label></dd><dd class="description">%s</dd></div>`+"\n",
		formatProgramSlug(program), program.FullStartTime.Format(time.RFC3339), program.FullStartTime.Format("15:04"),
		program.FullEndTime.Format(time.RFC3339), program.FullEndTime.Format("15:04"), program.Title, ticketAddition, program.IdWithTitle.Id, programSummary,
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
