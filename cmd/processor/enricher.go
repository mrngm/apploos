package main

import (
	"fmt"
	"log/slog"
)

type CustomLocationId int
type CustomProgramId int

const (
	LocationThiemeLoodsId CustomLocationId = -37 - iota
	LocationDollarsId
	LocationOnderbroekId
	LocationOpstandId
	LocationDeVereenigingId
	LocationDeWitteRaafId
)

var LocationTitlesToIds = map[string]CustomLocationId{
	"Thiemeloods":                          LocationThiemeLoodsId,
	"Dollars Muziekcafé":                   LocationDollarsId,
	"De Onderbroek":                        LocationOnderbroekId,
	"Café De Opstand":                      LocationOpstandId,
	"De Vereeniging":                       LocationDeVereenigingId,
	"De Witte Raaf":                        LocationDeWitteRaafId,
	"Valkhof Festival - Poort":             65968, // For manually splitting acts on this location
	"Valkhof Festival - St. Nicolaaskapel": 65966, // For manually splitting acts on this location
}

var UnknownLocationId CustomLocationId = -127

func EnrichGenericEvent(schedule *VierdaagseOverview, event EventData) error {
	// If the event did not have a known location ID, generate one
	if _, ok := LocationTitlesToIds[event.LocationTitle]; !ok {
		LocationTitlesToIds[event.LocationTitle] = UnknownLocationId
		UnknownLocationId--
	}
	lid := LocationTitlesToIds[event.LocationTitle]

	// Add a location. We can do negative IDs that typically don't conflict with those from the Vierdaagse program
	for _, loc := range schedule.Locations {
		if !event.LocationOverride && loc.IdWithTitle.Id == int(lid) {
			return fmt.Errorf("cannot enrich schedule due to conflichting Location ID: %d, %v", loc.IdWithTitle.Id, loc)
		}
	}
	theLoc := VierdaagseLocation{
		IdWithTitle: IdWithTitle{
			Id:    int(lid),
			Title: event.LocationTitle,
		},
		Slug: event.LocationSlug,
	}
	schedule.Locations = append(schedule.Locations, theLoc)

	programs := make([]VierdaagseProgram, 0, len(event.Programs))
	for _, prog := range event.Programs {
		if programFromEvent, ok := createProgramFromEventProgram(schedule, theLoc, prog); ok {
			programs = append(programs, programFromEvent)
		}
	}

	currentProgramIds := make(map[int]struct{})
	for _, currentProgram := range schedule.Programs {
		if _, ok := currentProgramIds[currentProgram.IdWithTitle.Id]; !ok {
			currentProgramIds[currentProgram.IdWithTitle.Id] = struct{}{}
		}
	}
	for _, program := range programs {
		if _, ok := currentProgramIds[program.IdWithTitle.Id]; ok {
			slog.Error("cannot add program due to conflicting ID", "id", program.IdWithTitle.Id, "program", program, "event", event)
			continue
		}
		slog.Info("adding programs from eventData", "program", program)
		schedule.Programs = append(schedule.Programs, program)
	}
	return nil

}

// vim: cc=120:
