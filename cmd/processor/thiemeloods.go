package main

import (
	"time"
)

type ICalendar struct {
	Events []ICalendarEvent `xml:"vcalendar>components>vevent"`
}

type ICalendarEvent struct {
	//DateTime time.Time `xml:"dtstamp>date-time"`
	Summary     string `xml:"properties>summary>text"`
	Description string `xml:"properties>description>text"`
	StartTime   string `xml:"properties>dtstart>date-time"`
	StartTimeTZ string `xml:"properties>dtstart>parameters>tzid>text"`
	EndTime     string `xml:"properties>dtend>date-time"`
	EndTimeTZ   string `xml:"properties>dtend>parameters>tzid>text"`
	URL         string `xml:"properties>url>uri"`
	CostType    string `xml:"properties>x-cost-type>unknown"`
	TicketPrice string `xml:"properties>x-cost>unknown"`
	TicketURL   string `xml:"properties>x-tickets-url>unknown"`

	FullStartTime time.Time
	FullEndTime   time.Time
}

func FilterThiemeloodsForVierdaagse(calendar ICalendar, startTime, endTime time.Time) []ICalendarEvent {
	ret := make([]ICalendarEvent, 0)
	for _, event := range calendar.Events {
		if event.FullStartTime.Before(startTime) {
			continue
		}
		if event.FullStartTime.After(endTime) {
			continue
		}
		ret = append(ret, event)
	}
	return ret
}
