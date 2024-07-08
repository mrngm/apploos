package main

import (
	//"bytes"
	"fmt"
	"log/slog"
)

type Location struct {
	Id       int
	Title    string
	Children []*Location
}

func (l *Location) String() string {
	return fmt.Sprintf("[%d] %s nChildren: %d", l.Id, l.Title, len(l.Children))
}

func RenderSchedule(everything VierdaagseOverview) ([]byte, error) {
	//buf := new(bytes.Buffer)

	// Valkhof parent is
	locations := make(map[int]*Location)
	for _, loc := range everything.Locations {
		if _, ok := locations[loc.Id]; !ok {
			locations[loc.Id] = &Location{
				Children: make([]*Location, 0),
			}
		}
		theLoc := locations[loc.Id]
		theLoc.Id = loc.Id
		theLoc.Title = loc.Title
		if loc.Parent > 0 {
			// Sub locations, fill in separate loop
			continue
		}
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
			theParentLoc.Children = append(theParentLoc.Children, theChildLoc)
		}
	}

	for _, theLoc := range locations {
		if len(theLoc.Children) > 0 {
			slog.Info("parent location", "title", theLoc.Title)
			for _, loc := range theLoc.Children {
				slog.Info("child location", "title", loc.Title)
				if len(loc.Children) > 0 {
					slog.Error("child location has more locations", "len", len(loc.Children))
				}
			}
		}
	}
	return nil, nil
}
