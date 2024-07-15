package main

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

type CustomLocationId int
type CustomProgramId int

const (
	LocationThiemeLoodsId CustomLocationId = -37 - iota
	LocationDollarsId
	LocationOnderbroekId
	LocationOpstandId
	LocationDeVereenigingId
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
		createProgram(schedule, "Ravetrain", createEventTime(1, 23, 0), createEventTime(1, 5, 0), theLoc, onderbroekEventSuffix),

		// Dag 2
		// 1
		createProgram(schedule, "Dj Soulseek & Team MUTE", createEventTime(2, 23, 0), createEventTime(2, 5, 0), theLoc, "90’s eurodance ai madness. "+onderbroekEventSuffix),

		// Dag 3
		// 2
		createProgram(schedule, "Chaos in Nijmegen", createEventTime(3, 20, 0), createEventTime(3, 0, 30), theLoc, "Pressure Pact, Stresssyteem, Bot Mes, Karel Anker en de beste stuurlui. Tickets: alleen deurverkoop. Entree: 5,- ~ 10,-. Vanaf 20u geopend. Tot middenacht zijn er bands."),
		// 3
		createProgram(schedule, "CIN AFTERPARTY", createEventTime(3, 0, 30), createEventTime(3, 6, 0), theLoc, "AcidTekno by: Bas Punkt ~ Johnny Crash ~ Dr. Graftak ~ Frixion Fanatic ~ Kayayay Madkat. "+onderbroekEventSuffix),

		// Dag 4
		// 4
		createProgram(schedule, "Brown Note Booking - Hippie Death Cult (US)", createEventTime(4, 21, 30), createEventTime(4, 2, 30), theLoc, "Explosieve hardrock met een vleugje psych, een snufje blues en een flinke scheut metal, Hippie Death Cult en Diggeth zullen Nijmegen op haar grondvesten doen trillen! Hippie Death Cult's journey through shameless and triumphant artistic expression has led them to become a vibrant force in the realms of psychedelia and riff-heavy rock n’ roll. This journey has not been without its challenges, but the band has always managed to emerge stronger and more determined than ever. Throughout their formative years, the band underwent what proved to be a very significant evolution, transitioning from a 4-piece to a more cohesive and harmonious power trio. This lineup currently consists of guitarist and founder Eddie Brnabic, vocalist and bassist Laura Phillips, and drummer Harry Silvers. (grotebroek.nl). Entree gift vanaf €5,- cash only."),
		// 5
		createProgram(schedule, "Brown Note Booking - Diggeth (NL)", createEventTime(4, 21, 30), createEventTime(4, 2, 30), theLoc, `Explosieve hardrock met een vleugje psych, een snufje blues en een flinke scheut metal, Hippie Death Cult en Diggeth zullen Nijmegen op haar grondvesten doen trillen! Diggeth, goede bekenden en graag geziene gasten in Nijmegen, timmeren enorm aan de weg. Tegenwoordig ook regelmatig op tour over de plas. Take 50 years of Hard Rock, Metal, Southern Rock and a bit of Progressive Rock; Diggeth will digest it and will spew out their mix of all these genres in songs with hooks, heaviness and groove! This kick ass 3-piece band does give a complete new meaning to “Metal-‘n-Roll” with their breakthrough album Gringos Galacticos. Their live shows are legendary; the mix of genres is never forced, it flows, it pulses, it grinds and most important; it grooves! It leaves you with an impressive "beep" in your ears and makes you wonder: "How can a three piece sound so big?" (grotebroek.nl). Entree gift vanaf €5,- cash only.`),
		// 6
		createProgram(schedule, "Brown Note Booking - DJ Coconaut & Miss MaryLane", createEventTime(4, 21, 30), createEventTime(4, 2, 30), theLoc, "PhosPhor Visual zal het vuurwerk completeren en DJ duo Coconaut & Miss MaryLane zullen het feestelijke gehalte nog wat opkrikken. (grotebroek.nl). Entree gift vanaf €5,- cash only."),

		// Dag 5
		// 7
		createProgram(schedule, "Bloody Queers: DANKE≠CISTEM", createEventTime(5, 23, 0), createEventTime(5, 5, 0), theLoc, onderbroekEventSuffix),

		// Dag 6
		// 8
		createProgram(schedule, "IMMERGE Bass Music Party", createEventTime(6, 23, 0), createEventTime(6, 5, 0), theLoc, "Tijdens de gezelligste week van het jaar in Nijmegen, De Vierdaagse Feesten, staan wij met Immerge in De Onderbroek. De line-up is nog even geheim, maar zoals je van ons verwacht presenteren wij een avond met een breed scala aan bass music. "+onderbroekEventSuffix),
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

// EnrichScheduleWithDollars expands the schedule with the events from Café Dollars
func EnrichScheduleWithDollars(schedule *VierdaagseOverview) error {
	// Add a location. We can do negative IDs that typically don't conflict with those from the Vierdaagse program
	for _, loc := range schedule.Locations {
		if loc.IdWithTitle.Id == int(LocationDollarsId) {
			return fmt.Errorf("cannot enrich schedule due to conflichting Location ID: %d, %v", loc.IdWithTitle.Id, loc)
		}
	}
	theLoc := VierdaagseLocation{
		IdWithTitle: IdWithTitle{
			Id:    int(LocationDollarsId),
			Title: "Dollars Muziekcafé",
		},
		// TODO?
	}

	schedule.Locations = append(schedule.Locations, theLoc)
	programs := []VierdaagseProgram{
		// Dag 1
		createProgram(schedule, "The GunZ of Boston", createEventTime(1, 18, 0), createEventTime(1, 19, 0), theLoc, "HE GUNZ OF BOSTON are a CLASSIC ROCK group. Their MUSIC harks back to the days of that great, bygone era of CLASSIC ROCK , the SEVENTIES and the EIGHTIES. When MUSIC would combine POWER and EMOTION, PASSION and GRACE. When a song would tell a story, when a SINGER was a SINGER and ROCK GUITARS ruled the world. (thegunzofboston.bandcamp.com)"),
		createProgram(schedule, "Funktie Elders", createEventTime(1, 21, 30), createEventTime(1, 22, 30), theLoc, "Funktie Elders is een achtkoppige coverband uit Nijmegen, opgericht in 2021. Met een onweerstaanbare mix van energie, talent en aanstekelijke muzikaliteit, toveren ze elk optreden om tot een gegarandeerd feest. (vierdaagsefeesten.nl)"),
		createProgram(schedule, "Dollars Mash-up", createEventTime(1, 0, 30), createEventTime(1, 1, 30), theLoc, ""),

		// Dag 2
		createProgram(schedule, "Aangeschoten", createEventTime(2, 18, 0), createEventTime(2, 19, 0), theLoc, ""),
		createProgram(schedule, "KeToBra", createEventTime(2, 20, 30), createEventTime(2, 21, 30), theLoc, `KeToBra is een Nederlandstalige popgroep uit Nijmegen, opgericht in 2017 door Ke, To en Bra. Beginnend als panfluit-groep sloeg Ketobra na hun single "WiFi In De Trein" een nieuwe richting in en lieten de panfluitmuziek achter zich. De formule van de band bestaat voornamelijk uit het combineren van humoristische teksten en diverse genres. (popronde.nl)`),
		createProgram(schedule, "The Evergreens", createEventTime(2, 0, 30), createEventTime(2, 2, 30), theLoc, "The Evergreens met een breed assortiment met rock en pop! Op de blokken tussen de menigte maakt dit een speciaal en uniek optreden. De fatastische stem van Robine Roordink wordt begeleid door gitaarvirtuoos Jeroen Wallar-Diemont, Kimon op de bas en Twan voor het ritme!! De zaterdag huisband van Dollars Nijmegen! (vierdaagsefeesten.nl)"),

		// Dag 3
		createProgram(schedule, "Tachycardia", createEventTime(3, 19, 0), createEventTime(3, 20, 0), theLoc, "Dit arsenaal aan muzikaal talent, dat al sinds 2004 onder de naam Tachycardia faam vergaart zowel binnen als buiten de [Medische] faculteit, bestaat uit zeven muzikanten en een manager. De band beschikt over een drummer, pianist, saxofonist, (bas) gitaristen en zangers. Dit geeft ze de mogelijkheid een zeer breed repetoire ten gehore te brengen aan zowel trouwe fans als nieuwe fans van de altijd groeiende fanbase. Aangezien ‘een breed repertoire’ niet specificeert of dat loopt van Jan Smit tot Lee Towers of van The Red Hot Chili Peppers tot Kyteman zal dit bericht daar iets duidelijker over zijn; het tweede. Tot gecoverde artiesten behoren onder andere RHCP, Arctic Monkeys, Bruno Mars, Calvin Harris, Guns ’n Roses, Robbie Williams, Imagine Dragons en nog veel meer in dit altijd veranderende repertoire. Belangrijker is wellicht om te vermelden dat er genoeg muzikaliteit aanwezig is om een eigen creatieve draai te geven aan iedere cover. Maar wat is een foutloze muzikale uitvoering zonder bijpassend charisma om de muziek tot leven te brengen? Daarom staat Tachycardia met een fysiologische tachycardie op het podium. Tachycardia treedt met veel plezier op tijdens activiteiten van de MFVN, zoals de ouderdag, de muziekmaand van de Aesculaaf en feestelijke onderwijsafsluitingen. Ook wordt buiten de faculteit hard aan de weg getimmerd en kan je Tachycardia zien op menig gala en feest. (mfvn.nl)"),
		createProgram(schedule, "Bootleg Betty", createEventTime(3, 22, 0), createEventTime(3, 23, 0), theLoc, "Bootleg Betty deinst er niet voor terug om je alle hoeken van de rootsmuziek te laten horen. Vol energie vuurt het Nijmeegse vijftal haar meerstemmige mix van rockabilly, pop, country en rock-‘n-roll op je af. De eigentijdse benadering van deze traditionele invloeden resulteert in een herkenbaar geluid dat alles behalve gedateerd is. (bootlegbetty.nl)"),
		createProgram(schedule, "The Newly Wets", createEventTime(3, 0, 30), createEventTime(3, 1, 30), theLoc, "Van country tot pop naar blues en rock, het komt allemaal voorbij. The Newly Wets geven muziek met een knipoog een nieuwe betekenis. (vierdaagsefeesten.nl)"),

		// Dag 4
		createProgram(schedule, "Band Zonder Faam", createEventTime(4, 18, 0), createEventTime(4, 19, 0), theLoc, ""),
		createProgram(schedule, "FOK!", createEventTime(4, 21, 30), createEventTime(4, 22, 30), theLoc, ""),
		createProgram(schedule, "The Kelly Cats", createEventTime(4, 0, 30), createEventTime(4, 1, 30), theLoc, "Not your average coverband. Hits & classics with a rock ‘n roll twist. (instagram.com/thekellycats.band/)"),

		// Dag 5
		createProgram(schedule, "Manatee", createEventTime(5, 18, 0), createEventTime(5, 19, 0), theLoc, "Manatee is een Nijmeegse coverband met hits van alle tijden. Je kan meezingen met een een breed repertoire aan guilty pleasures en gouwe ouwe; van Harry Styles tot ABBA en van Stevie Wonder tot Robbie Williams. (vierdaagsefeesten.nl)"),
		createProgram(schedule, "The Breaks", createEventTime(5, 21, 0), createEventTime(5, 22, 0), theLoc, "Pop/Rock/Coverband. Bruiloften, feesten & partijen! Van Tina Turner tot Bon Jovi tot Dua Lipa en nog véél meer! (instagram.com/the_breaks_nl/)"),
		createProgram(schedule, "The Tributes", createEventTime(5, 0, 30), createEventTime(5, 1, 30), theLoc, "Deze band brengt een avond vol tributes aan de legendes van pop- en rockmuziek. Denk bij THE TRIBUTES niet aan achtergrondmuziek, maar een show met Jeroen Waller-Diemont op gitaar, ondersteund door Twan arts op cajon en krachtige vocalen van charismatische zangeres Karlijn. De band, bestaande uit 3 jonge muzikanten, is ontstaan en gegroeid in Dollars: dé live kroeg van Nijmegen. Op de setlijst staan covers van o.a. Tina Turner, ACDC, Bon Jovi en Queen. Het publiek zal met de muzikale TRIBUTES van het begin tot het eind meebrullen. Aangestoken door de overdosis aan enthousiasme van de band. (vierdaagsefeesten.nl)"),

		// Dag 6
		createProgram(schedule, "The Oracles", createEventTime(6, 19, 0), createEventTime(6, 20, 0), theLoc, ""),
		createProgram(schedule, "De Gang Van Zaken", createEventTime(6, 21, 30), createEventTime(6, 22, 30), theLoc, "Wij zijn De Gang Van Zaken, een in Nijmegen gevestigde band met Utrechtse roots. We spelen een eigen combinatie van indie, pop, funk en een vleugje rock. (vierdaagsefeesten.nl)"),
		createProgram(schedule, "Royal Blend", createEventTime(6, 0, 30), createEventTime(6, 1, 30), theLoc, ""),

		// Dag 7
		createProgram(schedule, "Blueshift", createEventTime(7, 21, 0), createEventTime(7, 22, 0), theLoc, "Verwacht bij Blueshift een optreden vol meeslepende vocalen, scheurende gitaarsolo’s, rommelende bas en opzwepende drums! Deze vierkoppige Nijmeegse band speelt al bijna 10 jaar samen. De eerste jaren speelden ze, toen nog allemaal studenten aan de Radboud Universiteit, vooral covers van bekende nummers uit de blues, bluesrock en classic rock. Tegenwoordig gebruiken ze die invloeden voor hun eigen materiaal en hebben ze sinds 2020 verschillende nummers uitgebracht. Na deelname aan de Roos van Nijmegen in Doornroosje duikt de band regelmatig weer de studio in en zijn ze in de tussentijd op allerlei plekken live te vinden! (blueshiftband.nl)"),
		createProgram(schedule, "Low Hangin' Fruit", createEventTime(7, 1, 0), createEventTime(7, 2, 0), theLoc, "Low Hangin’ Fruit is not “your average coverband”. Vier ervaren enthousiaste muzikanten met een passie voor jouw favoriete guilty pleasures. Een unieke setlist waarin alle tijden worden aangetikt van 80’s tot 00’s en dit alles met een knipoog. Altijd een rockversie willen horen van Eternal Flame of Crazy in Love? Zin in de tropische vibes van Dreadlock Holiday? Of liever rocken op In The End? Dan is dit jouw band! Meezingen, meedansen en een lekker potje headbangen zijn geen opties, het is verplicht! (lowhanginfruitband.nl)"),
	}

	currentProgramIds := make(map[int]struct{})
	for _, currentProgram := range schedule.Programs {
		if _, ok := currentProgramIds[currentProgram.IdWithTitle.Id]; !ok {
			currentProgramIds[currentProgram.IdWithTitle.Id] = struct{}{}
		}
	}
	for _, program := range programs {
		if _, ok := currentProgramIds[program.IdWithTitle.Id]; ok {
			slog.Error("cannot add Dollars program due to conflicting ID", "id", program.IdWithTitle.Id, "program", program)
			continue
		}
		slog.Info("adding program from Dollars", "program", program)
		schedule.Programs = append(schedule.Programs, program)
	}
	return nil
}

// EnrichScheduleWithVereeniging expands the schedule with the events from Stadsschouwburg De Vereeniging
func EnrichScheduleWithVereeniging(schedule *VierdaagseOverview) error {
	// Add a location. We can do negative IDs that typically don't conflict with those from the Vierdaagse program
	for _, loc := range schedule.Locations {
		if loc.IdWithTitle.Id == int(LocationDeVereenigingId) {
			return fmt.Errorf("cannot enrich schedule due to conflichting Location ID: %d, %v", loc.IdWithTitle.Id, loc)
		}
	}
	theLoc := VierdaagseLocation{
		IdWithTitle: IdWithTitle{
			Id:    int(LocationDeVereenigingId),
			Title: "De Vereeniging",
		},
		// TODO?
	}

	schedule.Locations = append(schedule.Locations, theLoc)
	programs := []VierdaagseProgram{
		// Dag 1

		// Dag 2
		createProgram(schedule, "Restaurant en terras geopend", createEventTime(2, 10, 0), createEventTime(2, 23, 0), theLoc, ""),
		createProgram(schedule, "Speciaalbierplein", createEventTime(2, 14, 0), createEventTime(2, 20, 0), theLoc, ""),
		createProgram(schedule, "Optreden van DJ Bertil", createEventTime(2, 13, 0), createEventTime(2, 18, 0), theLoc, ""),

		// Dag 3
		createProgram(schedule, "Restaurant en terras geopend", createEventTime(3, 10, 0), createEventTime(3, 23, 0), theLoc, ""),
		createProgram(schedule, "Speciaalbierplein", createEventTime(3, 14, 0), createEventTime(3, 20, 0), theLoc, ""),
		createProgram(schedule, "Optreden van DJ Bertil", createEventTime(3, 13, 0), createEventTime(3, 18, 0), theLoc, ""),

		// Dag 4
		createProgram(schedule, "Restaurant en terras geopend", createEventTime(4, 10, 0), createEventTime(4, 23, 0), theLoc, ""),
		createProgram(schedule, "Speciaalbierplein", createEventTime(4, 14, 0), createEventTime(4, 20, 0), theLoc, ""),
		createProgram(schedule, "Optreden van DJ Bertil", createEventTime(4, 13, 0), createEventTime(4, 18, 0), theLoc, ""),

		// Dag 5
		createProgram(schedule, "Restaurant en terras geopend", createEventTime(5, 10, 0), createEventTime(5, 23, 0), theLoc, ""),
		createProgram(schedule, "Speciaalbierplein", createEventTime(5, 14, 0), createEventTime(5, 20, 0), theLoc, ""),
		createProgram(schedule, "Optreden van DJ Bertil", createEventTime(5, 13, 0), createEventTime(5, 18, 0), theLoc, ""),

		// Dag 6
		createProgram(schedule, "Restaurant en terras geopend", createEventTime(6, 9, 0), createEventTime(6, 23, 0), theLoc, ""),
		createProgram(schedule, "Speciaalbierplein", createEventTime(6, 14, 0), createEventTime(6, 21, 0), theLoc, ""),
		createProgram(schedule, "Optreden van DJ Danny", createEventTime(6, 14, 0), createEventTime(6, 19, 0), theLoc, ""),

		// Dag 7
		createProgram(schedule, "Terras geopend", createEventTime(7, 8, 0), createEventTime(7, 23, 0), theLoc, ""),
		createProgram(schedule, "Restaurant en terras geopend", createEventTime(7, 9, 0), createEventTime(7, 23, 0), theLoc, ""),
		createProgram(schedule, "Speciaalbierplein", createEventTime(7, 14, 0), createEventTime(7, 21, 0), theLoc, ""),
		createProgram(schedule, "Optreden van DJ Danny", createEventTime(7, 16, 0), createEventTime(7, 23, 0), theLoc, ""),
	}

	currentProgramIds := make(map[int]struct{})
	for _, currentProgram := range schedule.Programs {
		if _, ok := currentProgramIds[currentProgram.IdWithTitle.Id]; !ok {
			currentProgramIds[currentProgram.IdWithTitle.Id] = struct{}{}
		}
	}
	for _, program := range programs {
		if _, ok := currentProgramIds[program.IdWithTitle.Id]; ok {
			slog.Error("cannot add Vereeniging program due to conflicting ID", "id", program.IdWithTitle.Id, "program", program)
			continue
		}
		slog.Info("adding program from Vereeniging", "program", program)
		schedule.Programs = append(schedule.Programs, program)
	}
	return nil
}

// vim: cc=120:
