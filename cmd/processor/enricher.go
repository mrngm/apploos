package main

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CustomLocationId int
type CustomProgramId int

const (
	LocationThiemeLoodsId CustomLocationId = -37 - iota
	LocationDollarsId
	LocationOnderbroekId
	LocationOpstandId
)

var (
	// ProgramCustomId is a starting point for custom program IDs. Implementors must decrease this value upon using this
	// variable.
	ProgramCustomId CustomProgramId = -73
)

var (
	ThiemeloodsFilterFigureRegexp = regexp.MustCompile(`(?is)<figure .+</figure>`)
	ThiemeloodsFilterFooterRegexp = regexp.MustCompile(`(?is)<footer .+</footer>`)
	ThiemeloodsFilterDivRegexp    = regexp.MustCompile(`(?is)<div .+</div>`)
	ThiemeloodsFilterImgRegexp    = regexp.MustCompile(`(?is)<img .+ />`)
	ThiemeloodsFilterClassRegexp  = regexp.MustCompile(`(?is) class="[^"]+"`)
	ThiemeloodsFilterStrongRegexp = regexp.MustCompile(`(?is)</?strong>`)
	ThiemeloodsFilterHeaderRegexp = regexp.MustCompile(`(?is)</?h\d>`)
	ThiemeloodsFilterBreakRegexp  = regexp.MustCompile(`(?is)<br ?/?>`)
)

// EnrichScheduleWithThiemeloods expands the schedule with the events from the Thiemeloods
func EnrichScheduleWithThiemeloods(schedule *VierdaagseOverview, calendar ICalendar) error {
	// Add a location. We can do negative IDs that typically don't conflict with those from the Vierdaagse program
	for _, loc := range schedule.Locations {
		if loc.IdWithTitle.Id == int(LocationThiemeLoodsId) {
			return fmt.Errorf("cannot enrich schedule due to conflichting Location ID: %d, %v", loc.IdWithTitle.Id, loc)
		}
	}
	schedule.Locations = append(schedule.Locations, VierdaagseLocation{
		IdWithTitle: IdWithTitle{
			Id:    int(LocationThiemeLoodsId),
			Title: "Thiemeloods",
		},
		// TODO?
	})

	// Preformat the programs
	events := FilterThiemeloodsForVierdaagse(calendar, VierdaagseStartTime, VierdaagseEndTime)
	if len(events) == 0 {
		slog.Info("EnrichScheduleWithThiemeloods found no events")
		return nil
	}
	programs := make([]VierdaagseProgram, 0, len(calendar.Events))
	for _, event := range events {
		id := ProgramCustomId
		ticketPrice, err := strconv.ParseFloat(strings.ReplaceAll(event.TicketPrice, ",", "."), 10)
		if err != nil {
			slog.Error("cannot convert ticket price", "err", err, "ticketPrice", event.TicketPrice)
			ticketPrice = 999.0 // a quick look at their calendar shows everything has a cost
		}
		theDay, err := extractDayWithIdFromEvent(schedule, event)
		if err != nil {
			slog.Error("could not match date with event, skipping", "event", event)
			continue
		}

		// Thiemeloods may insert a variety of tags, let's strip a few of them out
		description := ThiemeloodsFilterFigureRegexp.ReplaceAllString(event.Description, "")
		description = ThiemeloodsFilterFooterRegexp.ReplaceAllString(description, "")
		description = ThiemeloodsFilterDivRegexp.ReplaceAllString(description, "")
		description = ThiemeloodsFilterImgRegexp.ReplaceAllString(description, "")
		description = ThiemeloodsFilterClassRegexp.ReplaceAllString(description, "")
		description = ThiemeloodsFilterStrongRegexp.ReplaceAllString(description, "")
		description = ThiemeloodsFilterHeaderRegexp.ReplaceAllString(description, "")
		description = ThiemeloodsFilterBreakRegexp.ReplaceAllString(description, "")
		description = strings.ReplaceAll(description, `Thiemeloods serveert tijden de Vierdaagse heerlijke gerechten van de houtskool barbecue met passende salade en rustiek stokbrood. Het is mogelijk een hiervoor combiticket concert/diner te kopen.`, "")
		description = strings.ReplaceAll(description, "\n", " ")

		description = description + ` Thiemeloods serveert tijden de Vierdaagse heerlijke gerechten van de houtskool barbecue met passende salade en rustiek stokbrood. Het is mogelijk een hiervoor combiticket concert/diner te kopen.`

		prog := VierdaagseProgram{
			IdWithTitle: IdWithTitle{
				Id:    int(id),
				Title: event.Summary,
			},
			Day: DayWithId{
				Id:   theDay.IdWithTitle.Id,
				Date: theDay.Date,
			},
			Location: SingularId{
				Id: int(LocationThiemeLoodsId),
			},
			Description:   description,
			TicketsPrice:  ticketPrice,
			TicketsLink:   event.TicketURL,
			FullStartTime: event.FullStartTime,
			FullEndTime:   event.FullEndTime,
		}
		programs = append(programs, prog)

		ProgramCustomId--
	}

	currentProgramIds := make(map[int]struct{})
	for _, currentProgram := range schedule.Programs {
		if _, ok := currentProgramIds[currentProgram.IdWithTitle.Id]; !ok {
			currentProgramIds[currentProgram.IdWithTitle.Id] = struct{}{}
		}
	}
	for _, program := range programs {
		if _, ok := currentProgramIds[program.IdWithTitle.Id]; ok {
			slog.Error("cannot add Thiemeloods program due to conflicting ID", "id", program.IdWithTitle.Id, "program", program)
			continue
		}
		slog.Info("adding program from Thiemeloods", "program", program)
		schedule.Programs = append(schedule.Programs, program)
	}

	return nil
}

func extractDayWithIdFromEvent(everything *VierdaagseOverview, event ICalendarEvent) (VierdaagseDay, error) {
	for _, day := range everything.Days {
		if event.FullStartTime.After(day.Date.Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) &&
			event.FullEndTime.Before(day.Date.AddDate(0, 0, 1).Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) {
			return day, nil
		}
	}
	return VierdaagseDay{}, fmt.Errorf("no match found")
}

// vim: cc=120:
