package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
)

var (
	jsonFile = flag.String("json", "all.json", "Specifies the filename to read in JSON format")
	prod     = flag.Bool("prod", false, "When given, don't show the TESTING banner")
)

func main() {
	flag.Parse()

	jsonContents, err := os.ReadFile(*jsonFile)
	if err != nil {
		slog.Error("cannot read JSON file", "err", err, "fn", *jsonFile)
		os.Exit(1)
	}

	everything := VierdaagseOverview{}
	err = json.Unmarshal(jsonContents, &everything)
	if err != nil {
		slog.Error("cannot unmarshal JSON", "err", err)
		os.Exit(1)
	}
	//	slog.Info("everything unpacked", "data", everything)
	RenderSchedule(everything)
}
