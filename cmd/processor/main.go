package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/mrngm/apploos/util"
)

var (
	jsonFile   = flag.String("json", "", "Specifies the filename to read in Vierdaagse JSON format")
	icalFile   = flag.String("ical", "", "Specifies the filename to read in Thiemeloods iCal XML format")
	prod       = flag.Bool("prod", false, "When given, don't show the TESTING banner")
	storage    = flag.String("storage", "", "Scan this directory for collecting Vierdaagse JSON files")
	pattern    = flag.String("pattern", "*.blob", "Only consider these files to be actual data files, see path.Match")
	out        = flag.String("out", "-", "Write to this file, or - for standard output")
	outDir     = flag.String("outDir", "", "Write to this directory, or use current working directory")
	cleanupTmp = flag.Bool("cleanTmp", false, "Cleanup temporary files after either a successful or unsuccessful write")
)

func readJsonFile(fn string) (VierdaagseOverview, error) {
	ret := VierdaagseOverview{}
	jsonContents, err := os.ReadFile(fn)
	if err != nil {
		slog.Error("cannot read JSON file", "err", err, "fn", fn)
		return ret, err
	}

	err = json.Unmarshal(jsonContents, &ret)
	if err != nil {
		slog.Error("cannot unmarshal JSON", "err", err)
		return ret, err
	}
	return ret, nil
}

func readICalFile(fn string) (ICalendar, error) {
	calendar := ICalendar{}
	icalContents, err := os.ReadFile(fn)
	if err != nil {
		slog.Error("cannot read iCal XML file", "err", err, "fn", fn)
		return calendar, err
	}

	err = xml.Unmarshal(icalContents, &calendar)
	if err != nil {
		slog.Error("cannot unmarshal XML", "err", err)
		return calendar, err
	}
	for i, event := range calendar.Events {
		if loc, err := time.LoadLocation(event.StartTimeTZ); err == nil {
			if startTime, err := time.ParseInLocation("2006-01-02T15:04:05", event.StartTime, loc); err == nil {
				calendar.Events[i].FullStartTime = startTime
			} else {
				//slog.Error("parsing starttime failed (ignoring event)", "err", err, "event", event)
				continue
			}
		}
		if loc, err := time.LoadLocation(event.EndTimeTZ); err == nil {
			if EndTime, err := time.ParseInLocation("2006-01-02T15:04:05", event.EndTime, loc); err == nil {
				calendar.Events[i].FullEndTime = EndTime
			} else {
				//slog.Error("parsing endtime failed, assuming endtime as startime + 1h", "err", err, "event", event)
				calendar.Events[i].FullEndTime = calendar.Events[i].FullStartTime.Add(1 * time.Hour)
			}
		}
	}
	return calendar, nil
}

func readStorageDir() (dirModTime time.Time, fileModTime time.Time, recentFile string, err error) {
	dirStat, err := os.Stat(*storage)
	if err != nil {
		slog.Error("could not stat storage dir", "err", err, "dir", *storage)
		return time.Time{}, time.Time{}, "", err
	}
	entries, err := os.ReadDir(*storage)
	if err != nil {
		slog.Error("could not read storage dir", "err", err, "dir", *storage)
		return time.Time{}, time.Time{}, "", err
	}
	patternMatched := make([]os.DirEntry, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matched, err := path.Match(*pattern, entry.Name())
		if err != nil {
			slog.Error("matching failed", "err", err, "pattern", *pattern, "entry", entry.Name())
			continue
		}
		if !matched {
			continue
		}
		patternMatched = append(patternMatched, entry)
	}
	if len(patternMatched) == 0 {
		slog.Info("no matches found", "dir", *storage, "pattern", *pattern)
		return time.Time{}, time.Time{}, "", fmt.Errorf("no matches")
	}

	slices.SortFunc(patternMatched, func(a, b os.DirEntry) int {
		infoA, errA := a.Info()
		infoB, errB := b.Info()
		if errA != nil || errB != nil {
			slog.Debug("sorting direntry failed", "errA", errA, "errB", errB, "entryA", a, "entryB", b)
			return 0
		}
		return infoA.ModTime().Compare(infoB.ModTime())
	})

	lastMatch := patternMatched[len(patternMatched)-1]
	fnInfo, err := lastMatch.Info()
	if err != nil {
		slog.Error("could not request information from last match", "err", err, "lastMatch", lastMatch)
		return time.Time{}, time.Time{}, "", fmt.Errorf("internal error")
	}
	return dirStat.ModTime(), fnInfo.ModTime(), filepath.Join(*storage, lastMatch.Name()), nil
}

func main() {
	flag.Parse()

	if len(*jsonFile) > 0 && len(*storage) > 0 {
		slog.Error("Please provide either -json or -storage")
		os.Exit(1)
	}

	if *outDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			slog.Error("could not get working directory", "err", err)
			os.Exit(1)
		}
		*outDir = cwd
	}

	everything := VierdaagseOverview{}
	if len(*jsonFile) > 0 {
		try, err := readJsonFile(*jsonFile)
		if err != nil {
			os.Exit(1)
		}
		everything = try
	} else if len(*storage) > 0 && len(*pattern) > 0 {
		// Automatically read *storage, only looking for files matching *pattern, returning the *storage modification
		// time, the most recent filename, and errors should they occur
		dirModTime, fileModTime, fn, err := readStorageDir()
		if err != nil {
			os.Exit(1)
		}
		slog.Info("Read storage dir", "dirModTime", dirModTime, "fn", fn, "fileModTime", fileModTime)
		try, err := readJsonFile(fn)
		if err != nil {
			os.Exit(1)
		}
		try.DirModTime = dirModTime
		try.FileModTime = fileModTime
		everything = try
	}

	if len(*icalFile) > 0 {
		calendar, err := readICalFile(*icalFile)
		if err != nil {
			os.Exit(1)
		}
		if err := EnrichScheduleWithThiemeloods(&everything, calendar); err != nil {
			slog.Error("could not enrich schedule with Thiemeloods", "err", err)
		}
	}

	if err := EnrichScheduleWithOpstand(&everything); err != nil {
		slog.Error("could not enrich schedule with Opstand", "err", err)
	}

	if err := EnrichScheduleWithOnderbroek(&everything); err != nil {
		slog.Error("could not enrich schedule with Onderbroek", "err", err)
	}

	if err := EnrichScheduleWithDollars(&everything); err != nil {
		slog.Error("could not enrich schedule with Dollars", "err", err)
	}

	if err := EnrichScheduleWithVereeniging(&everything); err != nil {
		slog.Error("could not enrich schedule with Vereeniging", "err", err)
	}

	output, err := RenderSchedule(everything)
	if err != nil {
		slog.Error("error rendering schedule", "err", err)
		os.Exit(1)
	}
	if *out == "-" {
		// Write to stdout
		fmt.Fprint(os.Stdout, string(output))
		return
	}

	written, err := util.SaveToDisk(context.TODO(), *outDir, *out, output, *cleanupTmp, true)
	if err != nil {
		slog.Error("failed saving to disk", "err", err)
	}
	slog.Debug("SaveToDisk returns", "written", written, "err", err)
}

// vim: cc=120:
