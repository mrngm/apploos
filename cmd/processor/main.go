package main

import (
	"context"
	"encoding/json"
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
	eventDir   = flag.String("eventDir", "", "Specifies the directory to import events from (JSON)")
	prod       = flag.Bool("prod", false, "When given, don't show the TESTING banner")
	storage    = flag.String("storage", "", "Scan this directory for collecting Vierdaagse JSON files")
	pattern    = flag.String("pattern", "*.blob", "Only consider these files to be actual data files, see path.Match")
	out        = flag.String("out", "-", "Write to this file, or - for standard output")
	outDir     = flag.String("outDir", "", "Write to this directory, or use current working directory. This automatically writes the stylesheet as style.css, and scripts as scripts.js.")
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

func readEventFile(fn string) (EventData, error) {
	data := EventData{}
	eventContents, err := os.ReadFile(fn)
	if err != nil {
		slog.Error("cannot read event JSON file", "err", err, "fn", fn)
		return data, err
	}

	err = json.Unmarshal(eventContents, &data)
	if err != nil {
		slog.Error("cannot unmarshal JSON", "err", err)
		return data, err
	}
	return data, nil
}

func readEventDir() ([]EventData, error) {
	eventData := make([]EventData, 0)
	if *eventDir == "" {
		return eventData, nil
	}

	_, err := os.Stat(*eventDir)
	if err != nil {
		slog.Error("could not stat event dir", "err", err, "dir", *eventDir)
		return eventData, err
	}
	entries, err := os.ReadDir(*eventDir)
	if err != nil {
		slog.Error("could not read event dir", "err", err, "dir", *eventDir)
		return eventData, err
	}
	patternMatched := make([]os.DirEntry, 0)
	pattern := "*.json"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matched, err := path.Match(pattern, entry.Name())
		if err != nil {
			slog.Error("matching failed", "err", err, "pattern", pattern, "entry", entry.Name())
			continue
		}
		if !matched {
			continue
		}
		patternMatched = append(patternMatched, entry)
	}
	if len(patternMatched) == 0 {
		slog.Info("no matches found", "dir", *eventDir, "pattern", pattern)
		return eventData, fmt.Errorf("no matches")
	}

	for _, eventFile := range patternMatched {
		fn := filepath.Join(*eventDir, eventFile.Name())
		data, err := readEventFile(fn)
		if err != nil {
			slog.Warn("could not read event file, skipping", "err", err, "dir", *eventDir, "fn", fn)
			continue
		}
		eventData = append(eventData, data)
	}
	return eventData, nil
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

	if *eventDir != "" {
		events, err := readEventDir()
		if err == nil {
			for _, event := range events {
				if err := EnrichGenericEvent(&everything, event); err != nil {
					slog.Error("could not enrich schedule with event", "event", event, "err", err)
				}
			}
		} else {
			slog.Error("could not read eventDir", "eventDir", *eventDir, "err", err)
		}
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

	written, err = util.SaveToDisk(context.TODO(), *outDir, "style.css", stylesheetCSS, *cleanupTmp, true)
	if err != nil {
		slog.Error("failed saving to disk", "err", err)
	}
	slog.Debug("SaveToDisk returns", "written", written, "err", err)

	written, err = util.SaveToDisk(context.TODO(), *outDir, "scripts.js", scriptingJS, *cleanupTmp, true)
	if err != nil {
		slog.Error("failed saving to disk", "err", err)
	}
	slog.Debug("SaveToDisk returns", "written", written, "err", err)
}

// vim: cc=120:
