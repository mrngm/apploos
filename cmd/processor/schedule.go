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

	"github.com/shopspring/decimal"
)

var (
	locationNameRemapping = map[string]string{
		"De Kaaij - aan de Waal": "De Kaaij",
		"Kaaij Hoog":             "De Kaaij (Hoog)",
	}
	childLocationPrefixDifferences = map[string]string{
		"De Kaaij aan de Waal": "De Kaaij",
		"Grote markt":          "Grote Markt",
	}
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
		if alias, ok := locationNameRemapping[loc.Title]; ok {
			theLoc.Alias = alias
		}
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
			theChildLoc.Alias = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(theChildLoc.Title, theParentLoc.Title), "- "))
			if theChildLoc.Alias == theChildLoc.Title {
				theChildLoc.DataQualityIssues |= DQIChildLocationContainsDifferentPrefix
				slog.Warn("child location has different prefix than parent's title", "childTitle", theChildLoc.Title, "parentTitle", theParentLoc.Title, "parentAlias", theParentLoc.Alias)
				for knownChildPrefix, _ := range childLocationPrefixDifferences {
					if strings.HasPrefix(theChildLoc.Alias, knownChildPrefix) {
						theChildLoc.Alias = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(theChildLoc.Title, knownChildPrefix), "- "))
						slog.Warn("replaced child location alias prefix", "childTitle", theChildLoc.Title, "knownChildPrefix", knownChildPrefix, "replacement", theChildLoc.Alias, "parentTitle", theParentLoc.Title, "parentAlias", theParentLoc.Alias)
						break
					}
				}
			}
			theParentLoc.Children = append(theParentLoc.Children, theChildLoc)
		}
	}

	sortedMainLocations := make([]*Location, 0, parentLocations)
	for _, theLoc := range locations {
		if len(theLoc.Children) > 0 || !theLoc.HasParent {
			sortedMainLocations = append(sortedMainLocations, theLoc)
			// Sort the children based on name
			slices.SortFunc(theLoc.Children, CompareLocationByEffectiveTitle)
			for _, childLoc := range theLoc.Children {
				if len(childLoc.Children) > 0 {
					slog.Error("child location has more locations", "len", len(childLoc.Children))
				}
			}
		}
	}
	slices.SortFunc(sortedMainLocations, CompareLocationByEffectiveTitle)
	return locations, sortedMainLocations
}

func CompareLocationByEffectiveTitle(a, b *Location) int {
	if a.Alias == "" && b.Alias != "" {
		return strings.Compare(a.Title, b.Alias)
	}
	if a.Alias != "" && b.Alias == "" {
		return strings.Compare(a.Alias, b.Title)
	}
	if a.Alias != "" && b.Alias != "" {
		return strings.Compare(a.Alias, b.Alias)
	}

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
	genreIdsToGenre := make(map[ /* genreId */ int]string)

	for _, genre := range everything.Genres {
		if _, ok := genreIdsToGenre[genre.Id]; !ok {
			genreIdsToGenre[genre.Id] = strings.ToLower(genre.Title)
		}
	}

	dayToPrograms := make(map[ /* dayId */ int][] /* sorted slice based on start_time full details */ *Program)
	programs := make(map[int]*Program)
	for _, prog := range everything.Programs {
		prog := prog
		tpf := decimal.NewFromFloatWithExponent(prog.TicketsPrice, -2)
		program := &Program{
			Id:             prog.IdWithTitle.Id,
			LocationId:     prog.Location.Id,
			Title:          prog.IdWithTitle.Title,
			Slug:           formatProgramSlug(prog),
			Summary:        cleanupHTML(prog.DescriptionShort),
			Details:        cleanupHTML(prog.Description),
			TicketPrice:    tpf,
			TicketLink:     prog.TicketsLink,
			TicketsSoldOut: prog.TicketsSoldOut,
			Genres:         make([]string, 0, len(prog.Genres)),
		}

		for _, genre := range prog.Genres {
			if genreTitle, ok := genreIdsToGenre[genre.Id]; ok {
				program.Genres = append(program.Genres, genreTitle)
			} else {
				slog.Warn("program contains genreId not in overview", "genre", genre, "progId", program.Id)
			}
		}
		if len(program.Genres) == 0 {
			program.DataQualityIssues |= DQINoGenres
		}
		slices.Sort(program.Genres)

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
			if strings.HasPrefix(prog.SortDate, "20260718") {
				dayId = 496721
				theDayDate = time.Date(2026, 7, 18, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20260719") {
				dayId = 496722
				theDayDate = time.Date(2026, 7, 19, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20260720") {
				dayId = 496723
				theDayDate = time.Date(2026, 7, 20, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20260721") {
				dayId = 496724
				theDayDate = time.Date(2026, 7, 21, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20260722") {
				dayId = 496725
				theDayDate = time.Date(2026, 7, 22, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20260723") {
				dayId = 496726
				theDayDate = time.Date(2026, 7, 23, 0, 0, 0, 0, CEST)
			} else if strings.HasPrefix(prog.SortDate, "20260724") {
				dayId = 496727
				theDayDate = time.Date(2026, 7, 24, 0, 0, 0, 0, CEST)
			}
		} else {
			dayId = prog.Day.Id
			theDayDate = prog.Day.Date
		}
		program.DayId = dayId
		if prog.FullStartTime.IsZero() {
			// No full start time defined yet, let's try to derive it
			if prog.StartTime == "" {
				// Try to derive the start time from SortDate
				if (strings.HasPrefix(prog.SortDate, "2026071") || strings.HasPrefix(prog.SortDate, "2026072")) && len(prog.SortDate) == 12 {
					program.FullStartTime = appendEventTime(theDayDate, prog.SortDate[8:10]+":"+prog.SortDate[10:12])
					program.StartTimeEstimated = true
				}
			} else {
				program.FullStartTime = appendEventTime(theDayDate, prog.StartTime)
			}
		} else {
			program.FullStartTime = prog.FullStartTime
			program.StartTimeEstimated = prog.StartTimeEstimated
		}
		if prog.FullEndTime.IsZero() {
			// No full end time defined yet, let's try to derive it
			if prog.EndTime == "" {
				// Guesstimate that the program takes 30m
				if prog.FullStartTime.IsZero() {
					// Use our derivation from SortDate
					program.FullEndTime = program.FullStartTime.Add(30 * time.Minute)
				} else {
					program.FullEndTime = prog.FullStartTime.Add(30 * time.Minute)
				}
				program.EndTimeEstimated = true
			} else {
				program.FullEndTime = appendEventTime(theDayDate, prog.EndTime)
			}
		} else {
			program.FullEndTime = prog.FullEndTime
			program.EndTimeEstimated = prog.EndTimeEstimated
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

		//slog.Info("adding programs from schedule", "program", *program)

		// Try to detect an additional timetable in the summary
		if matches := EmbeddedTimetableRegexp.FindAllStringSubmatch(program.Summary, -1); matches != nil {
			startTime := EmbeddedTimetableRegexp.SubexpIndex("startTime")
			endTime := EmbeddedTimetableRegexp.SubexpIndex("endTime")
			description := EmbeddedTimetableRegexp.SubexpIndex("description")
			for n, match := range matches {
				slog.Info("found timetable entry for program", "progId", program.Id, "title", program.Title, "start", match[startTime], "end", match[endTime], "desc", strings.TrimSpace(strings.TrimPrefix(match[description], ":")))
				// Add additional program
				additionalProgram := &Program{
					Id:            program.Id * -1,
					LocationId:    program.LocationId,
					Title:         strings.TrimSpace(strings.TrimPrefix(match[description], ":")),
					Slug:          program.Slug + "-add-" + strconv.Itoa(n),
					FullStartTime: appendEventTime(theDayDate, match[startTime]).Add(1 * time.Second), // for sorting after the parent event
					FullEndTime:   appendEventTime(theDayDate, match[endTime]),
				}
				if additionalProgram.FullStartTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
					additionalProgram.FullStartTime = additionalProgram.FullStartTime.AddDate(0, 0, 1)
				}
				if additionalProgram.FullEndTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
					additionalProgram.FullEndTime = additionalProgram.FullEndTime.AddDate(0, 0, 1)
				}
				dayToPrograms[dayId] = append(dayToPrograms[dayId], additionalProgram)
			}
		}
		// Try to detect an additional timetable in the details
		if matches := EmbeddedTimetableRegexp.FindAllStringSubmatch(program.Details, -1);
			matches != nil &&
			!strings.Contains(strings.ToLower(program.Details), "salsa stage") &&
			!strings.Contains(program.Details, "🪩✨") { // Kelfkensbos has a duplicate schedule
			startTime := EmbeddedTimetableRegexp.SubexpIndex("startTime")
			endTime := EmbeddedTimetableRegexp.SubexpIndex("endTime")
			description := EmbeddedTimetableRegexp.SubexpIndex("description")
			for n, match := range matches {
				slog.Info("found timetable entry for program", "progId", program.Id, "title", program.Title, "start", match[startTime], "end", match[endTime], "desc", strings.TrimSpace(strings.TrimPrefix(match[description], ":")))
				// Add additional program
				additionalProgram := &Program{
					Id:            program.Id * -1,
					LocationId:    program.LocationId,
					Title:         strings.TrimSpace(strings.TrimPrefix(match[description], ":")),
					Slug:          program.Slug + "-add-" + strconv.Itoa(n),
					FullStartTime: appendEventTime(theDayDate, match[startTime]).Add(1 * time.Second), // for sorting after the parent event
					FullEndTime:   appendEventTime(theDayDate, match[endTime]),
				}
				if additionalProgram.FullStartTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
					additionalProgram.FullStartTime = additionalProgram.FullStartTime.AddDate(0, 0, 1)
				}
				if additionalProgram.FullEndTime.Hour() < ROLLOVER_HOUR_FROM_START_OF_DAY {
					additionalProgram.FullEndTime = additionalProgram.FullEndTime.AddDate(0, 0, 1)
				}
				dayToPrograms[dayId] = append(dayToPrograms[dayId], additionalProgram)
			}
		}
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
		"decimalGtZero":  func(d decimal.Decimal) bool { return d.GreaterThan(decimal.Decimal{}) },
		"isVuurwerk":     func(s string) bool { return strings.Contains(strings.ToLower(s), "waal in vlammen") },
		"titleOrAlias": func(title, alias string) string {
			if alias != "" {
				return alias
			}
			return title
		},
		"join": func(elems []string, sep string) string { return strings.Join(elems, sep) },
	}

	tpl := template.Must(template.New("schedule").Funcs(templateFuncs).Parse(htmlTemplate))
	templateData := struct {
		StylesheetChecksumShort string
		ScriptingChecksumShort  string
		Schedule                *Schedule
		TestingBanner           template.HTML
		Navigation              template.HTML
		IsProduction            bool
	}{
		StylesheetChecksumShort: stylesheetCheckumShort,
		ScriptingChecksumShort:  scriptingChecksumShort,
		Schedule:                schedule,
		TestingBanner:           template.HTML(testingBanner),
		Navigation:              template.HTML(navigation),
		IsProduction:            *prod,
	}

	err = tpl.Execute(buf, templateData)

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderEvent(program *VierdaagseProgram, isEven bool) string {
	panic("renderEvent called")
	return ""
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
	RozeWoensdagStart = time.Date(2026, 7, 22, ROLLOVER_HOUR_FROM_START_OF_DAY, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	RozeWoensdagEnd   = RozeWoensdagStart.AddDate(0, 0, 1).Add(-1 * time.Nanosecond)
)

func onRozeWoensdag(program *VierdaagseProgram) bool {
	return onRozeWoensdagFromTime(program.FullStartTime) && onRozeWoensdagFromTime(program.FullEndTime)
}
func onRozeWoensdagFromTime(cmp time.Time) bool {
	return cmp.After(RozeWoensdagStart) && cmp.Before(RozeWoensdagEnd)
}

// vim: cc=120:
