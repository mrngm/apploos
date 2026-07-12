package main

import (
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

var (
	// ProgramCustomId is a starting point for custom program IDs. Implementors must decrease this value upon using this
	// variable.
	ProgramCustomId CustomProgramId = -73

	CEST = time.FixedZone("CEST", 2*60*60)

	EmbeddedTimetableRegexp = regexp.MustCompile(`(?m)^\s*(?P<startTime>\d{2}\s*:\s*\d{2})\s*-\s*(?P<endTime>\d{2}\s*:\d{2})\s*(?P<description>.+)$`)
)

func GetCustomProgramId() int {
	ProgramCustomId--
	return int(ProgramCustomId)
}

func extractDayWithIdFromEvent(everything *VierdaagseOverview, startTime time.Time, endTime time.Time) (VierdaagseDay, error) {
	for _, day := range everything.Days {
		if startTime.After(day.Date.Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) &&
			startTime.Before(day.Date.AddDate(0, 0, 1).Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) &&
			endTime.After(day.Date.Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) &&
			endTime.Before(day.Date.AddDate(0, 0, 1).Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) {
			slog.Info("extracted day", "startTime", startTime, "endTime", endTime, "day", day)
			return day, nil
		}
	}
	return VierdaagseDay{}, fmt.Errorf("no match found")
}

func createProgram(schedule *VierdaagseOverview, title string, startTime time.Time, endTime time.Time, location VierdaagseLocation, description string) VierdaagseProgram {
	theDay, err := extractDayWithIdFromEvent(schedule, startTime, endTime)
	if err != nil {
		slog.Error("could not match date with event, skipping", "event", title, "startTime", startTime, "endTime", endTime)
		return VierdaagseProgram{}
	}

	return VierdaagseProgram{
		IdWithTitle: IdWithTitle{
			Id:    GetCustomProgramId(),
			Title: title,
		},
		Day: DayWithId{
			Id:   theDay.IdWithTitle.Id,
			Date: theDay.Date,
		},
		Location:        SingularId{Id: location.IdWithTitle.Id},
		Description:     description,
		FullStartTime:   startTime,
		FullEndTime:     endTime,
		RolloverImplied: true,
	}
}

func createEventTime(day, startHour, startMinute int) time.Time {
	year := 2026
	month := 7
	theDay := 16 + day
	if startHour < ROLLOVER_HOUR_FROM_START_OF_DAY {
		theDay++
	}
	return time.Date(year, time.Month(month), theDay, startHour, startMinute, 0, 0, CEST)
}

func createProgramFromEventProgram(schedule *VierdaagseOverview, location VierdaagseLocation, prog *Program) (VierdaagseProgram, bool) {
	theDay, err := extractDayWithIdFromEvent(schedule, prog.FullStartTime, prog.FullEndTime)
	if err != nil {
		slog.Error("could not match date with event, skipping", "event", prog.Title, "startTime", prog.FullStartTime, "endTime", prog.FullEndTime)
		return VierdaagseProgram{}, false
	}

	tpf, exact := prog.TicketPrice.Float64()
	if !exact {
		slog.Warn("creating float64 from decimal, lost precision", "decimal", prog.TicketPrice, "float64", tpf)
	}

	return VierdaagseProgram{
		IdWithTitle: IdWithTitle{
			Id:    GetCustomProgramId(),
			Title: prog.Title,
		},
		Day: DayWithId{
			Id:   theDay.IdWithTitle.Id,
			Date: theDay.Date,
		},
		Location:           SingularId{Id: location.IdWithTitle.Id},
		Slug:               prog.Slug,
		Description:        prog.Details,
		DescriptionShort:   prog.Summary,
		FullStartTime:      prog.FullStartTime,
		StartTimeEstimated: prog.StartTimeEstimated,
		FullEndTime:        prog.FullEndTime,
		EndTimeEstimated:   prog.EndTimeEstimated,
		RolloverImplied:    prog.RolloverImplied,
		TicketsPrice:       tpf,
		TicketsLink:        prog.TicketLink,
		TicketsSoldOut:     prog.TicketsSoldOut,
	}, true
}

// vim: cc=120:
