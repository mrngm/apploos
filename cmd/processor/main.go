package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"log/slog"
	"os"
	"time"
)

var (
	jsonFile = flag.String("json", "", "Specifies the filename to read in Vierdaagse JSON format")
	icalFile = flag.String("ical", "", "Specifies the filename to read in Thiemeloods iCal XML format")
	prod     = flag.Bool("prod", false, "When given, don't show the TESTING banner")
)

func main() {
	flag.Parse()

	everything := VierdaagseOverview{}
	if len(*jsonFile) > 0 {
		jsonContents, err := os.ReadFile(*jsonFile)
		if err != nil {
			slog.Error("cannot read JSON file", "err", err, "fn", *jsonFile)
			os.Exit(1)
		}

		err = json.Unmarshal(jsonContents, &everything)
		if err != nil {
			slog.Error("cannot unmarshal JSON", "err", err)
			os.Exit(1)
		}
		//	slog.Info("everything unpacked", "data", everything)
	}
	if len(*icalFile) > 0 {
		icalContents, err := os.ReadFile(*icalFile)
		if err != nil {
			slog.Error("cannot read iCal XML file", "err", err, "fn", *icalFile)
			os.Exit(1)
		}

		calendar := ICalendar{}
		err = xml.Unmarshal(icalContents, &calendar)
		if err != nil {
			slog.Error("cannot unmarshal XML", "err", err)
			os.Exit(1)
		}
		for i, event := range calendar.Events {
			if loc, err := time.LoadLocation(event.StartTimeTZ); err == nil {
				if startTime, err := time.ParseInLocation("2006-01-02T15:04:05", event.StartTime, loc); err == nil {
					calendar.Events[i].FullStartTime = startTime
				} else {
					slog.Error("parsing starttime failed", "err", err)
				}
			}
			if loc, err := time.LoadLocation(event.EndTimeTZ); err == nil {
				if EndTime, err := time.ParseInLocation("2006-01-02T15:04:05", event.EndTime, loc); err == nil {
					calendar.Events[i].FullEndTime = EndTime
				} else {
					slog.Error("parsing endtime failed", "err", err)
				}
			}
		}
		for _, event := range calendar.Events {
			slog.Info("event", "summary", event.Summary, "start", event.FullStartTime, "end", event.FullEndTime)
		}
	}

	RenderSchedule(everything)
}
