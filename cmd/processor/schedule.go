package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"
)

func SetupDays(everything VierdaagseOverview) []*Day {
	days := make([]*Day, len(everything.Days))
	for i, day := range everything.Days {
		days[i] = &Day{
			Id:    day.IdWithTitle.Id,
			Title: day.IdWithTitle.Title,
			Date:  day.Date,
		}
	}
	slices.SortFunc(days, func(a, b *Day) int {
		return a.Date.Compare(b.Date)
	})
	return days
}

func SetupLocations(everything VierdaagseOverview) (map[int]*Location, []*Location) {
	locations := make(map[int]*Location)
	parentLocations := 0
	for _, loc := range everything.Locations {
		if _, ok := locations[loc.Id]; !ok {
			locations[loc.Id] = &Location{
				ProgramsByDay:        make(map[int][]*Program, 0),
				Children:             make([]*Location, 0),
				TotalNrProgramsByDay: make(map[int]int, 0),
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

	sortedMainLocations := make([]*Location, 0, parentLocations)
	for _, theLoc := range locations {
		if len(theLoc.Children) > 0 || !theLoc.HasParent {
			sortedMainLocations = append(sortedMainLocations, theLoc)
			// Sort the children based on name
			slices.SortFunc(theLoc.Children, CompareLocationByTitle)
			for _, childLoc := range theLoc.Children {
				if len(childLoc.Children) > 0 {
					slog.Error("child location has more locations", "len", len(childLoc.Children))
				}
			}
		}
	}
	slices.SortFunc(sortedMainLocations, CompareLocationByTitle)
	return locations, sortedMainLocations
}

func CompareLocationByTitle(a, b *Location) int {
	return strings.Compare(a.Title, b.Title)
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

func SetupPrograms(everything VierdaagseOverview) (map[int]*Program, map[int][]*Program) {
	dayToPrograms := make(map[ /* dayId */ int][] /* sorted slice based on start_time full details */ *Program)
	programs := make(map[int]*Program)
	for _, prog := range everything.Programs {
		prog := prog
		program := &Program{
			Id:         prog.IdWithTitle.Id,
			LocationId: prog.Location.Id,
			Title:      prog.IdWithTitle.Title,
			Slug:       formatProgramSlug(prog),
			Summary:    prog.DescriptionShort,
			Details:    cleanupHTML(prog.Description),
		}

		// Record some data quality issues, try to fix some
		if len(program.Details) < 3 {
			program.DataQualityIssues |= DQIDescriptionEmptyish
			slog.Debug("Removed programDetails after cleaning, length less than 3", "prog.Description", prog.Description, "cleaned_programDetails", program.Details)
			program.Details = ""
		}

		if len(program.Summary) == 0 && len(program.Details) > 0 {
			program.DataQualityIssues |= DQISummaryEmptyish
			lowestIndex := len(program.Details)
			lowestSeparator := "."
			for _, sep := range []string{".", "!", "?"} {
				if idx := strings.Index(program.Details, sep+" "); idx > -1 && idx < lowestIndex {
					lowestIndex = idx
					lowestSeparator = sep
				}
			}
			firstSentence, theRest, ok := strings.Cut(program.Details, lowestSeparator+" ")
			if !ok {
				// Swap summary and details
				program.DataQualityIssues |= DQINeededSummaryDescriptionSwap
				program.Summary, program.Details = program.Details, program.Summary
			} else {
				program.DataQualityIssues |= DQISummaryFromDescription
				program.Summary = firstSentence + lowestSeparator
				program.Details = theRest
			}
		}

		if len(program.Details) == 0 || program.Title == program.Details {
			program.DataQualityIssues |= DQIOnlySummary
		}

		// Calculate full start time and full end time. The start time is on the scheduled day. The end time might be on
		// the next day. Thanks to @yorickvP, we use ROLLOVER_HOUR_FROM_START_OF_DAY to determine if the event should be
		// shifted to the next day
		dayId := 0
		theDayDate := time.Time{}
		if prog.Day.IsZero() {
			program.DataQualityIssues |= DQINoDaySet
			if strings.HasPrefix(prog.SortDate, "20250712") {
				dayId = 370538
				theDayDate = time.Date(2025, 7, 12, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20250713") {
				dayId = 370546
				theDayDate = time.Date(2025, 7, 13, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20250714") {
				dayId = 370547
				theDayDate = time.Date(2025, 7, 14, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20250715") {
				dayId = 370548
				theDayDate = time.Date(2025, 7, 15, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20250716") {
				dayId = 370549
				theDayDate = time.Date(2025, 7, 16, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20250717") {
				dayId = 370550
				theDayDate = time.Date(2025, 7, 17, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20250718") {
				dayId = 370551
				theDayDate = time.Date(2025, 7, 18, 0, 0, 0, 0, CEST)
			}
		} else {
			dayId = prog.Day.Id
			theDayDate = prog.Day.Date
		}
		program.DayId = dayId
		if prog.StartTime == "" {
			// Try to derive the start time from SortDate
			if strings.HasPrefix(prog.SortDate, "2025071") && len(prog.SortDate) == 12 {
				program.FullStartTime = appendEventTime(theDayDate, prog.SortDate[8:10]+":"+prog.SortDate[10:12])
				program.StartTimeEstimated = true
			}
		}
		if prog.EndTime == "" {
			// Guesstimate that the program takes 30m
			program.FullEndTime = prog.FullStartTime.Add(30 * time.Minute)
			program.EndTimeEstimated = true
		}
		if program.FullStartTime.IsZero() && prog.StartTime != "" {
			program.FullStartTime = appendEventTime(theDayDate, prog.StartTime)
		}
		if program.FullEndTime.IsZero() && prog.EndTime != "" {
			program.FullEndTime = appendEventTime(theDayDate, prog.EndTime)
		}
		if !prog.RolloverImplied && program.FullStartTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
			program.FullStartTime = program.FullStartTime.AddDate(0, 0, 1)
		}
		if !prog.RolloverImplied && program.FullEndTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
			program.FullEndTime = program.FullEndTime.AddDate(0, 0, 1)
		}
		if program.FullStartTime.After(program.FullEndTime) {
			// EndTime should be after StartTime
			program.DataQualityIssues |= DQIEndTimeBeforeStart
		}
		program.CalculatedDuration = program.FullEndTime.Sub(program.FullStartTime)

		if _, ok := programs[program.Id]; !ok {
			programs[program.Id] = program
		}
		if _, ok := dayToPrograms[dayId]; !ok {
			dayToPrograms[dayId] = make([]*Program, 0)
		}
		dayToPrograms[dayId] = append(dayToPrograms[dayId], program)
	}
	for dayId := range dayToPrograms {
		slices.SortFunc(dayToPrograms[dayId], func(a, b *Program) int {
			return a.FullStartTime.Compare(b.FullStartTime) // NB: sometimes the SortDate has typo's, so use the interpreted times
		})
	}

	return programs, dayToPrograms
}

func augmentLocationWithPrograms(location *Location, dayId int, progs []*Program) {
	if _, ok := location.ProgramsByDay[dayId]; !ok {
		location.ProgramsByDay[dayId] = make([]*Program, 0)
	}
	if _, ok := location.TotalNrProgramsByDay[dayId]; !ok {
		location.TotalNrProgramsByDay[dayId] = 0
	}
	for _, prog := range progs {
		if prog.LocationId != location.Id {
			continue
		}
		// SetupPrograms already sorts by FullStartTime
		location.ProgramsByDay[dayId] = append(location.ProgramsByDay[dayId], prog)
		location.TotalNrProgramsByDay[dayId]++
	}
}

func SetupSchedule(everything VierdaagseOverview) *Schedule {
	_, sortedMainLocations := SetupLocations(everything)
	_, dayToPrograms := SetupPrograms(everything)

	// Augment each main location and their child locations with their program for each day
	for dayId, progs := range dayToPrograms {
		for _, location := range sortedMainLocations {
			augmentLocationWithPrograms(location, dayId, progs)
			for _, childLocation := range location.Children {
				augmentLocationWithPrograms(childLocation, dayId, progs)
				// Add total number of events to parent location
				location.TotalNrProgramsByDay[dayId] += childLocation.TotalNrProgramsByDay[dayId]
			}
		}
	}

	return &Schedule{
		Days:      SetupDays(everything),
		Locations: sortedMainLocations,
	}
}

func RenderSchedule(everything VierdaagseOverview) ([]byte, error) {
	buf := new(bytes.Buffer)
	var err error

	schedule := SetupSchedule(everything)
	// Day -> Location (parent) -> Lcations (child) -> Event
	//locs, sortedParents := SetupLocations(everything)
	//_, day2Program := SetupPrograms(everything)

	//slog.Info("sortedParents", "sortedParents", sortedParents)

	//eventIssues := make([]string, 0)

	templateFuncs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"isRoze": func(t time.Time) bool {
			cmp := t.Add(1*time.Second + time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)
			return cmp.After(RozeWoensdagStart) && cmp.Before(RozeWoensdagEnd)
		},
		"formatRFC3339DatetimeAttr": func(t time.Time) template.HTMLAttr {
			return template.HTMLAttr(`datetime="` + t.Format(time.RFC3339) + `"`)
		},
		"formatHourMins": func(t time.Time) string { return t.Format("15:04") },
	}

	tpl := template.Must(template.New("schedule").Funcs(templateFuncs).Parse(htmlTemplate))
	templateData := struct {
		StylesheetChecksumShort string
		Schedule                *Schedule
	}{
		StylesheetChecksumShort: stylesheetCheckumShort,
		Schedule:                schedule,
	}

	err = tpl.Execute(buf, templateData)

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

/*

	if !*prod {
		_, err = fmt.Fprint(buf, testingBanner+"\n")
		if err != nil {
			return nil, err
		}
		_, err = fmt.Fprint(buf, navigation+"\n")
		if err != nil {
			return nil, err
		}
	}
	for n, day := range days {
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
					err = renderParentLocation(buf, n, locs[theLoc.Id].Slug, theLoc.Title)
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
						err = renderParentLocation(buf, n, locs[theLoc.Id].Slug, theLoc.Title)
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
*/

func renderEvent(program *VierdaagseProgram, isEven bool) string {
	return ""
	/*
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

	   specialtyClass := ""

	   	if strings.Contains(strings.ToLower(program.Title), "vuurwerkspektakel") {
	   		specialtyClass = " fire-text"
	   	}

	   startTimeEstimatedIndicator := ""
	   endTimeEstimatedIndicator := ""

	   	if program.StartTimeEstimated {
	   		startTimeEstimatedIndicator = "?"
	   	}

	   	if program.EndTimeEstimated {
	   		endTimeEstimatedIndicator = "?"
	   	}

	   	if len(programDetails) == 0 || program.Title == programDetails {
	   		program.DataQualityIssues |= DQIOnlySummary
	   		return fmt.Sprintf(`    <div class="event%s"><h4 id="%s"><time datetime="%s">%s</time>%s - <time datetime="%s">%s</time>%s %s%s</h4><dd class="summary">%s</dd></div>`+"\n",
	   			specialtyClass,
	   			formatProgramSlug(program), program.FullStartTime.Format(time.RFC3339), program.FullStartTime.Format("15:04"), startTimeEstimatedIndicator,
	   			program.FullEndTime.Format(time.RFC3339), program.FullEndTime.Format("15:04"), endTimeEstimatedIndicator,
	   			program.Title, ticketAddition, programSummary)
	   	}

	   return fmt.Sprintf(`    <div class="event%s"><h4 id="%s"><time datetime="%s">%s</time>%s - <time datetime="%s">%s</time>%s %s%s</h4>`+

	   	`<input type="checkbox" class="meer-toggle" id="meer-%d" /><dd class="summary">%s `+
	   	`<label for="meer-%d" class="hide"></label></dd><dd class="description">%s</dd></div>`+"\n",
	   	specialtyClass,
	   	formatProgramSlug(program), program.FullStartTime.Format(time.RFC3339), program.FullStartTime.Format("15:04"), startTimeEstimatedIndicator,
	   	program.FullEndTime.Format(time.RFC3339), program.FullEndTime.Format("15:04"), endTimeEstimatedIndicator,
	   	program.Title, ticketAddition, program.IdWithTitle.Id, programSummary,
	   	program.IdWithTitle.Id,
	   	programDetails)
	*/
}

func logProgramDetailsWithDay(day VierdaagseDay, program *VierdaagseProgram) {
	slog.Info("Program details", "day", day.IdWithTitle.Title, "eventTitle", program.IdWithTitle.Title,
		"startTime", program.FullStartTime,
		"endTime", program.FullEndTime,
		"duration", program.CalculatedDuration,
		"startTimeEstimated", program.StartTimeEstimated,
	)
}

func formatProgramSlug(program VierdaagseProgram) string {
	if program.Slug == "" {
		return fmt.Sprintf("unknown-slug-%d", program.IdWithTitle.Id)
	}
	return fmt.Sprintf("%s-%d", program.Slug, program.IdWithTitle.Id)
}

var (
	RozeWoensdagStart = time.Date(2025, 7, 16, ROLLOVER_HOUR_FROM_START_OF_DAY, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	RozeWoensdagEnd   = RozeWoensdagStart.AddDate(0, 0, 1).Add(-1 * time.Nanosecond)
)

func onRozeWoensdag(program *VierdaagseProgram) bool {
	return onRozeWoensdagFromTime(program.FullStartTime) && onRozeWoensdagFromTime(program.FullEndTime)
}
func onRozeWoensdagFromTime(cmp time.Time) bool {
	return cmp.After(RozeWoensdagStart) && cmp.Before(RozeWoensdagEnd)
}

// vim: cc=120:
