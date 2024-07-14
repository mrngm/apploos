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

	CEST = time.FixedZone("CEST", 2*60*60)
)

func GetCustomProgramId() int {
	ProgramCustomId--
	return int(ProgramCustomId)
}

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
		id := GetCustomProgramId()
		ticketPrice, err := strconv.ParseFloat(strings.ReplaceAll(event.TicketPrice, ",", "."), 10)
		if err != nil {
			slog.Error("cannot convert ticket price", "err", err, "ticketPrice", event.TicketPrice)
			ticketPrice = 999.0 // a quick look at their calendar shows everything has a cost
		}
		theDay, err := extractDayWithIdFromEvent(schedule, event.FullStartTime, event.FullEndTime)
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

func extractDayWithIdFromEvent(everything *VierdaagseOverview, startTime time.Time, endTime time.Time) (VierdaagseDay, error) {
	for _, day := range everything.Days {
		if startTime.After(day.Date.Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) &&
			endTime.Before(day.Date.AddDate(0, 0, 1).Add(time.Duration(ROLLOVER_HOUR_FROM_START_OF_DAY)*time.Hour)) {
			return day, nil
		}
	}
	return VierdaagseDay{}, fmt.Errorf("no match found")
}

// EnrichScheduleWithOpstand expands the schedule with the events from Kollektief Kafé De Opstand
func EnrichScheduleWithOpstand(schedule *VierdaagseOverview) error {
	// Add a location. We can do negative IDs that typically don't conflict with those from the Vierdaagse program
	for _, loc := range schedule.Locations {
		if loc.IdWithTitle.Id == int(LocationOpstandId) {
			return fmt.Errorf("cannot enrich schedule due to conflichting Location ID: %d, %v", loc.IdWithTitle.Id, loc)
		}
	}
	theLoc := VierdaagseLocation{
		IdWithTitle: IdWithTitle{
			Id:    int(LocationOpstandId),
			Title: "Café De Opstand",
		},
		// TODO?
	}

	schedule.Locations = append(schedule.Locations, theLoc)
	programs := []VierdaagseProgram{
		// Dag 1
		createProgram(schedule, "Sjonnie & Het Talent", createEventTime(1, 21, 0), createEventTime(1, 21, 45), theLoc, ""),
		createProgram(schedule, "Hi-Fi Spitfires (UK)", createEventTime(1, 22, 0), createEventTime(1, 22, 45), theLoc, "Hi Fi Spitfires are a three piece punk rock band who formed in 2008. We play all of our own material. Described by one journalist as a band who would fit comfortably in a record collection between Give 'em enough rope by the Clash and SLF's Inflammable Material. We have probably played more gigs abroad than in UK. Steve Straughan vocals & guitar, Tony Taylor bass & vocals, Dean Ross drums & bv's. (hifispitfires.bandcamp.com)"),
		createProgram(schedule, "Chaos 8 (UK)", createEventTime(1, 23, 0), createEventTime(1, 23, 45), theLoc, `FORMED IN THE EARLY PART OF 2012 BY GUITARIST/SONGWRITER PAUL WILLIAMS AND SINGER/LYRICIST BEKI STRAUGHAN ON LEAD VOCALS, WHO WERE LATER JOINED BY MUSICAL ALLIES IN THE FORM OF BASSIST JAMES "OZ" BOWEY, COMPLETING THIS JUGGERNAUT OF A FOUR PIECE IS STEVEN NAISBET ON DRUMS/PERCUSSION. (chaos8.bandcamp.com)`),

		// Dag 2
		createProgram(schedule, "Kelsey", createEventTime(2, 20, 0), createEventTime(2, 20, 45), theLoc, "KELSEY is niet één persoon, maar een groep personen. Een collectief van vrienden met een gedeelde liefde voor harde, chaotische en tegendraadse muziek. Een band die zich muzikaal ergens op de grens tussen Metal, (Post-)Hardcore en Punk bevindt, en de muren hiertussen volledig afbreekt. (popronde.nl)"),
		createProgram(schedule, "Curselifter", createEventTime(2, 21, 0), createEventTime(2, 21, 45), theLoc, "Curselifter uit Utrecht speelt bijtende metallic hardcore. De teksten brengen seksueel geweld, machogedrag en machtsmisbruik onder de aandacht, terwijl de muziek inspireert tot stagediven en vechten met je vrienden. (popronde.nl)"),
		createProgram(schedule, "Outahead", createEventTime(2, 22, 0), createEventTime(2, 22, 45), theLoc, "Gewapend met stage antics waar Kurt Cobain 'u' tegen zou zeggen, verklaart Outahead de oorlog aan het verzadigde post-punk landschap. 'Soundtrack to a Car Crash'. (popronde.nl)"),
		createProgram(schedule, "Park and Ride", createEventTime(2, 23, 0), createEventTime(2, 23, 45), theLoc, "Park and Ride is een machine ontstaan op een zesde verdieping in Amsterdam. Met hun atmosferische blend van post-punk en hardcore nemen ze je mee in hun wereld, waar chaos en orde in harmonie samen leven. (popronde.nl)"),

		// Dag 3
		createProgram(schedule, "Vierdaagse Yoga", createEventTime(3, 16, 0), createEventTime(3, 20, 30), theLoc, "Rek- en strekoefeningen voor zowel de 4daagselopers als degenen die het avondprogramma doen. Vergeet niet je eigen mat mee te nemen."),
		createProgram(schedule, "Politie Warnsveld", createEventTime(3, 21, 0), createEventTime(3, 21, 45), theLoc, "Politie Warnsveld is niet voor de zwakhartigen. Geef jezelf de kans om door je as te klappen op onze ontembare energie. Met onze enge tunes en rake teksten pakken we je bij de hand op weg naar de wondere wereld van Pret-Ska! Zet je schrap, want het is een tsunami aan genot, een paardenmiddel waardoor je schuimbekkend het daglicht weer hoopt te zien. (popronde.nl)"),
		createProgram(schedule, "Don't Wake Me", createEventTime(3, 22, 0), createEventTime(3, 22, 45), theLoc, "Van intiem en doordringend, naar 'recht op je bakkes'. Met invloeden van o.a. Radiohead en Elbow, biedt Don't wake me een gelaagde en dynamische ervaring. Het trio bestaat sinds 2017. Alle leden van de band hebben een groot deel van hun muzikale ervaring opgedaan met andere instrumenten, bands en genres. (glurenbijdeburen.nl)"),
		createProgram(schedule, "Spit", createEventTime(3, 23, 0), createEventTime(3, 23, 45), theLoc, ""),

		// Dag 4
		createProgram(schedule, "DJ ANUBYS", createEventTime(4, 17, 0), createEventTime(4, 20, 30), theLoc, "Kom de benen weer los dansen bij DJ ANUBYS (instagram.com/anubysbeats/)"),
		createProgram(schedule, "Shitman", createEventTime(4, 21, 0), createEventTime(4, 21, 45), theLoc, "Onze moshpit is een safe space. (instagram.com/shitman.band/"),
		createProgram(schedule, "Trashvault", createEventTime(4, 22, 0), createEventTime(4, 22, 45), theLoc, "Hardcore Improv band. (instagram.com/trashvaultnoise/"),
		createProgram(schedule, "Portray", createEventTime(4, 23, 0), createEventTime(4, 23, 45), theLoc, "PORTRAY, a dynamic and genre-bending band that delivers an unforgettable musical experience. Their music captivates audiences with crystal-clear melodies and an irresistible energy from the first note. Drawing inspiration from garage rock, punk, and psychedelic realms, PORTRAY's sound blazes with intensity and joy. Their music reflects personal struggles with identity and existence, while also giving a voice to the voiceless in navigating the difficulties of living in modern society. (checksonar.nl/portray)"),

		// Dag 5
		createProgram(schedule, "Roze Wodka Woensdag", createEventTime(5, 20, 1), createEventTime(5, 00, 00), theLoc, "De Roze Woensdag editie van de Wodka Woensdag"),
		createProgram(schedule, "Trui", createEventTime(5, 21, 0), createEventTime(5, 21, 45), theLoc, "TRUI de band, bestaande uit oud-klasgenoten en tindermatches, maakt feministische punk en catchy muziek met anderzijds maatschappijkritische noot. Met nummers over manspreading, de klimaatcrisis, cryptocurrency en slecht planten ouderschap omschrijft TRUI hun genre zelf het liefst als Hang Youth meets Kinderen voor Kinderen meets Folk Punk. (popronde.nl)"),
		createProgram(schedule, "Designer Violence", createEventTime(5, 22, 0), createEventTime(5, 22, 45), theLoc, "Wilde synthesizers, op hol geslagen drumcomputers en emotionele vocalen van twee vrouwen die in je gezicht schreeuwen. Vanuit een gedeelde passie voor alles wat gruizig, ruw en underground is, weet Designer Violence elke zaal weer te transformeren tot een zweterige nachtclub waar de zon niet opkomt en de subs net wat te hard staan. (popronde.nl)"),
		createProgram(schedule, "Miss Conduct & The Homewreckers", createEventTime(5, 23, 0), createEventTime(5, 23, 45), theLoc, `𝓒𝓾𝓽 𝓶𝔂 𝓵𝓲𝓯𝓮 𝓲𝓷𝓽𝓸 𝓹𝓲𝓮𝓬𝓮𝓼, 𝓽𝓱𝓲𝓼 𝓲𝓼 𝓶𝔂 𝓵𝓪𝓼𝓽 𝓻𝓮𝓼𝓸𝓻𝓽
💀🖤𝕊𝕃𝔸𝕐𝕄𝕆 🖤💀 (instagram.com/xx_miss_conduct_xx/)`),

		// Dag 6
		createProgram(schedule, "DJ ANUBYS", createEventTime(6, 17, 0), createEventTime(6, 20, 30), theLoc, "Kom de benen weer los dansen bij DJ ANUBYS (instagram.com/anubysbeats/)"),
		createProgram(schedule, "Dood Vogeltje", createEventTime(6, 21, 0), createEventTime(6, 21, 45), theLoc, "Je moeders favo sludgy punx. Martijn: drums, Hans: gitaar, Lourens: zang, bas. (doodvogeltje.bandcamp.com)"),
		createProgram(schedule, "Statues On Fire (Brazil)", createEventTime(6, 22, 0), createEventTime(6, 22, 45), theLoc, "Punk Rock from Santo André/SP, Brazil. (instagram.com/statuesonfire)"),
		createProgram(schedule, "Periot", createEventTime(6, 23, 0), createEventTime(6, 23, 45), theLoc, "These days them girls are the real bastards! Deze punkband komt uit Arnhem en staat bekend om de lekkere gitiaar riffs, zwaar distorted bas, neanderthaler drums en opzweepende meeschreeuw-teksten. Als je nog niet van hun EP Petra of hun jaarlijkse Kutfeest hebt gehoord, dan mis je echt wat. Dus, trek zondagse outfit aan en spring een gat in de lucht! (popronde.nl)"),

		// Dag 7
		createProgram(schedule, "Face painting", createEventTime(7, 15, 30), createEventTime(7, 19, 30), theLoc, "Laura Jasmijns Blossoming Body Art. Wil je een unieke look met de Vierdaagse? Laat dan je gezicht beschilderen bij Café De Opstand!"),
		createProgram(schedule, "Flukes of Sendington (Australia)", createEventTime(7, 21, 0), createEventTime(7, 21, 45), theLoc, ""),
		createProgram(schedule, "Razernij", createEventTime(7, 22, 0), createEventTime(7, 22, 45), theLoc, "Razernij is een nieuwe black metal band die zijn oorsprong vindt in de duistere krochten van Nijmegen. Hun muziek wordt gekenmerkt door hypnotiserende riffs, blast beats en angstaanjagende vocalen die luisteraars meeslepen naar een wereld van duisternis en chaos. Het debuutoptreden van Razernij vond plaats tijdens het evenement “Heel Nijmegen Plat”, waar ze het publiek verbijsterden met hun meedogenloze en intense performance. (metalfrom.nl)"),
		createProgram(schedule, "Sing Along Riot", createEventTime(7, 23, 0), createEventTime(7, 23, 45), theLoc, "Punkrock karaoke from the Netherlands. Always wanted to sing in a punkband? Pick a song, climb the stage and rock out with us! (instagram.com/singalongriot/)"),
	}

	currentProgramIds := make(map[int]struct{})
	for _, currentProgram := range schedule.Programs {
		if _, ok := currentProgramIds[currentProgram.IdWithTitle.Id]; !ok {
			currentProgramIds[currentProgram.IdWithTitle.Id] = struct{}{}
		}
	}
	for _, program := range programs {
		if _, ok := currentProgramIds[program.IdWithTitle.Id]; ok {
			slog.Error("cannot add Opstand program due to conflicting ID", "id", program.IdWithTitle.Id, "program", program)
			continue
		}
		slog.Info("adding program from Opstand", "program", program)
		schedule.Programs = append(schedule.Programs, program)
	}
	return nil
}

// EnrichScheduleWithOnderbroek expands the schedule with the events from De Onderbroek
func EnrichScheduleWithOnderbroek(schedule *VierdaagseOverview) error {
	// Add a location. We can do negative IDs that typically don't conflict with those from the Vierdaagse program
	for _, loc := range schedule.Locations {
		if loc.IdWithTitle.Id == int(LocationOnderbroekId) {
			return fmt.Errorf("cannot enrich schedule due to conflichting Location ID: %d, %v", loc.IdWithTitle.Id, loc)
		}
	}
	theLoc := VierdaagseLocation{
		IdWithTitle: IdWithTitle{
			Id:    int(LocationOnderbroekId),
			Title: "De Onderbroek",
		},
		// TODO?
	}

	onderbroekEventSuffix := `Entree: donatie / donation (cash only), er kan geen cash gepind worden bij de kassa. Er zijn pinautomaten in de omgeving.`

	schedule.Locations = append(schedule.Locations, theLoc)
	programs := []VierdaagseProgram{
		// Dag 1
		// 0
		createProgram(schedule, "Ravetrain", createEventTime(1, 23, 0), createEventTime(2, 5, 0), theLoc, onderbroekEventSuffix),

		// Dag 2
		// 1
		createProgram(schedule, "Dj Soulseek & Team MUTE", createEventTime(2, 23, 0), createEventTime(3, 5, 0), theLoc, "90’s eurodance ai madness. "+onderbroekEventSuffix),

		// Dag 3
		// 2
		createProgram(schedule, "Chaos in Nijmegen", createEventTime(3, 20, 0), createEventTime(4, 0, 30), theLoc, "Pressure Pact, Stresssyteem, Bot Mes, Karel Anker en de beste stuurlui. Tickets: alleen deurverkoop. Entree: 5,- ~ 10,-. Vanaf 20u geopend. Tot middenacht zijn er bands."),
		// 3
		createProgram(schedule, "CIN AFTERPARTY", createEventTime(3, 0, 30), createEventTime(4, 6, 0), theLoc, "AcidTekno by: Bas Punkt ~ Johnny Crash ~ Dr. Graftak ~ Frixion Fanatic ~ Kayayay Madkat. "+onderbroekEventSuffix),

		// Dag 4
		// 4
		createProgram(schedule, "Brown Note Booking - Hippie Death Cult (US)", createEventTime(4, 21, 30), createEventTime(5, 2, 30), theLoc, "Explosieve hardrock met een vleugje psych, een snufje blues en een flinke scheut metal, Hippie Death Cult en Diggeth zullen Nijmegen op haar grondvesten doen trillen! Hippie Death Cult's journey through shameless and triumphant artistic expression has led them to become a vibrant force in the realms of psychedelia and riff-heavy rock n’ roll. This journey has not been without its challenges, but the band has always managed to emerge stronger and more determined than ever. Throughout their formative years, the band underwent what proved to be a very significant evolution, transitioning from a 4-piece to a more cohesive and harmonious power trio. This lineup currently consists of guitarist and founder Eddie Brnabic, vocalist and bassist Laura Phillips, and drummer Harry Silvers. (grotebroek.nl). Entree gift vanaf €5,- cash only."),
		// 5
		createProgram(schedule, "Brown Note Booking - Diggeth (NL)", createEventTime(4, 21, 30), createEventTime(5, 2, 30), theLoc, `Explosieve hardrock met een vleugje psych, een snufje blues en een flinke scheut metal, Hippie Death Cult en Diggeth zullen Nijmegen op haar grondvesten doen trillen! Diggeth, goede bekenden en graag geziene gasten in Nijmegen, timmeren enorm aan de weg. Tegenwoordig ook regelmatig op tour over de plas. Take 50 years of Hard Rock, Metal, Southern Rock and a bit of Progressive Rock; Diggeth will digest it and will spew out their mix of all these genres in songs with hooks, heaviness and groove! This kick ass 3-piece band does give a complete new meaning to “Metal-‘n-Roll” with their breakthrough album Gringos Galacticos. Their live shows are legendary; the mix of genres is never forced, it flows, it pulses, it grinds and most important; it grooves! It leaves you with an impressive "beep" in your ears and makes you wonder: "How can a three piece sound so big?" (grotebroek.nl). Entree gift vanaf €5,- cash only.`),
		// 6
		createProgram(schedule, "Brown Note Booking - DJ Coconaut & Miss MaryLane", createEventTime(4, 21, 30), createEventTime(5, 2, 30), theLoc, "PhosPhor Visual zal het vuurwerk completeren en DJ duo Coconaut & Miss MaryLane zullen het feestelijke gehalte nog wat opkrikken. (grotebroek.nl). Entree gift vanaf €5,- cash only."),

		// Dag 5
		// 7
		createProgram(schedule, "Bloody Queers: DANKE≠CISTEM", createEventTime(5, 23, 0), createEventTime(6, 5, 0), theLoc, onderbroekEventSuffix),

		// Dag 6
		// 8
		createProgram(schedule, "IMMERGE Bass Music Party", createEventTime(6, 23, 0), createEventTime(7, 5, 0), theLoc, "Tijdens de gezelligste week van het jaar in Nijmegen, De Vierdaagse Feesten, staan wij met Immerge in De Onderbroek. De line-up is nog even geheim, maar zoals je van ons verwacht presenteren wij een avond met een breed scala aan bass music. "+onderbroekEventSuffix),
	}
	currentProgramIds := make(map[int]struct{})
	for _, currentProgram := range schedule.Programs {
		if _, ok := currentProgramIds[currentProgram.IdWithTitle.Id]; !ok {
			currentProgramIds[currentProgram.IdWithTitle.Id] = struct{}{}
		}
	}
	for idx, program := range programs {
		if _, ok := currentProgramIds[program.IdWithTitle.Id]; ok {
			slog.Error("cannot add Onderbroek program due to conflicting ID", "id", program.IdWithTitle.Id, "program", program)
			continue
		}
		switch idx {
		case 2, 4, 5:
			program.TicketsPrice = 5.0
		}
		slog.Info("adding program from Onderbroek", "program", program)
		schedule.Programs = append(schedule.Programs, program)
	}
	return nil
}

func createProgram(schedule *VierdaagseOverview, title string, startTime time.Time, endTime time.Time, location VierdaagseLocation, description string) VierdaagseProgram {
	theDay, err := extractDayWithIdFromEvent(schedule, startTime, endTime)
	if err != nil {
		slog.Error("could not match date with event, skipping", "event", title)
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
		Location:      SingularId{Id: location.IdWithTitle.Id},
		Description:   description,
		FullStartTime: startTime,
		FullEndTime:   endTime,
	}
}

func createEventTime(day, startHour, startMinute int) time.Time {
	year := 2024
	month := 7
	theDay := 12 + day
	return time.Date(year, time.Month(month), theDay, startHour, startMinute, 0, 0, CEST)
}

// vim: cc=120:
